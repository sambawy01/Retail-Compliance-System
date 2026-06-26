"""Detector protocol and Detection dataclass.

A Detector takes a frame + timestamp and returns a list of Detection
records. Each implementation wraps a specific model (YOLO for person/object,
ArcFace for face embeddings).

The pipeline depends only on the Detector protocol so it can be tested with
stub implementations and the real models can be swapped freely.
"""
from __future__ import annotations

from dataclasses import dataclass
from typing import Optional, Protocol

import numpy as np


@dataclass
class Detection:
    """One detected object/person/face in one frame."""

    label: str
    confidence: float
    bbox: tuple[float, float, float, float]  # x1, y1, x2, y2 in pixels
    keypoints: Optional[list[tuple[float, float, float]]] = None  # (x, y, vis)
    track_id: Optional[int] = None


class Detector(Protocol):
    def detect(self, frame: np.ndarray, frame_ts: float) -> list[Detection]:
        ...