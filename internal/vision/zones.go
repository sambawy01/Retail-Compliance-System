// Package vision — zones.go implements zone CRUD on the vision Service.
package vision

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sambawy01/Retail-Compliance-System/pkg/database"
)

// CreateZoneInput is the payload for creating a zone.
type CreateZoneInput struct {
	CameraID uuid.UUID `json:"camera_id"`
	Name     string    `json:"name"`
	Kind     ZoneKind  `json:"kind"`
	Polygon  any       `json:"polygon"`
	Capacity *int      `json:"capacity,omitempty"`
}

// ErrZoneNotFound is returned when a zone is not found.
var ErrZoneNotFound = errors.New("vision: zone not found")

// CreateZone inserts a new zone row.
func (s *Service) CreateZone(ctx context.Context, in CreateZoneInput) (Zone, error) {
	var z Zone
	err := database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		z = Zone{
			ZoneID:   uuid.New(),
			CameraID: in.CameraID,
			Name:     in.Name,
			Kind:     in.Kind,
			Polygon:  in.Polygon,
			Capacity: in.Capacity,
		}
		const q = `
			INSERT INTO vision_zones (zone_id, org_id, camera_id, name, kind, polygon, capacity)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING created_at`
		return tx.QueryRow(ctx, q,
			z.ZoneID, z.OrgID, z.CameraID, z.Name, z.Kind, z.Polygon, z.Capacity,
		).Scan(&z.CreatedAt)
	})
	if err != nil {
		return Zone{}, fmt.Errorf("vision: create zone: %w", err)
	}
	return z, nil
}

// ListZones returns all zones for a given camera.
func (s *Service) ListZones(ctx context.Context, cameraID string) ([]Zone, error) {
	var out []Zone
	err := database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		camID, parseErr := uuid.Parse(cameraID)
		if parseErr != nil {
			return fmt.Errorf("vision: invalid camera_id: %w", parseErr)
		}
		const q = `
			SELECT zone_id, org_id, camera_id, name, kind, polygon, capacity, created_at
			FROM vision_zones
			WHERE camera_id = $1
			ORDER BY name`
		rows, qerr := tx.Query(ctx, q, camID)
		if qerr != nil {
			return qerr
		}
		defer rows.Close()
		for rows.Next() {
			var z Zone
			if serr := rows.Scan(
				&z.ZoneID, &z.OrgID, &z.CameraID, &z.Name, &z.Kind, &z.Polygon, &z.Capacity, &z.CreatedAt,
			); serr != nil {
				return serr
			}
			out = append(out, z)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteZone removes a zone by ID.
func (s *Service) DeleteZone(ctx context.Context, zoneID string) error {
	return database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		id, parseErr := uuid.Parse(zoneID)
		if parseErr != nil {
			return ErrZoneNotFound
		}
		const q = `DELETE FROM vision_zones WHERE zone_id = $1`
		cmd, err := tx.Exec(ctx, q, id)
		if err != nil {
			return err
		}
		if cmd.RowsAffected() == 0 {
			return ErrZoneNotFound
		}
		return nil
	})
}