"""Event builders for vision.compliance.* subjects.

Each function returns a dict matching the Watch Dog API's
POST /api/v1/vision/detections body. Subjects come from edge.events.__init__
— never inline strings here.
"""
from __future__ import annotations

from datetime import datetime, timezone

from . import (
    COMPLIANCE_BLOCKED_EXIT,
    COMPLIANCE_CLEANLINESS_ALERT,
    COMPLIANCE_HYGIENE_VIOLATION,
    COMPLIANCE_PHONE_USAGE,
    COMPLIANCE_UNIFORM_VIOLATION,
)


def _now_iso() -> str:
    return datetime.now(timezone.utc).isoformat()


def _event(
    camera_id: str,
    location_id: str,
    event_type: str,
    confidence: float,
    payload: dict,
    clip_id: str | None = None,
) -> dict:
    return {
        "camera_id": camera_id,
        "location_id": location_id,
        "event_type": event_type,
        "confidence": confidence,
        "payload": payload,
        "clip_id": clip_id or "",
        "detected_at": _now_iso(),
    }


def phone_usage(
    camera_id: str,
    location_id: str,
    zone_id: str,
    confidence: float,
    frame_url: str = "",
    clip_id: str | None = None,
) -> dict:
    return _event(
        camera_id,
        location_id,
        COMPLIANCE_PHONE_USAGE,
        confidence,
        {
            "violation_type": "phone_usage",
            "zone_id": zone_id,
            "description": "Employee using phone in checkout/aisles zone",
            "confidence": confidence,
            "frame_url": frame_url,
            "clip_id": clip_id or "",
        },
        clip_id,
    )


def uniform_violation(
    camera_id: str,
    location_id: str,
    zone_id: str,
    confidence: float,
    missing_items: list[str],
    frame_url: str = "",
    clip_id: str | None = None,
) -> dict:
    return _event(
        camera_id,
        location_id,
        COMPLIANCE_UNIFORM_VIOLATION,
        confidence,
        {
            "violation_type": "uniform_violation",
            "zone_id": zone_id,
            "missing_items": missing_items,
            "confidence": confidence,
            "frame_url": frame_url,
            "clip_id": clip_id or "",
        },
        clip_id,
    )


def hygiene_violation(
    camera_id: str,
    location_id: str,
    zone_id: str,
    confidence: float,
    violation_detail: str,
    frame_url: str = "",
    clip_id: str | None = None,
) -> dict:
    return _event(
        camera_id,
        location_id,
        COMPLIANCE_HYGIENE_VIOLATION,
        confidence,
        {
            "violation_type": "hygiene_violation",
            "zone_id": zone_id,
            "detail": violation_detail,
            "confidence": confidence,
            "frame_url": frame_url,
            "clip_id": clip_id or "",
        },
        clip_id,
    )


def cleanliness_alert(
    camera_id: str,
    location_id: str,
    zone_id: str,
    confidence: float,
    mess_type: str,
    frame_url: str = "",
    clip_id: str | None = None,
) -> dict:
    return _event(
        camera_id,
        location_id,
        COMPLIANCE_CLEANLINESS_ALERT,
        confidence,
        {
            "violation_type": "cleanliness_alert",
            "zone_id": zone_id,
            "mess_type": mess_type,
            "description": f"Spill/mess detected: {mess_type}",
            "confidence": confidence,
            "frame_url": frame_url,
            "clip_id": clip_id or "",
        },
        clip_id,
    )


def blocked_exit(
    camera_id: str,
    location_id: str,
    zone_id: str,
    confidence: float,
    frame_url: str = "",
    clip_id: str | None = None,
) -> dict:
    return _event(
        camera_id,
        location_id,
        COMPLIANCE_BLOCKED_EXIT,
        confidence,
        {
            "violation_type": "blocked_exit",
            "zone_id": zone_id,
            "description": "Exit path blocked by objects or persons",
            "confidence": confidence,
            "frame_url": frame_url,
            "clip_id": clip_id or "",
        },
        clip_id,
    )