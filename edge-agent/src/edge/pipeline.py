"""Per-camera pipeline orchestrator for retail compliance.

One CameraPipeline is created per registered camera. It receives frames
from the RTSP thread, dispatches to the configured detectors, builds events
when retail compliance conditions are met, and forwards them to the
on_detection callback (wired to the Watch Dog API client by agent.py).

The pipeline knows nothing about HTTP, threading, or models — only the
Detector protocol and zone geometry.

Retail checks implemented:
  - phone_usage: person + phone detected in checkout/aisles zone
  - cleanliness_alert: spill/mess detected (placeholder)
  - checkout_bottleneck: too many people in checkout zone
  - loitering: person stationary in entrance zone for >N seconds
  - after_hours: person detected outside business hours
  - buddy_punch: face match fails during clock-in
  - slip_fall: pose detection — person on ground (placeholder)
  - cash_drawer: person at register with no transaction (placeholder)
  - loyalty_recognized: face match succeeds
  - occupancy_update: person count per zone
"""
from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, time, timezone
from typing import Callable, Optional

import numpy as np

from .config import CameraAssignment
from .detectors import Detection, Detector
from .detectors.zone_geometry import normalize_bbox_center, point_in_zone
from .events import compliance, customer, operations, safety, security
from .logging_setup import get_logger

log = get_logger(__name__)

# Retail zone type constants
ZONE_CHECKOUT = "checkout"
ZONE_AISLES = "aisles"
ZONE_STOCKROOM = "stockroom"
ZONE_BACK_OFFICE = "back_office"
ZONE_ENTRANCE = "entrance"
ZONE_RESTROOM = "restroom"
ZONE_RESTRICTED = "restricted"
ZONE_PRIVACY_MASK = "privacy_mask"

# Default thresholds
CHECKOUT_BOTTLENECK_THRESHOLD = 4
LOITERING_THRESHOLD_SECONDS = 120.0
OCCUPANCY_REPORT_INTERVAL = 15.0  # seconds between occupancy reports


