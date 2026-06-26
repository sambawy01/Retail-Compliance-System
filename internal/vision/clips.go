// Package vision — clips.go implements clip metadata CRUD.
package vision

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sambawy01/Retail-Compliance-System/pkg/database"
)

type InsertClipInput struct {
	LocationID      uuid.UUID     `json:"location_id"`
	CameraID        uuid.UUID     `json:"camera_id"`
	S3Bucket        string         `json:"s3_bucket"`
	S3Key           string         `json:"s3_key"`
	DurationSeconds float64       `json:"duration_seconds"`
	StartsAt        time.Time     `json:"starts_at"`
	RetentionTier   RetentionTier `json:"retention_tier"`
}

var ErrClipNotFound = errors.New("vision: clip not found")

func (s *Service) InsertClip(ctx context.Context, in InsertClipInput) (Clip, error) {
	return insertClip(ctx, s.pool, in)
}

func (s *Service) GetClip(ctx context.Context, clipID string) (Clip, error) {
	return getClip(ctx, s.pool, clipID)
}

func (s *Service) ListClips(ctx context.Context, cameraID string, limit int) ([]Clip, error) {
	return listClips(ctx, s.pool, cameraID, limit)
}

func (s *Service) UpdateRetentionTier(ctx context.Context, clipID string, tier RetentionTier) error {
	return updateRetentionTier(ctx, s.pool, clipID, tier)
}

func insertClip(ctx context.Context, pool *pgxpool.Pool, in InsertClipInput) (Clip, error) {
	if pool == nil {
		return Clip{}, ErrClipNotFound
	}
	c := Clip{
		ClipID:          uuid.New(),
		LocationID:      in.LocationID,
		CameraID:        in.CameraID,
		S3Bucket:        in.S3Bucket,
		S3Key:           in.S3Key,
		DurationSeconds: in.DurationSeconds,
		StartsAt:        in.StartsAt,
		RetentionTier:   in.RetentionTier,
	}
	if c.RetentionTier == "" {
		c.RetentionTier = RetentionHot
	}
	err := database.TenantTx(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		const q = `INSERT INTO clips (clip_id, org_id, location_id, camera_id, s3_bucket, s3_key, duration_seconds, starts_at, retention_tier)
			VALUES ($1, current_setting('app.current_org_id')::uuid, $2, $3, $4, $5, $6, $7, $8) RETURNING created_at`
		return tx.QueryRow(ctx, q, c.ClipID, c.LocationID, c.CameraID, c.S3Bucket, c.S3Key, c.DurationSeconds, c.StartsAt, string(c.RetentionTier)).Scan(&c.CreatedAt)
	})
	if err != nil {
		return Clip{}, fmt.Errorf("vision: insert clip: %w", err)
	}
	return c, nil
}

func getClip(ctx context.Context, pool *pgxpool.Pool, clipID string) (Clip, error) {
	if pool == nil {
		return Clip{}, ErrClipNotFound
	}
	id, err := uuid.Parse(clipID)
	if err != nil {
		return Clip{}, fmt.Errorf("vision: invalid clip_id: %w", err)
	}
	var c Clip
	err = database.TenantTx(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT clip_id, org_id, location_id, camera_id, s3_bucket, s3_key, duration_seconds, starts_at, retention_tier, created_at FROM clips WHERE clip_id = $1`, id).Scan(&c.ClipID, &c.OrgID, &c.LocationID, &c.CameraID, &c.S3Bucket, &c.S3Key, &c.DurationSeconds, &c.StartsAt, &c.RetentionTier, &c.CreatedAt)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Clip{}, ErrClipNotFound
	}
	if err != nil {
		return Clip{}, fmt.Errorf("vision: get clip: %w", err)
	}
	return c, nil
}

func listClips(ctx context.Context, pool *pgxpool.Pool, cameraID string, limit int) ([]Clip, error) {
	if pool == nil {
		return nil, ErrClipNotFound
	}
	if limit <= 0 {
		limit = 50
	}
	camID, err := uuid.Parse(cameraID)
	if err != nil {
		return nil, fmt.Errorf("vision: invalid camera_id: %w", err)
	}
	var out []Clip
	err = database.TenantTx(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT clip_id, org_id, location_id, camera_id, s3_bucket, s3_key, duration_seconds, starts_at, retention_tier, created_at FROM clips WHERE camera_id = $1 ORDER BY created_at DESC LIMIT $2`, camID, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var c Clip
			if err := rows.Scan(&c.ClipID, &c.OrgID, &c.LocationID, &c.CameraID, &c.S3Bucket, &c.S3Key, &c.DurationSeconds, &c.StartsAt, &c.RetentionTier, &c.CreatedAt); err != nil {
				return err
			}
			out = append(out, c)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("vision: list clips: %w", err)
	}
	return out, nil
}

func updateRetentionTier(ctx context.Context, pool *pgxpool.Pool, clipID string, tier RetentionTier) error {
	if pool == nil {
		return ErrClipNotFound
	}
	id, err := uuid.Parse(clipID)
	if err != nil {
		return fmt.Errorf("vision: invalid clip_id: %w", err)
	}
	return database.TenantTx(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		cmd, err := tx.Exec(ctx, `UPDATE clips SET retention_tier = $2 WHERE clip_id = $1`, id, string(tier))
		if err != nil {
			return err
		}
		if cmd.RowsAffected() == 0 {
			return ErrClipNotFound
		}
		return nil
	})
}