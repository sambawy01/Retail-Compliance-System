"""Geometry helpers for polygonal zone hit-testing.

Zones are stored as normalized coordinates (0..1) so they are
resolution-independent. Detectors emit pixel-space bboxes; this module
converts and tests containment using a ray-casting algorithm with no
external geometry dependencies.
"""
from __future__ import annotations


def point_in_zone(nx: float, ny: float, polygon: list[dict]) -> bool:
    """Return True if normalized point (nx, ny) is inside the polygon.

    Polygon is a list of {"x": float, "y": float} in normalized coordinates.
    Uses the ray-casting algorithm — no external geometry library required.
    """
    coords = [(p["x"], p["y"]) for p in polygon]
    if len(coords) < 3:
        return False

    inside = False
    n = len(coords)
    j = n - 1
    for i in range(n):
        xi, yi = coords[i]
        xj, yj = coords[j]
        # Check if the ray from (nx, ny) horizontally crosses edge (j, i)
        if ((yi > ny) != (yj > ny)):
            x_intersect = (xj - xi) * (ny - yi) / (yj - yi) + xi
            if nx < x_intersect:
                inside = not inside
        j = i
    return inside


def normalize_bbox_center(
    bbox: tuple[float, float, float, float], frame_w: int, frame_h: int
) -> tuple[float, float]:
    """Return the normalized (cx, cy) of a pixel-space bbox."""
    x1, y1, x2, y2 = bbox
    return ((x1 + x2) / 2.0 / frame_w, (y1 + y2) / 2.0 / frame_h)


def bbox_in_zone(
    bbox: tuple[float, float, float, float],
    polygon: list[dict],
    frame_w: int,
    frame_h: int,
) -> bool:
    """Return True if the bbox center falls inside the normalized polygon."""
    cx, cy = normalize_bbox_center(bbox, frame_w, frame_h)
    return point_in_zone(cx, cy, polygon)