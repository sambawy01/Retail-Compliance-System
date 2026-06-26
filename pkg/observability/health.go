// Package observability — health.go provides an HTTP health-check handler.
package observability

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Checker is a minimal ping interface implemented by *pgxpool.Pool.
type Checker interface {
	Ping(ctx context.Context) error
}

// HealthHandler returns an http.HandlerFunc that reports liveness and the
// status of optional dependencies. If db is non-nil it is pinged with a short
// timeout.
func HealthHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := http.StatusOK
		resp := map[string]any{
			"status":     "ok",
			"checked_at": time.Now().UTC().Format(time.RFC3339),
		}

		if db != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := db.Ping(ctx); err != nil {
				status = http.StatusServiceUnavailable
				resp["status"] = "degraded"
				resp["db"] = "unreachable"
			} else {
				resp["db"] = "ok"
			}
		}

		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(resp)
	}
}