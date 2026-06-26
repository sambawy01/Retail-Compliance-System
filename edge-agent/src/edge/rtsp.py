"""OpenCV-backed RTSP frame source with auto-reconnect.

Yields decoded BGR frames at a target FPS. On stream loss, retries after a
short backoff. Uses OpenCV VideoCapture (lazy import inside ``stream()``).
"""
from __future__ import annotations

import time
from typing import Iterator, Tuple

import numpy as np

from .logging_setup import get_logger

log = get_logger(__name__)


class RTSPSource:
    """RTSP frame source reading at ``target_fps`` via OpenCV."""

    def __init__(self, rtsp_url: str, target_fps: int = 10):
        self._url = rtsp_url
        self._target_fps = target_fps
        self._frame_interval = 1.0 / target_fps

    def stream(self) -> Iterator[Tuple[np.ndarray, float]]:
        import cv2  # lazy import — only needed when actually streaming

        while True:
            cap = cv2.VideoCapture(self._url)
            if not cap.isOpened():
                log.warning("rtsp_reconnect", url=self._url, error="cannot open")
                time.sleep(2)
                continue

            last_emit = 0.0
            while True:
                ok, frame = cap.read()
                if not ok or frame is None:
                    break
                now = time.monotonic()
                if now - last_emit < self._frame_interval:
                    continue
                last_emit = now
                yield frame, now

            cap.release()
            log.warning("rtsp_reconnect", url=self._url, error="stream ended")
            time.sleep(2)