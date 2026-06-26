"""WebSocket signaling client for the edge agent.

Connects to the Watch Dog backend's WebRTC signaling endpoint and exchanges
JSON messages with the cloud signaling hub.

The `websockets` library is imported lazily so the rest of the edge agent
(config validation, RTSP reading, detectors) keeps working even on hosts
where `websockets` failed to install.
"""
from __future__ import annotations

import asyncio
import json
from typing import Any, Optional

from .logging_setup import get_logger

log = get_logger(__name__)


class SignalingClient:
    """Thin WS client. One instance per camera.

    Connection is established lazily in connect() so construction is
    safe even when the agent runs without websockets installed.
    """

    def __init__(self, base_ws_url: str, token: str, camera_id: str):
        self._url = (
            f"{base_ws_url.rstrip('/')}"
            f"/api/v1/vision/webrtc/publisher/{camera_id}"
        )
        self._token = token
        self._camera_id = camera_id
        self._ws: Optional[Any] = None
        self._closed = asyncio.Event()

    async def connect(self) -> None:
        """Open the WS handshake. Raises RuntimeError if websockets is missing."""
        try:
            import websockets  # noqa: WPS433 (intentional lazy import)
        except ImportError as exc:
            raise RuntimeError(
                "websockets is not installed; WebRTC publisher disabled"
            ) from exc

        self._ws = await websockets.connect(
            self._url,
            additional_headers={"Authorization": f"Bearer {self._token}"},
            ping_interval=20,
            ping_timeout=60,
            max_size=2**20,  # 1 MiB — SDP can be a few KB, well under this
        )
        log.info("signaling_connected", camera_id=self._camera_id)

    async def send(self, msg: dict) -> None:
        if self._ws is None:
            return
        await self._ws.send(json.dumps(msg))

    async def recv(self) -> Optional[dict]:
        if self._ws is None:
            return None
        try:
            raw = await self._ws.recv()
        except Exception as e:  # noqa: BLE001
            log.warning(
                "signaling_recv_failed",
                error=str(e),
                camera_id=self._camera_id,
            )
            return None
        if isinstance(raw, (bytes, bytearray)):
            raw = raw.decode("utf-8", errors="replace")
        try:
            return json.loads(raw)
        except json.JSONDecodeError:
            log.warning(
                "signaling_bad_json",
                raw=str(raw)[:120],
                camera_id=self._camera_id,
            )
            return None

    async def close(self) -> None:
        if self._ws is None:
            return
        try:
            await self._ws.close()
        except Exception:  # noqa: BLE001
            pass
        self._ws = None
        self._closed.set()