"""Event builders for operations-related subjects:
operations.checkout_bottleneck, inventory.stockroom_anomaly, camera.degraded.
"""
from __future__ import annotations

from datetime import datetime, timezone

from . import (
    CAMERA_DEGRADED,
    INVENTORY_STOCKROOM_ANOMALY,
    OPS_CHECKOUT_BOTTLENECK,
)


def _now_iso() -> str:
    return datetime.now(timezone.utc).isoformat()


def checkout_bottleneck(
    camera_id: str,
    location_id: str,
    zone_id: str,
    zone_name: str,
    person_count: int,
    threshold: int,
    severity: str,
    frame_url: str = "",
    clip_id: str | None = None,
) -> dict:
    return {
        "camera_id": camera_id,
        "location_id": location_id,
        "event_type": OPS_CHECKOUT_BOTTLENECK,
        "confidence": 1.0,
        "payload": {
            "zone_id": zone_id,
            "zone_name": zone_name,
            "person_count": person_count,
            "threshold": threshold,
            "severity": severity,
            "description": f"Checkout bottleneck: {person_count} > {threshold}",
            "frame_url": frame_url,
            "clip_id": clip_id or "",
        },
        "clip_id": clip_id or "",
        "detected_at": _now_iso(),
    }


def stockroom_anomaly(
    camera_id: str,
    location_id: str,
    zone_id: str,
    anomaly_type: str,
    confidence: float,
    description: str,
    frame_url: str = "",
    clip_id: str | None = None,
) -> dict:
    return {
        "camera_id": camera_id,
        "location_id": location_id,
        "event_type": INVENTORY_STOCKROOM_ANOMALY,
        "confidence": confidence,
        "payload": {
            "zone_id": zone_id,
            "anomaly_type": anomaly_type,
            "description": description,
            "confidence": confidence,
            "frame_url": frame_url,
            "clip_id": clip_id or "",
        },
        "clip_id": clip_id or "",
        "detected_at": _now_iso(),
    }


def camera_degraded(
    camera_id: str,
    location_id: str,
    reason: str,
    observed_fps: float | None = None,
) -> dict:
    return {
        "camera_id": camera_id,
        "location_id": location_id,
        "event_type": CAMERA_DEGRADED,
        "confidence": 1.0,
        "payload": {
            "reason": reason,
            "observed_fps": observed_fps,
        },
        "clip_id": "",
        "detected_at": _now_iso(),
    }