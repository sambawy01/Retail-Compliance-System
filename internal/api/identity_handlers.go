// Package api — identity_handlers.go contains identity/face recognition HTTP handlers.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sambawy01/Retail-Compliance-System/internal/identity"
	"github.com/sambawy01/Retail-Compliance-System/pkg/database"
)

func (s *Server) listPersons(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")
	persons, err := s.identity.ListPersons(r.Context(), kind)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list persons")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"persons": persons})
}

func (s *Server) enrollPerson(w http.ResponseWriter, r *http.Request) {
	var in identity.EnrollPersonInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	person, err := s.identity.EnrollPerson(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to enroll person")
		return
	}
	writeJSON(w, http.StatusCreated, person)
}

func (s *Server) getPerson(w http.ResponseWriter, r *http.Request) {
	personID := chi.URLParam(r, "personID")
	person, err := s.identity.GetPerson(r.Context(), personID)
	if errors.Is(err, identity.ErrPersonNotFound) {
		writeError(w, http.StatusNotFound, "person not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get person")
		return
	}
	writeJSON(w, http.StatusOK, person)
}

func (s *Server) revokePerson(w http.ResponseWriter, r *http.Request) {
	personID := chi.URLParam(r, "personID")
	reason := r.URL.Query().Get("reason")
	if err := s.identity.RevokePerson(r.Context(), personID, reason); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to revoke person")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) recordConsent(w http.ResponseWriter, r *http.Request) {
	personID := chi.URLParam(r, "personID")
	var in identity.ConsentInput
	in.PersonID = personID
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := s.identity.RecordConsent(r.Context(), in); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record consent")
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) getPersonConsent(w http.ResponseWriter, r *http.Request) {
	personID := chi.URLParam(r, "personID")
	var consentRecords []map[string]any
	err := database.TenantTx(r.Context(), s.pool, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT consent_id, consent_text, consent_locale, captured_by, captured_at, revoked, revoked_at, lawful_basis
			FROM identity_consents WHERE person_id = $1 ORDER BY captured_at DESC`, personID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var consentID, consentText, consentLocale, capturedBy, lawfulBasis string
			var capturedAt time.Time
			var revoked bool
			var revokedAt *time.Time
			if err := rows.Scan(&consentID, &consentText, &consentLocale, &capturedBy, &capturedAt, &revoked, &revokedAt, &lawfulBasis); err != nil {
				return err
			}
			consentRecords = append(consentRecords, map[string]any{
				"consent_id": consentID, "consent_text": consentText, "consent_locale": consentLocale,
				"captured_by": capturedBy, "captured_at": capturedAt, "revoked": revoked,
				"revoked_at": revokedAt, "lawful_basis": lawfulBasis,
			})
		}
		return rows.Err()
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get consent records")
		return
	}
	if consentRecords == nil {
		consentRecords = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, consentRecords)
}

func (s *Server) getPersonAudit(w http.ResponseWriter, r *http.Request) {
	personID := chi.URLParam(r, "personID")
	var auditRecords []map[string]any
	err := database.TenantTx(r.Context(), s.pool, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT audit_id, purpose, triggered_by, accessed_at, camera_id
			FROM identity_access_audit WHERE person_id = $1 ORDER BY accessed_at DESC LIMIT 100`, personID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var auditID, purpose, triggeredBy string
			var accessedAt time.Time
			var cameraID *uuid.UUID
			if err := rows.Scan(&auditID, &purpose, &triggeredBy, &accessedAt, &cameraID); err != nil {
				return err
			}
			auditRecords = append(auditRecords, map[string]any{
				"audit_id": auditID, "purpose": purpose, "triggered_by": triggeredBy,
				"accessed_at": accessedAt, "camera_id": cameraID,
			})
		}
		return rows.Err()
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get audit log")
		return
	}
	if auditRecords == nil {
		auditRecords = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, auditRecords)
}

func (s *Server) insertTemplate(w http.ResponseWriter, r *http.Request) {
	personID := chi.URLParam(r, "personID")
	var in identity.TemplateInput
	in.PersonID = personID
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := s.identity.InsertTemplate(r.Context(), in); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store template")
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) matchFace(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Embedding []float64 `json:"embedding"`
		Threshold float64   `json:"threshold"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	result, err := s.identity.MatchFace(r.Context(), in.Embedding, in.Threshold)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "match failed")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) listAuditLog(w http.ResponseWriter, r *http.Request) {
	// TODO: implement proper audit log listing with pagination
	writeJSON(w, http.StatusOK, map[string]any{"audit": []any{}})
}