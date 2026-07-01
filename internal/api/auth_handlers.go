// Package api — auth_handlers.go contains authentication-related HTTP handlers.
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/sambawy01/Retail-Compliance-System/internal/tenant"
	"github.com/sambawy01/Retail-Compliance-System/pkg/database"
	"github.com/jackc/pgx/v5"
)

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "watch-dog",
		"version": "0.1.0",
	})
}

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Email == "" || body.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password required")
		return
	}

	// Check account lockout
	if s.loginTracker != nil && s.loginTracker.IsLocked(body.Email) {
		writeError(w, http.StatusTooManyRequests, "account temporarily locked due to too many failed attempts")
		return
	}

	ctx := context.Background()
	var userID, dbOrgID, role, displayName, dbPasswordHash string
	err := s.pool.QueryRow(ctx,
		`SELECT user_id, org_id, role, display_name, password_hash FROM fn_login_lookup($1)`,
		body.Email,
	).Scan(&userID, &dbOrgID, &role, &displayName, &dbPasswordHash)
	if err != nil {
		slog.Error("login_lookup_failed", "error", err, "email", body.Email)
		if s.loginTracker != nil { s.loginTracker.RecordFailure(body.Email) }
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcryptCompare([]byte(dbPasswordHash), []byte(body.Password)); err != nil {
		slog.Error("login_bcrypt_failed", "email", body.Email, "error", err)
		if s.loginTracker != nil { s.loginTracker.RecordFailure(body.Email) }
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Login successful — reset failure counter
	if s.loginTracker != nil { s.loginTracker.RecordSuccess(body.Email) }

	if s.auth == nil {
		writeError(w, http.StatusServiceUnavailable, "authentication service unavailable")
		return
	}
	token, err := s.auth.GenerateToken(userID, dbOrgID, role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "watchdog_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		MaxAge:   86400,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user": map[string]any{
			"user_id":      userID,
			"email":        body.Email,
			"display_name": displayName,
			"role":         role,
			"org_id":       dbOrgID,
		},
	})
}

func (s *Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "watchdog_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) meHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(userCtxKey{}).(string)
	role, _ := r.Context().Value(roleCtxKey{}).(string)
	orgID, err := tenant.OrgIDFrom(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid context")
		return
	}
	var email, displayName string
	err = database.TenantTx(r.Context(), s.pool, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT email, display_name FROM users WHERE user_id = $1`, userID,
		).Scan(&email, &displayName)
	})
	if err != nil {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"user_id":      userID,
			"email":        email,
			"display_name": displayName,
			"role":         role,
			"org_id":       orgID.String(),
		},
	})
}

func (s *Server) refreshHandler(w http.ResponseWriter, r *http.Request) {
	var token string
	if cookie, err := r.Cookie("watchdog_session"); err == nil && cookie.Value != "" {
		token = cookie.Value
	} else {
		header := r.Header.Get("Authorization")
		if strings.HasPrefix(header, "Bearer ") {
			token = strings.TrimPrefix(header, "Bearer ")
		}
	}
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing authentication token")
		return
	}
	if s.auth == nil {
		writeError(w, http.StatusServiceUnavailable, "authentication service unavailable")
		return
	}
	claims, err := s.auth.ValidateToken(token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}
	newToken, err := s.auth.GenerateToken(claims.UserID, claims.OrgID, claims.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "watchdog_session",
		Value:    newToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		MaxAge:   86400,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"token": newToken,
		"user": map[string]any{
			"user_id": claims.UserID,
			"role":    claims.Role,
			"org_id":  claims.OrgID,
		},
	})
}