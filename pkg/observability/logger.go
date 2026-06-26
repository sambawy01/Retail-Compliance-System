// Package observability provides structured logging and health-check
// helpers for the Watch Dog service.
package observability

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// NewLogger returns a slog.Logger configured at the given level writing to
// stderr in JSON format. In non-prod it uses a readable text handler.
func NewLogger(level, env string) *slog.Logger {
	lvl := parseLevel(level)
	w := io.Writer(os.Stderr)

	if env == "prod" || env == "production" {
		h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lvl})
		return slog.New(h)
	}
	h := slog.NewTextHandler(w, &slog.HandlerOptions{Level: lvl})
	return slog.New(h)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}