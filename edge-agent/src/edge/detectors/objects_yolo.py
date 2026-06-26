"""Object detector for retail compliance classes.

Wraps a YOLO model for retail-specific objects: phone, cup, bag, etc. The
weights file is supplied at construction; the default class map below is the
expected output of the custom-trained model. Falls back to COCO classes
(yolov8n.pt) if no custom weights are available.
"""
from __future__ import annotations

import numpy as np

from . import Detection

# Retail-specific object classes (custom-trained model)
OBJECT_CLASSES: dict[int, str] = {
    0: "phone",
    1: "cup",
    2: "bag",
    3: "cart",
    4: "basket",
    5: "bottle",
    6: "spill",
    7: "mess",
}

# COCO class names for fallback (subset relevant to retail)
COCO_CLASSES: dict[int, str] = {
    67: "phone",   # cell phone
    41: "cup",
    24: "bag",      # backpack
    25: "bag",      # umbrella (often detected alongside)
    26: "bag",      # handbag
    28: "bag",      # suitcase
    39: "bottle",
    40: "bottle",
    73: "book",
}


class ObjectYOLO:
    def __init__(self, weights: str = "yolov8n.pt", min_confidence: float = 0.4):
        from ultralytics import YOLO  # lazy

        self._model = YOLO(weights)
        self._thresh = min_confidence
        self._is_coco = "coco" in weights.lower() or weights == "yolov8n.pt"
        self._class_map = COCO_CLASSES if self._is_coco else OBJECT_CLASSES

    def detect(self, frame: np.ndarray, frame_ts: float) -> list[Detection]:
        results = self._model.predict(frame, conf=self._thresh, verbose=False)
        out: list[Detection] = []
        for r in results:
            if r.boxes is None:
                continue
            for box in r.boxes:
                cls = int(box.cls[0])
                label = self._class_map.get(cls, f"class_{cls}")
                conf = float(box.conf[0])
                x1, y1, x2, y2 = map(float, box.xyxy[0])
                out.append(
                    Detection(
                        label=label,
                        confidence=conf,
                        bbox=(x1, y1, x2, y2),
                    )
                )
        return out