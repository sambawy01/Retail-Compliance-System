"""YOLO person detector.

Wraps ultralytics.YOLO to emit only 'person' detections (COCO class 0).
Real weights are loaded from disk at construction time; the model itself
runs at ~30+ fps on a modest discrete GPU.
"""
from __future__ import annotations

from typing import TYPE_CHECKING

import numpy as np

from . import Detection

if TYPE_CHECKING:
    pass  # ultralytics imported lazily


class PersonYOLO:
    def __init__(self, weights: str = "yolov8n.pt", min_confidence: float = 0.5):
        from ultralytics import YOLO  # lazy import — only needed when used

        self._model = YOLO(weights)
        self._thresh = min_confidence

    def detect(self, frame: np.ndarray, frame_ts: float) -> list[Detection]:
        results = self._model.predict(
            frame, classes=[0], conf=self._thresh, verbose=False
        )
        out: list[Detection] = []
        for r in results:
            if r.boxes is None:
                continue
            for box in r.boxes:
                conf = float(box.conf[0])
                x1, y1, x2, y2 = map(float, box.xyxy[0])
                out.append(
                    Detection(
                        label="person",
                        confidence=conf,
                        bbox=(x1, y1, x2, y2),
                    )
                )
        return out