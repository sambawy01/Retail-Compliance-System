"""Per-camera WebRTC publisher.

Listens on a SignalingClient for viewer-join events from the cloud hub.
On each one, builds a fresh RTCPeerConnection whose video track is backed by
the camera's RTSP stream via aiortc.contrib.media.MediaPlayer, creates an
SDP offer, and sends it back through signaling.

aiortc is imported lazily so the rest of the edge agent (detection pipeline,
clip uploader) keeps running even when aiortc is missing — for example in CI
which uses the lightweight ci extras set.
"""
from __future__ import annotations

import asyncio
from typing import Any, Optional

from .logging_setup import get_logger
from .signaling_client import SignalingClient

log = get_logger(__name__)


def _import_aiortc() -> Optional[Any]:
    """Lazy aiortc import; returns the module or None if unavailable."""
    try:
        import aiortc  # noqa: WPS433
        from aiortc.contrib import media as _media  # noqa: F401, WPS433

        return aiortc
    except Exception as e:  # noqa: BLE001
        log.warning("aiortc_unavailable", error=str(e))
        return None


class CameraPublisher:
    """Owns one RTSP source + one RTCPeerConnection per active viewer.

    For the single-edge-PC model we treat each new viewer-join as a fresh
    session and re-create the PC. A real multi-viewer setup would use an SFU.
    """

    def __init__(
        self,
        camera_id: str,
        rtsp_url: str,
        signaling: SignalingClient,
        turn_url: Optional[str] = None,
        turn_user: Optional[str] = None,
        turn_pass: Optional[str] = None,
    ):
        self._camera_id = camera_id
        self._rtsp = rtsp_url
        self._signaling = signaling
        self._turn = turn_url
        self._user = turn_user
        self._pass = turn_pass
        self._pc: Any = None
        self._aiortc = _import_aiortc()

    def _ice_servers(self) -> list:
        if self._aiortc is None:
            return []
        from aiortc import RTCIceServer  # type: ignore[import]

        servers = [RTCIceServer(urls=["stun:stun.l.google.com:19302"])]
        if self._turn:
            servers.append(
                RTCIceServer(
                    urls=[self._turn],
                    username=self._user,
                    credential=self._pass,
                )
            )
        return servers

    async def run(self) -> None:
        """Main loop. Exits when the signaling connection closes."""
        if self._aiortc is None:
            log.warning(
                "webrtc_publisher_disabled",
                camera_id=self._camera_id,
                reason="aiortc not installed",
            )
            return

        try:
            await self._signaling.connect()
        except Exception as e:  # noqa: BLE001
            log.warning(
                "signaling_connect_failed",
                camera_id=self._camera_id,
                error=str(e),
            )
            return

        try:
            while True:
                msg = await self._signaling.recv()
                if msg is None:
                    break
                mtype = msg.get("type", "")
                if mtype == "viewer-join":
                    await self._handle_viewer_join()
                elif mtype == "sdp-answer":
                    await self._handle_answer(msg.get("payload", {}))
                elif mtype == "ice":
                    pass  # aiortc handles trickle ICE inside SDP
                else:
                    log.debug(
                        "signaling_unknown_msg",
                        type=mtype,
                        camera_id=self._camera_id,
                    )
        finally:
            await self._close_pc()
            await self._signaling.close()

    async def _handle_viewer_join(self) -> None:
        await self._close_pc()
        from aiortc import (  # type: ignore[import]
            RTCPeerConnection,
            RTCConfiguration,
        )
        from aiortc.contrib.media import MediaPlayer  # type: ignore[import]

        self._pc = RTCPeerConnection(
            RTCConfiguration(iceServers=self._ice_servers())
        )

        try:
            player = MediaPlayer(self._rtsp, options={"rtsp_transport": "tcp"})
        except Exception as e:  # noqa: BLE001
            log.warning("rtsp_open_failed", camera_id=self._camera_id, error=str(e))
            return
        if player.video is not None:
            self._pc.addTrack(player.video)

        offer = await self._pc.createOffer()
        await self._pc.setLocalDescription(offer)
        await self._signaling.send(
            {
                "type": "sdp-offer",
                "payload": {"sdp": self._pc.localDescription.sdp},
            }
        )
        log.info("offer_sent", camera_id=self._camera_id)

    async def _handle_answer(self, payload: dict) -> None:
        if self._pc is None:
            return
        sdp = payload.get("sdp", "")
        if not sdp:
            return
        from aiortc import RTCSessionDescription  # type: ignore[import]

        try:
            await self._pc.setRemoteDescription(
                RTCSessionDescription(sdp=sdp, type="answer")
            )
            log.info("answer_received", camera_id=self._camera_id)
        except Exception as e:  # noqa: BLE001
            log.warning("set_remote_desc_failed", camera_id=self._camera_id, error=str(e))

    async def _close_pc(self) -> None:
        if self._pc is None:
            return
        try:
            await self._pc.close()
        except Exception:  # noqa: BLE001
            pass
        self._pc = None


async def run_publishers(cfg, cameras) -> None:
    """Spin up one CameraPublisher task per camera and run them concurrently.

    Returns when all tasks complete (typically: never, until the process
    receives a shutdown signal). Exceptions in individual publishers are
    captured by asyncio.gather(return_exceptions=True) so one failing
    camera does not take the whole agent down.
    """
    if not cfg.signaling_ws_url:
        log.info("webrtc_disabled", reason="signaling_ws_url not configured")
        return

    tasks = []
    for cam in cameras:
        sig = SignalingClient(cfg.signaling_ws_url, cfg.api_token, cam.camera_id)
        pub = CameraPublisher(
            camera_id=cam.camera_id,
            rtsp_url=cam.rtsp_url,
            signaling=sig,
            turn_url=cfg.turn_url,
            turn_user=cfg.turn_user,
            turn_pass=cfg.turn_pass,
        )
        tasks.append(asyncio.create_task(pub.run()))
    if not tasks:
        return
    await asyncio.gather(*tasks, return_exceptions=True)