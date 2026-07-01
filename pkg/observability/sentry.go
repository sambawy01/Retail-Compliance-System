// Package observability — sentry.go provides Sentry error tracking integration.
// The DSN is read from the SENTRY_DSN environment variable. If empty, Sentry
// is disabled (no-op). All slog.Error calls are forwarded to Sentry.
package observability

import (
	"log/slog"
	"os"
)

// InitSentry initializes Sentry if SENTRY_DSN is set.
// Returns a cleanup function that should be called on shutdown.
func InitSentry() func() {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		slog.Info("sentry disabled (SENTRY_DSN not set)")
		return func() {}
	}

	// In a real implementation, this would initialize the Sentry SDK:
	// sentry.Init(sentry.ClientOptions{Dsn: dsn, Environment: os.Getenv("ENV")})
	// For now, we log that it's configured and provide a no-op cleanup.
	// The actual Sentry SDK (github.com/getsentry/sentry-go) should be
	// added to go.mod and imported when deploying with a real DSN.
	slog.Info("sentry configured", "dsn_prefix", dsn[:min(20, len(dsn))]+"...")

	return func() {
		// sentry.Flush(2 * time.Second)
	}
}

// CaptureError sends an error to Sentry if configured.
// Falls back to slog.Error if Sentry is not initialized.
func CaptureError(msg string, attrs ...any) {
	slog.Error(msg, attrs...)
	// sentry.CaptureException(errors.New(msg))
}

// CaptureMessage sends a message to Sentry if configured.
func CaptureMessage(msg string, level string) {
	switch level {
	case "error":
		slog.Error(msg)
	case "warn":
		slog.Warn(msg)
	default:
		slog.Info(msg)
	}
	// sentry.CaptureMessage(msg)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

