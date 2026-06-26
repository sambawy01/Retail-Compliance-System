// Package identity implements the face identification service for Watch Dog.
// It manages person enrollment, biometric templates, consent records,
// access audit logs, and face matching via pgvector cosine similarity.
package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sambawy01/Retail-Compliance-System/internal/event"
	"github.com/sambawy01/Retail-Compliance-System/pkg/database"
)

// Person represents an enrolled person (employee or customer).
type Person struct {
	PersonID    string     `json:"person_id"`
	OrgID       string     `json:"org_id"`
	Kind        string     `json:"kind"` // "employee" or "customer"
	DisplayName string     `json:"display_name"`
	EnrolledAt  time.Time  `json:"enrolled_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// MatchResult holds the outcome of a face similarity search.
type MatchResult struct {
	PersonID    string  `json:"person_id"`
	DisplayName string  `json:"display_name"`
	Similarity  float64 `json:"similarity"`
	Matched     bool    `json:"matched"`
}

// EnrollPersonInput is the payload for enrolling a new person.
type EnrollPersonInput struct {
	Kind        string `json:"kind"`
	DisplayName string `json:"display_name"`
}

// ConsentInput is the payload for recording consent.
type ConsentInput struct {
	PersonID      string `json:"person_id"`
	ConsentText   string `json:"consent_text"`
	ConsentLocale string `json:"consent_locale"` // "en" or "ar"
	CapturedBy    string `json:"captured_by"`
}

// TemplateInput is the payload for storing a face embedding.
type TemplateInput struct {
	PersonID    string    `json:"person_id"`
	Embedding   []float64 `json:"embedding"`
	QualityScore float64  `json:"quality_score"`
}

// AuditInput is the payload for recording an access audit entry.
type AuditInput struct {
	PersonID   string `json:"person_id"`
	Purpose    string `json:"purpose"` // recognize, enroll, review, export, erase
	TriggeredBy string `json:"triggered_by"`
	CameraID   string `json:"camera_id,omitempty"`
}

// ErrPersonNotFound is returned when a person is not found.
var ErrPersonNotFound = errors.New("identity: person not found")

// Service provides face identity operations.
type Service struct {
	pool *database.Pool
	bus  *event.Bus
}

// New creates a new identity Service.
func New(pool *database.Pool, bus *event.Bus) *Service {
	return &Service{pool: pool, bus: bus}
}

// EnrollPerson creates a new identity_persons row.
func (s *Service) EnrollPerson(ctx context.Context, in EnrollPersonInput) (Person, error) {
	p := Person{
		PersonID:    uuid.NewString(),
		Kind:        in.Kind,
		DisplayName: in.DisplayName,
	}
	err := database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			INSERT INTO identity_persons (person_id, org_id, kind, display_name)
			VALUES ($1, current_setting('app.current_org_id')::uuid, $2, $3)
			RETURNING enrolled_at, created_at, updated_at`,
			p.PersonID, p.Kind, p.DisplayName,
		).Scan(&p.EnrolledAt, &p.CreatedAt, &p.UpdatedAt)
	})
	if err != nil {
		return Person{}, fmt.Errorf("identity: enroll person: %w", err)
	}
	return p, nil
}

