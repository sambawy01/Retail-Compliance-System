"""Event builders for customer/occupancy subjects:
customer.loyalty_recognized, occupancy.update, activity.update.
"""
from __future__ import annotations

from datetime import datetime, timezone

from . import (
    ACTIVITY_UPDATE,
    CUSTOMER_LOYALTY_RECOGNIZED,
    OCCUPANCY_UPDATE,
)


def _now_iso() -> str:
    return datetime.now(timezone.utc).isoformat()


def loyalty_recognized(
    camera_id: str,
    location_id: str,
    zone_id: str,
    person_id: str,
    display_name: str,
    similarity: float,
    confidence: float,
) -> dict:
    return {
        "camera_id": camera_id,
        "location_id": location_id,
        "event_type": CUSTOMER_LOYALTY_RECOGNIZED,
        "confidence": confidence,
        "payload": {
            "zone_id": zone_id,
            "person_id": person_id,
            "display_name": display_name,
            "similarity": similarity,
            "confidence": confidence,
        },
        "clip_id": "",
        "detected_at": _now_iso(),
    }


def occupancy_update(
    camera_id: str,
    location_id: str,
    zone_id: str,
    zone_name: str,
    person_count: int,
    previous_count: int,
    utilization_pct: float,
    frame_url: str = "",
) -> dict:
    return {
        "camera_id": camera_id,
        "location_id": location_id,
        "event_type": OCCUPANCY_UPDATE,
        "confidence": 1.0,
        "payload": {
            "zone_id": zone_id,
            "zone_name": zone_name,
            "person_count": person_count,
            "previous_count": previous_count,
            "delta": person_count - previous_count,
            "utilization_pct": utilization_pct,
            "frame_url": frame_url,
        },
        "clip_id": "",
        "detected_at": _now_iso(),
    }


def activity_update(
    camera_id: str,
    location_id: str,
    zone_id: str,
    activity_type: str,
    actor_count: int,
    duration_secs: int,
) -> dict:
    return {
        "camera_id": camera_id,
        "location_id": location_id,
        "event_type": ACTIVITY_UPDATE,
        "confidence": 1.0,
        "payload": {
            "zone_id": zone_id,
            "activity_type": activity_type,
            "actor_count": actor_count,
            "duration_secs": duration_secs,
        },
        "clip_id": "",
        "detected_at": _now_iso(),
    }