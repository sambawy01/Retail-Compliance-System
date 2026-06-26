"""ArcFace face detector + embedder.

Wraps insightface.app.FaceAnalysis to emit 512-d face embeddings (the same
shape as identity_templates.embedding in the cloud database).

Like person_yolo.py, the heavy import (insightface) is lazy: it happens
inside __init__, never at module import time. Test code can import this
module and instantiate stubs without needing the real insightface wheel.
"""
from __future__ import annotations

from typing import TYPE_CHECKING

import numpy as np

from . import Detection

if TYPE_CHECKING:
    pass  # insightface imported lazily


class ArcFaceDetector:
    """Extracts 512-d face embeddings from each detected face.

    Each returned Detection has:
        label: "face"
        confidence: insightface det_score (0..1)
        bbox: (x1, y1, x2, y2) in pixels
        keypoints: a length-512 list of (embed_value, 0.0, 0.0) tuples — the
            embedding is packed into the existing keypoint channel to avoid
            broadening the Detection schema. embedding_of() below unpacks it.
    """

    def __init__(
        self,
        model_name: str = "buffalo_l",
        det_size: tuple[int, int] = (640, 640),
    ):
        from insightface.app import FaceAnalysis  # lazy import

        self._app = FaceAnalysis(name=model_name)
        self._app.prepare(ctx_id=0, det_size=det_size)

    def detect(self, frame: np.ndarray, frame_ts: float) -> list[Detection]:
        del frame_ts  # unused — insightface does not need it
        faces = self._app.get(frame)
        out: list[Detection] = []
        for f in faces:
            x1, y1, x2, y2 = map(float, f.bbox)
            d = Detection(
                label="face",
                confidence=float(getattr(f, "det_score", 0.0)),
                bbox=(x1, y1, x2, y2),
            )
            # Pack the 512-d embedding into keypoints to avoid expanding
            # the Detection dataclass. embedding_of() below unpacks it.
            emb = getattr(f, "normed_embedding", None)
            if emb is not None:
                d.keypoints = [(float(v), 0.0, 0.0) for v in emb]
            out.append(d)
        return out


def embedding_of(detection: Detection) -> list[float] | None:
    """Extract the embedding floats from a face Detection produced by
    ArcFaceDetector.

    Returns None if the detection has no embedding attached (e.g. it was a
    different detector's output, or insightface failed to compute one).
    """
    if detection.keypoints is None:
        return None
    return [kp[0] for kp in detection.keypoints]