// Package vision defines the retail-adapted domain types for the Watch Dog
// compliance system: cameras, zones, detections, and clips.
package vision

import (
	"time"

	"github.com/google/uuid"
)

// Severity classifies the urgency of a detection.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

// CameraStatus describes the operational state of a camera.
type CameraStatus string

const (
	CameraStatusPending CameraStatus = "pending"
	CameraStatusOnline  CameraStatus = "online"
	CameraStatusOffline CameraStatus = "offline"
	CameraStatusDegraded CameraStatus = "degraded"
)

// ZoneKind categorises a monitored area within a retail location.
type ZoneKind string

const (
	ZoneKindCheckout    ZoneKind = "checkout"
	ZoneKindAisles      ZoneKind = "aisles"
	ZoneKindStockroom    ZoneKind = "stockroom"
	ZoneKindBackOffice  ZoneKind = "back_office"
	ZoneKindEntrance    ZoneKind = "entrance"
	ZoneKindRestroom    ZoneKind = "restroom"
	ZoneKindRestricted  ZoneKind = "restricted"
	ZoneKindPrivacyMask ZoneKind = "privacy_mask"
)

// RetentionTier controls the storage lifecycle of a clip.
type RetentionTier string

const (
	RetentionHot     RetentionTier = "hot"
	RetentionWarm    RetentionTier = "warm"
	RetentionGlacier RetentionTier = "glacier"
	RetentionDeleted RetentionTier = "deleted"
)

// Camera is a CCTV device attached to a retail location.
type Camera struct {
	CameraID       uuid.UUID      `json:"camera_id"`
	OrgID          uuid.UUID      `json:"org_id"`
	LocationID     uuid.UUID      `json:"location_id"`
	Name           string         `json:"name"`
	RTSPURL        string         `json:"rtsp_url"`
	LocalAgentID   *uuid.UUID     `json:"local_agent_id,omitempty"`
	Status         CameraStatus   `json:"status"`
	LastHeartbeatAt *time.Time     `json:"last_heartbeat_at,omitempty"`
	FeatureFlags    map[string]any `json:"feature_flags,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// Zone is a logical region within a location monitored by a camera.
type Zone struct {
	ZoneID     uuid.UUID `json:"zone_id"`
	OrgID      uuid.UUID `json:"org_id"`
	CameraID   uuid.UUID `json:"camera_id"`
	Name       string    `json:"name"`
	Kind       ZoneKind  `json:"kind"`
	Polygon    any       `json:"polygon,omitempty"` // geojson or pg polygon
	Capacity   *int      `json:"capacity,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// Detection is an event raised by the vision pipeline for a camera/zone.
type Detection struct {
	DetectionID  uuid.UUID      `json:"detection_id"`
	OrgID        uuid.UUID      `json:"org_id"`
	LocationID   uuid.UUID      `json:"location_id"`
	CameraID     uuid.UUID      `json:"camera_id"`
	ZoneID       *uuid.UUID     `json:"zone_id,omitempty"`
	EventType    string         `json:"event_type"`
	Severity     Severity       `json:"severity"`
	Confidence   float64        `json:"confidence"`
	Payload      map[string]any  `json:"payload,omitempty"`
	ClipID       *uuid.UUID     `json:"clip_id,omitempty"`
	DetectedAt   time.Time      `json:"detected_at"`
	CreatedAt    time.Time      `json:"created_at"`
}

// Clip is a stored video segment backing a detection.
type Clip struct {
	ClipID          uuid.UUID     `json:"clip_id"`
	OrgID           uuid.UUID     `json:"org_id"`
	LocationID      uuid.UUID     `json:"location_id"`
	CameraID        uuid.UUID     `json:"camera_id"`
	S3Bucket        string        `json:"s3_bucket"`
	S3Key           string        `json:"s3_key"`
	DurationSeconds float64       `json:"duration_seconds"`
	StartsAt        time.Time     `json:"starts_at"`
	RetentionTier   RetentionTier `json:"retention_tier"`
	CreatedAt       time.Time     `json:"created_at"`
}