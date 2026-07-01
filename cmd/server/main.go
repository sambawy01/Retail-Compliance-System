// Package main is the Watch Dog server entry point. It loads configuration,
// wires up the database pool, event bus, vision and identity services, and
// serves the HTTP API with graceful shutdown on SIGTERM/SIGINT.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver for pgx

	"github.com/sambawy01/Retail-Compliance-System/internal/alerts"
	"github.com/sambawy01/Retail-Compliance-System/internal/api"
	"github.com/sambawy01/Retail-Compliance-System/internal/auth"
	"github.com/sambawy01/Retail-Compliance-System/internal/config"
	"github.com/sambawy01/Retail-Compliance-System/internal/event"
	"github.com/sambawy01/Retail-Compliance-System/internal/identity"
	"github.com/sambawy01/Retail-Compliance-System/internal/migrations"
	"github.com/sambawy01/Retail-Compliance-System/internal/notifications"
	"github.com/sambawy01/Retail-Compliance-System/internal/staff"
	"github.com/sambawy01/Retail-Compliance-System/internal/vision"
	"github.com/sambawy01/Retail-Compliance-System/internal/webrtc"
	"github.com/sambawy01/Retail-Compliance-System/pkg/database"
	"github.com/sambawy01/Retail-Compliance-System/pkg/observability"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Create logger
	logger := observability.NewLogger(cfg.LogLevel, cfg.Env)
	slog.SetDefault(logger)

	slog.Info("starting watchdog server", "port", cfg.Port, "env", cfg.Env)

	// Initialize Sentry error tracking (no-op if SENTRY_DSN not set)
	sentryCleanup := observability.InitSentry()
	defer sentryCleanup()

	// Build the database URL. If DB_PASSWORD is set, we construct the URL
	// from parts to avoid URL-encoding issues with special characters in
	// the password (pgx's URL parser can mangle !, @, etc).
	dbURL := cfg.DatabaseURL
	if cfg.DBPassword != "" {
		dbURL = fmt.Sprintf("postgresql://%s:%s@%s:%s/postgres?sslmode=require",
			cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort)
	}
	slog.Info("connecting to database", "host", cfg.DBHost, "port", cfg.DBPort, "user", cfg.DBUser)

	// Create DB pool
	poolCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := database.NewPool(poolCtx, dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	slog.Info("database connected")

	// Run auto-migrations (embedded SQL files, tracked in schema_migrations table)
	migCtx, migCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer migCancel()
	migDB, err := sql.Open("pgx", dbURL)
	if err != nil {
		slog.Error("migration_open_failed", "error", err)
	} else {
		defer migDB.Close()
		if err := migrations.Run(migCtx, migDB); err != nil {
			slog.Error("migration_failed", "error", err)
		} else {
			slog.Info("migrations complete")
		}
	}

	// Create event bus with a logging middleware
	bus := event.New(func(h event.Handler) event.Handler {
		return func(ctx context.Context, env event.Envelope) error {
			slog.Debug("event_published", "subject", env.EventType, "event_id", env.EventID)
			return h(ctx, env)
		}
	})

	// Create vision service + register handlers
	visionSvc := vision.New(pool, bus, logger)
	visionSvc.RegisterHandlers()

	// Wire B2 credentials for presigned URL generation
	vision.SetPresignerCredentials(cfg.B2KeyID, cfg.B2AppKey, cfg.B2Bucket, "")
	slog.Info("b2 presigner wired", "configured", cfg.B2KeyID != "")

	// Create identity service
	identitySvc := identity.New(pool, bus)

	// Create notifications service
	notifSvc := notifications.New(pool)

	// Create staff service
	staffSvc := staff.New(pool)

	// Create auth service — prefer base64 env vars, fall back to file paths.
	// Fail hard if auth keys are not configured: an unauthenticated server
	// is a security liability, not a convenience.
	var authSvc *auth.Service
	if cfg.JWTPrivateKeyB64 != "" && cfg.JWTPublicKeyB64 != "" {
		authSvc, err = auth.NewFromBase64(cfg.JWTPrivateKeyB64, cfg.JWTPublicKeyB64)
		if err != nil {
			return fmt.Errorf("auth: failed to init from base64 keys: %w", err)
		}
	} else if cfg.JWTPrivateKey != "" && cfg.JWTPublicKey != "" {
		authSvc, err = auth.New(cfg.JWTPrivateKey, cfg.JWTPublicKey)
		if err != nil {
			return fmt.Errorf("auth: failed to init from key files: %w", err)
		}
	} else {
		return fmt.Errorf("auth: JWT keys are required (set JWT_PRIVATE_KEY_B64/JWT_PUBLIC_KEY_B64 or JWT_PRIVATE_KEY_PATH/JWT_PUBLIC_KEY_PATH)")
	}

	// Create WebRTC signaling server
	signalingServer := webrtc.New(pool)

	// Create API server + router
	apiServer := api.NewServer(pool, bus, visionSvc, identitySvc, authSvc, notifSvc, staffSvc, signalingServer, api.APIConfig{
		AllowedOrigins: cfg.AllowedOrigins,
	})

	// Wire Telegram alerts (dry-run if not configured)
	tgSender := alerts.New(cfg.TelegramBotToken, cfg.TelegramChatID)
	tgSender.Subscribe(bus)
	slog.Info("telegram alerts wired", "configured", cfg.TelegramBotToken != "")

	handler := apiServer.Router()

	// Also expose the observability health endpoint alongside the API routes.
	mux := http.NewServeMux()
	mux.Handle("/healthz", observability.HealthHandler(pool))
	mux.Handle("/metrics", http.HandlerFunc(observability.MetricsHandler))
	mux.Handle("/", handler)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		slog.Info("http server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// Graceful shutdown on SIGTERM/SIGINT
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	select {
	case <-stop:
		slog.Info("shutdown signal received")
	case err := <-errCh:
		slog.Error("server error", "error", err)
		return err
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
		return err
	}
	slog.Info("server stopped")

	return nil
}