// GetPerson retrieves a person by ID.
func (s *Service) GetPerson(ctx context.Context, personID string) (Person, error) {
	var p Person
	err := database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT person_id, org_id::text, kind, display_name, enrolled_at, revoked_at, created_at, updated_at
			FROM identity_persons WHERE person_id = $1`,
			personID,
		).Scan(&p.PersonID, &p.OrgID, &p.Kind, &p.DisplayName, &p.EnrolledAt, &p.RevokedAt, &p.CreatedAt, &p.UpdatedAt)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Person{}, ErrPersonNotFound
	}
	if err != nil {
		return Person{}, fmt.Errorf("identity: get person: %w", err)
	}
	return p, nil
}

// ListPersons returns all persons of a given kind (empty kind = all).
func (s *Service) ListPersons(ctx context.Context, kind string) ([]Person, error) {
	var out []Person
	err := database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		var rows pgx.Rows
		var err error
		if kind == "" {
			rows, err = tx.Query(ctx, `SELECT person_id, org_id::text, kind, display_name, enrolled_at, revoked_at, created_at, updated_at FROM identity_persons ORDER BY enrolled_at DESC`)
		} else {
			rows, err = tx.Query(ctx, `SELECT person_id, org_id::text, kind, display_name, enrolled_at, revoked_at, created_at, updated_at FROM identity_persons WHERE kind = $1 ORDER BY enrolled_at DESC`, kind)
		}
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var p Person
			if err := rows.Scan(&p.PersonID, &p.OrgID, &p.Kind, &p.DisplayName, &p.EnrolledAt, &p.RevokedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
				return err
			}
			out = append(out, p)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("identity: list persons: %w", err)
	}
	return out, nil
}

// RevokePerson marks a person as revoked.
func (s *Service) RevokePerson(ctx context.Context, personID, reason string) error {
	return database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		cmd, err := tx.Exec(ctx, `UPDATE identity_persons SET revoked_at = now(), updated_at = now() WHERE person_id = $1`, personID)
		if err != nil {
			return err
		}
		if cmd.RowsAffected() == 0 {
			return ErrPersonNotFound
		}
		return nil
	})
}

// RecordConsent inserts a consent record.
func (s *Service) RecordConsent(ctx context.Context, in ConsentInput) error {
	return database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO identity_consents (consent_id, org_id, person_id, consent_text, consent_locale, captured_by, signature_sha256)
			VALUES ($1, current_setting('app.current_org_id')::uuid, $2, $3, $4, $5, $6)`,
			uuid.NewString(), in.PersonID, in.ConsentText, in.ConsentLocale, in.CapturedBy, []byte(in.PersonID+in.ConsentText))
		return err
	})
}

// InsertTemplate stores a face embedding for a person.
func (s *Service) InsertTemplate(ctx context.Context, in TemplateInput) error {
	return database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		// Convert embedding to pgvector format
		vecStr := "["
		for i, v := range in.Embedding {
			if i > 0 {
				vecStr += ","
			}
			vecStr += fmt.Sprintf("%f", v)
		}
		vecStr += "]"
		_, err := tx.Exec(ctx, `
			INSERT INTO identity_templates (template_id, org_id, person_id, embedding, embedding_ct, embedding_nonce, kms_dek_id, quality_score)
			VALUES ($1, current_setting('app.current_org_id')::uuid, $2, $3::vector, $4, $5, $6, $7)`,
			uuid.NewString(), in.PersonID, vecStr, []byte{}, []byte{}, "default", in.QualityScore)
		return err
	})
}

// MatchFace performs a pgvector cosine similarity search.
func (s *Service) MatchFace(ctx context.Context, embedding []float64, threshold float64) (*MatchResult, error) {
	vecStr := "["
	for i, v := range embedding {
		if i > 0 {
			vecStr += ","
		}
		vecStr += fmt.Sprintf("%f", v)
	}
	vecStr += "]"

	var result MatchResult
	err := database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT p.person_id, p.display_name, 1 - (t.embedding <=> $1::vector) as similarity
			FROM identity_templates t
			JOIN identity_persons p ON p.person_id = t.person_id
			WHERE p.revoked_at IS NULL
			  AND 1 - (t.embedding <=> $1::vector) >= $2
			ORDER BY t.embedding <=> $1::vector
			LIMIT 1`,
			vecStr, threshold,
		).Scan(&result.PersonID, &result.DisplayName, &result.Similarity)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return &MatchResult{Matched: false}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("identity: match face: %w", err)
	}
	result.Matched = true
	return &result, nil
}

// AuditAccess records an access to biometric data.
func (s *Service) AuditAccess(ctx context.Context, in AuditInput) error {
	return database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		var cameraID *uuid.UUID
		if in.CameraID != "" {
			id, err := uuid.Parse(in.CameraID)
			if err == nil {
				cameraID = &id
			}
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO identity_access_audit (audit_id, org_id, person_id, purpose, triggered_by, camera_id)
			VALUES ($1, current_setting('app.current_org_id')::uuid, $2, $3, $4, $5)`,
			uuid.NewString(), in.PersonID, in.Purpose, in.TriggeredBy, cameraID)
		return err
	})
}