@dataclass
class CameraPipeline:
    """Per-camera retail compliance orchestrator."""

    camera: CameraAssignment
    location_id: str
    person_detector: Detector
    on_detection: Callable[[dict], None]
    object_detector: Optional[Detector] = None
    face_detector: Optional[Detector] = None
    api_client: Optional[object] = None  # WatchDogClient for face matching
    zones: list[dict] = field(default_factory=list)
    # Business hours for after_hours detection
    business_hours_open: time = field(default_factory=lambda: time(7, 0))
    business_hours_close: time = field(default_factory=lambda: time(22, 0))
    # Internal state
    _last_event_ts: dict[str, float] = field(default_factory=dict)
    _debounce_s: float = 10.0
    _occupancy_last_report: float = 0.0
    _prev_occupancy: dict[str, int] = field(default_factory=dict)
    # Loitering tracking: zone_id -> {track_key -> first_seen_ts}
    _loiter_tracker: dict[str, dict[str, float]] = field(default_factory=dict)

    def process_frame(self, frame: np.ndarray, frame_ts: float) -> None:
        """Process one frame through all enabled retail checks."""
        people = self.person_detector.detect(frame, frame_ts)
        objs: list[Detection] = []
        if self.object_detector is not None:
            objs = self.object_detector.detect(frame, frame_ts)

        faces: list[Detection] = []
        if self.face_detector is not None:
            faces = self.face_detector.detect(frame, frame_ts)

        h, w = _frame_shape(frame)

        # Run all enabled checks based on feature flags
        flags = self.camera.feature_flags

        if flags.get("phone_usage", True):
            self._check_phone_usage(frame_ts, people, objs, w, h)

        if flags.get("cleanliness", False):
            self._check_cleanliness(frame_ts, objs, w, h)

        if flags.get("checkout_bottleneck", True):
            self._check_checkout_bottleneck(frame_ts, people, w, h)

        if flags.get("loitering", True):
            self._check_loitering(frame_ts, people, w, h)

        if flags.get("after_hours", True):
            self._check_after_hours(frame_ts, people, w, h)

        if flags.get("slip_fall", False):
            self._check_slip_fall(frame_ts, people, w, h)

        if flags.get("cash_drawer", False):
            self._check_cash_drawer(frame_ts, people, objs, w, h)

        if flags.get("loyalty", False) and faces:
            self._check_loyalty(frame_ts, faces, w, h)

        if flags.get("occupancy", True):
            self._check_occupancy(frame_ts, people, w, h)

    # -- Individual checks ------------------------------------------------

    def _check_phone_usage(
        self,
        frame_ts: float,
        people: list[Detection],
        objs: list[Detection],
        w: int,
        h: int,
    ) -> None:
        """Person + phone detected in checkout or aisles zone."""
        if not people:
            return
        phones = [o for o in objs if o.label == "phone"]
        if not phones:
            return

        target_zones = self._zones_of_type(ZONE_CHECKOUT, ZONE_AISLES)
        if not target_zones:
            return

        for p in people:
            cx, cy = normalize_bbox_center(p.bbox, w, h)
            in_zone = any(point_in_zone(cx, cy, z["polygon"]) for z in target_zones)
            if not in_zone:
                continue
            # Check if any phone bbox overlaps with this person
            phone_near = any(
                _bbox_overlap(p.bbox, ph.bbox) for ph in phones
            )
            if phone_near and self._debounced("phone_usage", frame_ts):
                zone_id = self._zone_id_at(cx, cy)
                ev = compliance.phone_usage(
                    camera_id=self.camera.camera_id,
                    location_id=self.location_id,
                    zone_id=zone_id,
                    confidence=min(p.confidence, 0.85),
                )
                self.on_detection(ev)

    def _check_cleanliness(
        self,
        frame_ts: float,
        objs: list[Detection],
        w: int,
        h: int,
    ) -> None:
        """Spill/mess detected — placeholder logic using object labels."""
        mess_objs = [o for o in objs if o.label in ("spill", "mess")]
        if not mess_objs:
            return
        for m in mess_objs:
            cx, cy = normalize_bbox_center(m.bbox, w, h)
            zone_id = self._zone_id_at(cx, cy)
            if self._debounced("cleanliness_alert", frame_ts):
                ev = compliance.cleanliness_alert(
                    camera_id=self.camera.camera_id,
                    location_id=self.location_id,
                    zone_id=zone_id,
                    confidence=m.confidence,
                    mess_type=m.label,
                )
                self.on_detection(ev)

    def _check_checkout_bottleneck(
        self,
        frame_ts: float,
        people: list[Detection],
        w: int,
        h: int,
    ) -> None:
        """Too many people in checkout zone."""
        checkout_zones = self._zones_of_type(ZONE_CHECKOUT)
        if not checkout_zones:
            return
        for zone in checkout_zones:
            count = sum(
                1
                for p in people
                if point_in_zone(*normalize_bbox_center(p.bbox, w, h), zone["polygon"])
            )
            threshold = zone.get("config", {}).get("bottleneck_threshold",
                                                     CHECKOUT_BOTTLENECK_THRESHOLD)
            if count >= threshold and self._debounced(
                f"checkout_bottleneck:{zone.get('zone_id', '')}", frame_ts
            ):
                severity = "high" if count >= threshold * 1.5 else "medium"
                ev = operations.checkout_bottleneck(
                    camera_id=self.camera.camera_id,
                    location_id=self.location_id,
                    zone_id=zone.get("zone_id", ""),
                    zone_name=zone.get("name", ""),
                    person_count=count,
                    threshold=threshold,
                    severity=severity,
                )
                self.on_detection(ev)

    def _check_loitering(
        self,
        frame_ts: float,
        people: list[Detection],
        w: int,
        h: int,
    ) -> None:
        """Person stationary in entrance zone for >N seconds."""
        entrance_zones = self._zones_of_type(ZONE_ENTRANCE)
        if not entrance_zones:
            return
        for zone in entrance_zones:
            zid = zone.get("zone_id", "")
            tracker = self._loiter_tracker.setdefault(zid, {})
            current_keys = set()
            for i, p in enumerate(people):
                cx, cy = normalize_bbox_center(p.bbox, w, h)
                if not point_in_zone(cx, cy, zone["polygon"]):
                    continue
                # Simple spatial key for tracking (grid-cell approximation)
                key = f"{int(cx * 10)}:{int(cy * 10)}"
                current_keys.add(key)
                if key not in tracker:
                    tracker[key] = frame_ts
                else:
                    dwell = frame_ts - tracker[key]
                    threshold = zone.get("config", {}).get(
                        "loitering_threshold_seconds", LOITERING_THRESHOLD_SECONDS
                    )
                    if dwell >= threshold and self._debounced(
                        f"loitering:{zid}:{key}", frame_ts
                    ):
                        ev = security.loitering(
                            camera_id=self.camera.camera_id,
                            location_id=self.location_id,
                            zone_id=zid,
                            confidence=0.7,
                            dwell_seconds=dwell,
                        )
                        self.on_detection(ev)
            # Remove keys no longer present
            for k in list(tracker):
                if k not in current_keys:
                    del tracker[k]

    def _check_after_hours(
        self,
        frame_ts: float,
        people: list[Detection],
        w: int,
        h: int,
    ) -> None:
        """Person detected outside business hours."""
        now = datetime.now(timezone.utc).time()
        if self.business_hours_open <= now <= self.business_hours_close:
            return  # within business hours
        if not people:
            return
        if self._debounced("after_hours", frame_ts):
            # Report on first person detected
            p = people[0]
            cx, cy = normalize_bbox_center(p.bbox, w, h)
            zone_id = self._zone_id_at(cx, cy)
            ev = security.after_hours(
                camera_id=self.camera.camera_id,
                location_id=self.location_id,
                zone_id=zone_id,
                confidence=p.confidence,
            )
            self.on_detection(ev)

    def _check_slip_fall(
        self,
        frame_ts: float,
        people: list[Detection],
        w: int,
        h: int,
    ) -> None:
        """Pose detection — person on ground (placeholder).

        A real implementation would use a pose estimator (YOLO-pose or
        MediaPipe) to detect when a person's body is horizontal. For now,
        this is a placeholder that uses bbox aspect ratio: if a person bbox
        is wider than tall, it may indicate someone on the ground.
        """
        for p in people:
            x1, y1, x2, y2 = p.bbox
            bw = x2 - x1
            bh = y2 - y1
            if bh > 0 and bw / bh > 1.5:  # wider than tall → likely on ground
                cx, cy = normalize_bbox_center(p.bbox, w, h)
                zone_id = self._zone_id_at(cx, cy)
                if self._debounced("slip_fall", frame_ts):
                    ev = safety.slip_fall(
                        camera_id=self.camera.camera_id,
                        location_id=self.location_id,
                        zone_id=zone_id,
                        confidence=0.6,
                    )
                    self.on_detection(ev)

    def _check_cash_drawer(
        self,
        frame_ts: float,
        people: list[Detection],
        objs: list[Detection],
        w: int,
        h: int,
    ) -> None:
        """Person at register with no transaction (placeholder).

        A real implementation would integrate with POS data to check if a
        transaction is active. For now, this is a placeholder.
        """
        # Placeholder: person detected in checkout zone near cash drawer object
        checkout_zones = self._zones_of_type(ZONE_CHECKOUT)
        if not checkout_zones:
            return
        cash_drawers = [o for o in objs if o.label == "cash_drawer"]
        if not cash_drawers or not people:
            return
        for zone in checkout_zones:
            zid = zone.get("zone_id", "")
            for p in people:
                cx, cy = normalize_bbox_center(p.bbox, w, h)
                if not point_in_zone(cx, cy, zone["polygon"]):
                    continue
                if self._debounced(f"cash_drawer:{zid}", frame_ts):
                    ev = security.cash_drawer(
                        camera_id=self.camera.camera_id,
                        location_id=self.location_id,
                        zone_id=zid,
                        confidence=0.5,
                    )
                    self.on_detection(ev)

    def _check_loyalty(
        self,
        frame_ts: float,
        faces: list[Detection],
        w: int,
        h: int,
    ) -> None:
        """Face match succeeds — emit loyalty_recognized event."""
        from .detectors.face_arcface import embedding_of

        if self.api_client is None:
            return
        for f in faces:
            emb = embedding_of(f)
            if emb is None:
                continue
            try:
                match = self.api_client.match_face(emb, threshold=0.6)
            except Exception as e:  # noqa: BLE001
                log.warning("face_match_failed", error=str(e),
                            camera_id=self.camera.camera_id)
                continue
            if match is None:
                continue
            cx, cy = normalize_bbox_center(f.bbox, w, h)
            zone_id = self._zone_id_at(cx, cy)
            similarity = float(match.get("similarity", 0.0))
            person_id = str(match.get("person_id", ""))
            display_name = str(match.get("display_name", ""))
            ev = customer.loyalty_recognized(
                camera_id=self.camera.camera_id,
                location_id=self.location_id,
                zone_id=zone_id,
                person_id=person_id,
                display_name=display_name,
                similarity=similarity,
                confidence=f.confidence,
            )
            self.on_detection(ev)

    def _check_occupancy(
        self,
        frame_ts: float,
        people: list[Detection],
        w: int,
        h: int,
    ) -> None:
        """Person count per zone — emit occupancy_update periodically."""
        if frame_ts - self._occupancy_last_report < OCCUPANCY_REPORT_INTERVAL:
            return
        self._occupancy_last_report = frame_ts
        for zone in self.zones:
            zid = zone.get("zone_id", "")
            zname = zone.get("name", "")
            polygon = zone.get("polygon", [])
            if not polygon or len(polygon) < 3:
                continue
            count = sum(
                1
                for p in people
                if point_in_zone(*normalize_bbox_center(p.bbox, w, h), polygon)
            )
            prev = self._prev_occupancy.get(zid, 0)
            capacity = zone.get("config", {}).get("capacity", 20)
            util = (count / capacity * 100.0) if capacity > 0 else 0.0
            ev = customer.occupancy_update(
                camera_id=self.camera.camera_id,
                location_id=self.location_id,
                zone_id=zid,
                zone_name=zname,
                person_count=count,
                previous_count=prev,
                utilization_pct=util,
            )
            self.on_detection(ev)
            self._prev_occupancy[zid] = count

    # -- Helpers ----------------------------------------------------------

    def _debounced(self, kind: str, frame_ts: float) -> bool:
        """Return True if a new event of this kind is allowed to fire now.

        First call always allowed. Subsequent calls within _debounce_s
        are suppressed.
        """
        if kind not in self._last_event_ts:
            self._last_event_ts[kind] = frame_ts
            return True
        last = self._last_event_ts[kind]
        if frame_ts - last < self._debounce_s:
            return False
        self._last_event_ts[kind] = frame_ts
        return True

    def _zones_of_type(self, *zone_types: str) -> list[dict]:
        """Return zones matching any of the given zone types."""
        return [
            z for z in self.zones
            if z.get("kind") in zone_types or z.get("type") in zone_types
        ]

    def _zone_id_at(self, nx: float, ny: float) -> str:
        for z in self.zones:
            if point_in_zone(nx, ny, z.get("polygon", [])):
                return z.get("zone_id", "")
        return ""


def _frame_shape(frame: np.ndarray) -> tuple[int, int]:
    """Return (height, width) from a frame array."""
    if frame.ndim == 3:
        return frame.shape[0], frame.shape[1]
    return frame.shape[0], frame.shape[1]


def _bbox_overlap(
    a: tuple[float, float, float, float],
    b: tuple[float, float, float, float],
) -> bool:
    """Return True if two bboxes overlap (IoU > 0)."""
    ax1, ay1, ax2, ay2 = a
    bx1, by1, bx2, by2 = b
    return not (ax2 < bx1 or bx2 < ax1 or ay2 < by1 or by2 < ay1)