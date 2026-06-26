// Package vision — detections.go implements Detection insert and list
// operations. InsertDetection resolves severity via ResolveSeverity, inserts
// the row, and publishes an event on the bus.
package vision

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sambawy01/Retail-Compliance-System/internal/event"
	"github.com/sambawy01/Retail-Compliance-System/internal/tenant"
	"github.com/sambawy01/Retail-Compliance-System/pkg/database"
)

// InsertDetectionInput is the payload for recording a detection.
type InsertDetectionInput struct {
	CameraID   uuid.UUID     `json:"camera_id"`
	ZoneID     *uuid.UUID    `json:"zone_id,omitempty"`
	EventType  string        `json:"event_type"`
	Confidence float64       `json:"confidence"`
	Payload    map[string]any `json:"payload,omitempty"`
	ClipID     *uuid.UUID    `json:"clip_id,omitempty"`
	DetectedAt time.Time     `json:"detected_at"`
}

// ListDetectionsFilter filters detection queries.
type ListDetectionsFilter struct {
	CameraID  string
	EventType string
	Severity  string
	Since     time.Time
	Until     time.Time
	Limit     int
}

// ErrDetectionNotFound is returned when no detection matches.
var ErrDetectionNotFound = errors.New("vision: detection not found")

// InsertDetection resolves severity, inserts the detection row, and publishes
// an event on the bus for downstream consumers.
func (s *Service) InsertDetection(ctx context.Context, in InsertDetectionInput) (Detection, error) {
	var det Detection
	det.DetectionID = uuid.New()
	det.CameraID = in.CameraID
	det.ZoneID = in.ZoneID
	det.EventType = in.EventType
	det.Severity = ResolveSeverity(in.EventType)
	det.Confidence = in.Confidence
	det.Payload = in.Payload
	det.ClipID = in.ClipID
	if in.DetectedAt.IsZero() {
		det.DetectedAt = time.Now().UTC()
	} else {
		det.DetectedAt = in.DetectedAt
	}

	err := database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		const q = `
			INSERT INTO vision_detections (detection_id, org_id, camera_id, zone_id, event_type, severity, confidence, payload, clip_id, detected_at)
			VALUES ($1, current_setting('app.current_org_id')::uuid, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING org_id, location_id, created_at`
		return tx.QueryRow(ctx, q,
			det.DetectionID, det.CameraID, det.ZoneID, det.EventType, string(det.Severity),
			det.Confidence, det.Payload, det.ClipID, det.DetectedAt,
		).Scan(&det.OrgID, &det.LocationID, &det.CreatedAt)
	})
	if err != nil {
		return Detection{}, fmt.Errorf("vision: insert detection: %w", err)
	}

	// Publish event on the bus (best-effort).
	if s.bus != nil {
		orgID, _ := tenant.OrgIDFrom(ctx)
		s.bus.Publish(ctx, event.Envelope{
			EventID:       det.DetectionID.String(),
			EventType:     det.EventType,
			OrgID:         orgID.String(),
			Source:        "vision",
			SchemaVersion: 1,
			Payload:       det,
		})
	}

	return det, nil
}

// ListDetections returns detections matching the given filter.
func (s *Service) ListDetections(ctx context.Context, f ListDetectionsFilter) ([]Detection, error) {
	var dets []Detection
	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}

	var camFilter uuid.UUID
	if f.CameraID != "" {
		id, err := uuid.Parse(f.CameraID)
		if err != nil {
			return nil, fmt.Errorf("vision: invalid camera id: %w", err)
		}
		camFilter = id
	}

	var sinceVal, untilVal any
	if !f.Since.IsZero() {
		sinceVal = f.Since
	}
	if !f.Until.IsZero() {
		untilVal = f.Until
	}

	err := database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		const q = `
			SELECT detection_id, org_id, location_id, camera_id, zone_id, event_type,
			       severity, confidence, payload, clip_id, detected_at, created_at
			FROM vision_detections
			WHERE ($1::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR camera_id = $1)
			  AND ($2 = '' OR event_type = $2)
			  AND ($3 = '' OR severity = $3)
			  AND ($4::timestamptz IS NULL OR detected_at >= $4)
			  AND ($5::timestamptz IS NULL OR detected_at <= $5)
			ORDER BY detected_at DESC
			LIMIT $6`
		rows, err := tx.Query(ctx, q, camFilter, f.EventType, f.Severity, sinceVal, untilVal, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var d Detection
			if err := rows.Scan(
				&d.DetectionID, &d.OrgID, &d.LocationID, &d.CameraID, &d.ZoneID, &d.EventType,
				&d.Severity, &d.Confidence, &d.Payload, &d.ClipID, &d.DetectedAt, &d.CreatedAt,
			); err != nil {
				return err
			}
			dets = append(dets, d)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("vision: list detections: %w", err)
	}
	return dets, nil
}