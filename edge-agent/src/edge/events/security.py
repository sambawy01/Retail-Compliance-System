"""Event builders for security-related subjects:
theft.cash_drawer, access.after_hours, security.loitering, labor.buddy_punch.
"""
from __future__ import annotations

from datetime import datetime, timezone

from . import (
    ACCESS_AFTER_HOURS,
    LABOR_BUDDY_PUNCH,
    SECURITY_LOITERING,
    THEFT_CASH_DRAWER,
)


def _now_iso() -> str:
    return datetime.now(timezone.utc).isoformat()


def cash_drawer(
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
        "event_type": THEFT_CASH_DRAWER,
        "confidence": confidence,
        "payload": {
            "anomaly_type": "cash_drawer_tamper",
            "zone_id": zone_id,
            "description": "Cash drawer open without matching POS transaction",
            "confidence": confidence,
            "frame_url": frame_url,
            "clip_id": clip_id or "",
        },
        "clip_id": clip_id or "",
        "detected_at": _now_iso(),
    }


def after_hours(
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
        "event_type": ACCESS_AFTER_HOURS,
        "confidence": confidence,
        "payload": {
            "anomaly_type": "after_hours_access",
            "zone_id": zone_id,
            "description": "Person detected after business hours",
            "confidence": confidence,
            "frame_url": frame_url,
            "clip_id": clip_id or "",
        },
        "clip_id": clip_id or "",
        "detected_at": _now_iso(),
    }


def loitering(
    camera_id: str,
    location_id: str,
    zone_id: str,
    confidence: float,
    dwell_seconds: float,
    person_count: int = 1,
    frame_url: str = "",
    clip_id: str | None = None,
) -> dict:
    return {
        "camera_id": camera_id,
        "location_id": location_id,
        "event_type": SECURITY_LOITERING,
        "confidence": confidence,
        "payload": {
            "anomaly_type": "loitering",
            "zone_id": zone_id,
            "dwell_seconds": dwell_seconds,
            "person_count": person_count,
            "description": f"Person stationary for {dwell_seconds:.0f}s in zone",
            "confidence": confidence,
            "frame_url": frame_url,
            "clip_id": clip_id or "",
        },
        "clip_id": clip_id or "",
        "detected_at": _now_iso(),
    }


def buddy_punch(
    camera_id: str,
    location_id: str,
    shift_id: str,
    matched_person_id: str | None,
    confidence: float,
    notes: str,
    frame_url: str = "",
    clip_id: str | None = None,
) -> dict:
    return {
        "camera_id": camera_id,
        "location_id": location_id,
        "event_type": LABOR_BUDDY_PUNCH,
        "confidence": confidence,
        "payload": {
            "anomaly_type": "buddy_punch",
            "shift_id": shift_id,
            "matched_person_id": matched_person_id or "",
            "confidence": confidence,
            "notes": notes,
            "frame_url": frame_url,
            "clip_id": clip_id or "",
        },
        "clip_id": clip_id or "",
        "detected_at": _now_iso(),
    }