"""Event builders for vision.safety.* subjects."""
from __future__ import annotations

from datetime import datetime, timezone

from . import SAFETY_SLIP_FALL


def _now_iso() -> str:
    return datetime.now(timezone.utc).isoformat()


def slip_fall(
    camera_id: str,
    location_id: str,
    zone_id: str,
    confidence: float,
    frame_url: str = "",
    clip_id: str | None = None,
) -> dict:
    return {
        "camera_id": camera_id,
        "location_id": location_id,
        "event_type": SAFETY_SLIP_FALL,
        "confidence": confidence,
        "payload": {
            "anomaly_type": "slip_fall",
            "zone_id": zone_id,
            "description": "Person detected on the ground",
            "confidence": confidence,
            "frame_url": frame_url,
            "clip_id": clip_id or "",
        },
        "clip_id": clip_id or "",
        "detected_at": _now_iso(),
    }