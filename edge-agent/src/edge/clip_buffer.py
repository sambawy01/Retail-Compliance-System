"""Rolling frame buffer used to extract event clips on demand.

Each camera owns one RollingClipBuffer. Frames are pushed continuously by
the RTSP thread; when an event fires, the pipeline asks for the frames
in a window around the event timestamp and hands them to the encoder/
uploader.
"""
from __future__ import annotations

from collections import deque
from typing import Deque, Tuple

import numpy as np

Frame = Tuple[float, np.ndarray]


class RollingClipBuffer:
    def __init__(self, window_seconds: int, fps: int):
        self._window = window_seconds
        self._fps = fps
        # Double the expected count to absorb bursts above target fps.
        self._frames: Deque[Frame] = deque(maxlen=window_seconds * fps * 2)

    def push(self, frame: np.ndarray, ts: float) -> None:
        self._frames.append((ts, frame))
        cutoff = ts - self._window
        while self._frames and self._frames[0][0] < cutoff:
            self._frames.popleft()

    def snapshot(self) -> list[Frame]:
        return list(self._frames)

    def extract_around(
        self, event_ts: float, before: float, after: float
    ) -> list[Frame]:
        lo, hi = event_ts - before, event_ts + after
        return [(t, f) for (t, f) in self._frames if lo <= t <= hi]

    def __len__(self) -> int:
        return len(self._frames)