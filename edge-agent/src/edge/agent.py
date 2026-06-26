"""Edge agent entry point.

Wires the config → per-camera RTSP source + pipeline → heartbeat → cloud API
→ WebRTC publishers. Run as `python -m edge.agent /path/to/config.yaml`.
The Docker entrypoint (see Dockerfile) does exactly that with the
bind-mounted config.
"""
from __future__ import annotations

import asyncio
import signal
import sys
import threading

from .api_client import WatchDogClient
from .clip_buffer import RollingClipBuffer
from .config import EdgeConfig
from .heartbeat import HeartbeatLoop
from .logging_setup import configure_logging, get_logger
from .pipeline import CameraPipeline
from .rtsp import RTSPSource

log = get_logger(__name__)


def run(config_path: str) -> int:
    configure_logging()
    cfg = EdgeConfig.load(config_path)
    log.info("agent_start", agent_id=cfg.agent_id, cameras=len(cfg.cameras))

    client = WatchDogClient(cfg.api_url, cfg.api_token)

    # Heavy model loading is intentionally deferred to construction time, so
    # an operator without GPU still gets the agent to import + run config
    # validation. The actual detect() calls require a real GPU and weights.
    person_detector = _load_person_detector()
    object_detector = _load_object_detector()
    face_detector = _load_face_detector()

    stop_event = threading.Event()
    threads: list[threading.Thread] = []

    def on_detection(ev: dict) -> None:
        try:
            client.post_detection(ev)
        except Exception as e:  # noqa: BLE001
            log.error("post_detection_failed", error=str(e))

    for cam in cfg.cameras:
        try:
            cam_meta = client.get_camera(cam.camera_id)
            zones = client.list_zones(cam.camera_id)
            location_id = cam_meta.get("location_id", "")
        except Exception as e:  # noqa: BLE001
            log.warning("bootstrap_failed", camera_id=cam.camera_id, error=str(e))
            location_id = ""
            zones = []

        buf = RollingClipBuffer(window_seconds=cfg.clip_buffer_seconds, fps=10)
        pipe = CameraPipeline(
            camera=cam,
            location_id=location_id,
            person_detector=person_detector,
            object_detector=object_detector,
            face_detector=face_detector,
            on_detection=on_detection,
            zones=zones,
            api_client=client,
        )
        src = RTSPSource(cam.rtsp_url, target_fps=10)
        t = threading.Thread(
            target=_camera_loop,
            args=(src, pipe, buf, stop_event),
            daemon=True,
            name=f"camera-{cam.camera_id}",
        )
        threads.append(t)

    hb = HeartbeatLoop(
        client, [c.camera_id for c in cfg.cameras], cfg.heartbeat_seconds
    )
    hb.start()
    for t in threads:
        t.start()

    # WebRTC publishers run in their own daemon thread with an asyncio loop.
    # They are best-effort — if aiortc/websockets are not installed,
    # run_publishers() logs and returns. The detection pipeline is unaffected.
    if cfg.signaling_ws_url:
        threading.Thread(
            target=_run_publishers_thread,
            args=(cfg,),
            daemon=True,
            name="webrtc-publishers",
        ).start()

    def _shutdown(signum, _frame):
        log.info("agent_shutdown_signal", signal=signum)
        stop_event.set()

    signal.signal(signal.SIGTERM, _shutdown)
    signal.signal(signal.SIGINT, _shutdown)

    stop_event.wait()
    hb.stop()
    client.close()
    return 0


def _run_publishers_thread(cfg: EdgeConfig) -> None:
    """Thread target that owns the asyncio loop for WebRTC publishers."""
    try:
        # Lazy import — agent.py must stay importable even without aiortc.
        from .webrtc_publisher import run_publishers

        asyncio.run(run_publishers(cfg, cfg.cameras))
    except Exception as e:  # noqa: BLE001
        log.warning("webrtc_publishers_failed", error=str(e))


def _load_person_detector():
    """Lazy-load PersonYOLO. Returns a NullDetector if the ML lib is missing
    (so the agent can boot for config-validation runs without ultralytics
    installed). Production deployments must have the ML libs available.
    """
    try:
        from .detectors.person_yolo import PersonYOLO

        return PersonYOLO()
    except Exception as e:  # noqa: BLE001
        log.warning("person_detector_unavailable", error=str(e))
        return _NullDetector()


def _load_object_detector():
    try:
        from .detectors.objects_yolo import ObjectYOLO

        return ObjectYOLO(weights="weights/objects.pt")
    except Exception as e:  # noqa: BLE001
        log.warning("object_detector_unavailable", error=str(e))
        return _NullDetector()


def _load_face_detector():
    try:
        from .detectors.face_arcface import ArcFaceDetector

        return ArcFaceDetector()
    except Exception as e:  # noqa: BLE001
        log.warning("face_detector_unavailable", error=str(e))
        return None


class _NullDetector:
    """No-op detector returned when an ML library is missing. Logs once at
    construction; subsequent detect() calls return [].
    """

    def detect(self, frame, frame_ts):  # noqa: D401
        return []


def _camera_loop(
    src: RTSPSource,
    pipe: CameraPipeline,
    buf: RollingClipBuffer,
    stop_event: threading.Event,
) -> None:
    for frame, ts in src.stream():
        if stop_event.is_set():
            break
        buf.push(frame, ts)
        try:
            pipe.process_frame(frame, ts)
        except Exception as e:  # noqa: BLE001
            get_logger(__name__).error("frame_process_failed", error=str(e))


if __name__ == "__main__":
    sys.exit(run(sys.argv[1] if len(sys.argv) > 1 else "/etc/watchdog/config.yaml"))