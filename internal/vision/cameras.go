// Package vision — cameras.go implements Camera CRUD operations scoped to
// the tenant via Row Level Security. All queries run through database.TenantTx
// which sets the app.current_org_id session variable before executing.
package vision

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/sambawy01/Retail-Compliance-System/pkg/database"
)

// CreateCameraInput is the payload for creating a camera.
type CreateCameraInput struct {
	LocationID uuid.UUID `json:"location_id"`
	Name       string    `json:"name"`
	RTSPURL    string    `json:"rtsp_url"`
}

// UpdateCameraInput is the payload for updating a camera. Pointer fields are
// optional; nil fields leave the column unchanged.
type UpdateCameraInput struct {
	Name    *string       `json:"name,omitempty"`
	RTSPURL *string       `json:"rtsp_url,omitempty"`
	Status  *CameraStatus `json:"status,omitempty"`
}

// ErrCameraNotFound is returned when a camera is not found within the tenant scope.
var ErrCameraNotFound = errors.New("vision: camera not found")

// CreateCamera inserts a new camera row scoped to the tenant in the context.
func (s *Service) CreateCamera(ctx context.Context, in CreateCameraInput) (Camera, error) {
	var cam Camera
	err := database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		const q = `
			INSERT INTO vision_cameras (camera_id, org_id, location_id, name, rtsp_url, status)
			VALUES ($1, current_setting('app.current_org_id', true)::uuid, $2, $3, $4, $5)
			RETURNING camera_id, org_id, location_id, name, rtsp_url, status,
			          last_heartbeat_at, created_at, updated_at`
		cam.CameraID = uuid.New()
		cam.Status = CameraStatusPending
		return tx.QueryRow(ctx, q,
			cam.CameraID, in.LocationID, in.Name, in.RTSPURL, string(cam.Status),
		).Scan(
			&cam.CameraID, &cam.OrgID, &cam.LocationID, &cam.Name, &cam.RTSPURL, &cam.Status,
			&cam.LastHeartbeatAt, &cam.CreatedAt, &cam.UpdatedAt,
		)
	})
	if err != nil {
		return Camera{}, fmt.Errorf("vision: create camera: %w", err)
	}
	return cam, nil
}

// GetCamera retrieves a single camera by ID within the tenant scope.
func (s *Service) GetCamera(ctx context.Context, cameraID string) (Camera, error) {
	var cam Camera
	id, err := uuid.Parse(cameraID)
	if err != nil {
		return Camera{}, fmt.Errorf("vision: invalid camera id: %w", err)
	}
	err = database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		const q = `
			SELECT camera_id, org_id, location_id, name, rtsp_url, status,
			       last_heartbeat_at, created_at, updated_at
			FROM vision_cameras
			WHERE camera_id = $1`
		return tx.QueryRow(ctx, q, id).Scan(
			&cam.CameraID, &cam.OrgID, &cam.LocationID, &cam.Name, &cam.RTSPURL, &cam.Status,
			&cam.LastHeartbeatAt, &cam.CreatedAt, &cam.UpdatedAt,
		)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Camera{}, ErrCameraNotFound
	}
	if err != nil {
		return Camera{}, fmt.Errorf("vision: get camera: %w", err)
	}
	return cam, nil
}

// ListCameras lists cameras for a location; if locationID is empty, all
// cameras for the tenant are returned.
func (s *Service) ListCameras(ctx context.Context, locationID string) ([]Camera, error) {
	var cams []Camera
	locFilter := uuid.Nil
	if locationID != "" {
		id, err := uuid.Parse(locationID)
		if err != nil {
			return nil, fmt.Errorf("vision: invalid location id: %w", err)
		}
		locFilter = id
	}
	err := database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		const q = `
			SELECT camera_id, org_id, location_id, name, rtsp_url, status,
			       last_heartbeat_at, created_at, updated_at
			FROM vision_cameras
			WHERE ($1::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR location_id = $1)
			ORDER BY name`
		rows, err := tx.Query(ctx, q, locFilter)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var c Camera
			if err := rows.Scan(
				&c.CameraID, &c.OrgID, &c.LocationID, &c.Name, &c.RTSPURL, &c.Status,
				&c.LastHeartbeatAt, &c.CreatedAt, &c.UpdatedAt,
			); err != nil {
				return err
			}
			cams = append(cams, c)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("vision: list cameras: %w", err)
	}
	return cams, nil
}

// UpdateCamera updates mutable camera fields.
func (s *Service) UpdateCamera(ctx context.Context, cameraID string, in UpdateCameraInput) (Camera, error) {
	var cam Camera
	id, err := uuid.Parse(cameraID)
	if err != nil {
		return Camera{}, fmt.Errorf("vision: invalid camera id: %w", err)
	}
	err = database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		var statusVal any
		if in.Status != nil {
			statusVal = string(*in.Status)
		}
		const q = `
			UPDATE vision_cameras
			SET name       = COALESCE($2, name),
			    rtsp_url   = COALESCE($3, rtsp_url),
			    status     = COALESCE($4, status),
			    updated_at = now()
			WHERE camera_id = $1
			RETURNING camera_id, org_id, location_id, name, rtsp_url, status,
			          last_heartbeat_at, created_at, updated_at`
		return tx.QueryRow(ctx, q, id, in.Name, in.RTSPURL, statusVal).Scan(
			&cam.CameraID, &cam.OrgID, &cam.LocationID, &cam.Name, &cam.RTSPURL, &cam.Status,
			&cam.LastHeartbeatAt, &cam.CreatedAt, &cam.UpdatedAt,
		)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Camera{}, ErrCameraNotFound
	}
	if err != nil {
		return Camera{}, fmt.Errorf("vision: update camera: %w", err)
	}
	return cam, nil
}

// UpdateCameraStatus updates the camera status and last_heartbeat timestamp.
// Used by camera heartbeat pings.
func (s *Service) UpdateCameraStatus(ctx context.Context, cameraID string, status CameraStatus) error {
	id, err := uuid.Parse(cameraID)
	if err != nil {
		return fmt.Errorf("vision: invalid camera id: %w", err)
	}
	var tag pgconn.CommandTag
	err = database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		const q = `
			UPDATE vision_cameras
			SET status = $2, last_heartbeat_at = now(), updated_at = now()
			WHERE camera_id = $1`
		var err error
		tag, err = tx.Exec(ctx, q, id, string(status))
		return err
	})
	if err != nil {
		return fmt.Errorf("vision: update status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCameraNotFound
	}
	return nil
}

// DeleteCamera removes a camera row within the tenant scope.
func (s *Service) DeleteCamera(ctx context.Context, cameraID string) error {
	id, err := uuid.Parse(cameraID)
	if err != nil {
		return fmt.Errorf("vision: invalid camera id: %w", err)
	}
	var tag pgconn.CommandTag
	err = database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		const q = `DELETE FROM vision_cameras WHERE camera_id = $1`
		var err error
		tag, err = tx.Exec(ctx, q, id)
		return err
	})
	if err != nil {
		return fmt.Errorf("vision: delete camera: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCameraNotFound
	}
	return nil
}