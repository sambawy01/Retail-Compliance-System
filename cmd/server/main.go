// Package main is the Watch Dog server entry point. It loads configuration,
// wires up the database pool, event bus, vision and identity services, and
// serves the HTTP API with graceful shutdown on SIGTERM/SIGINT.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sambawy01/Retail-Compliance-System/internal/alerts"
	"github.com/sambawy01/Retail-Compliance-System/internal/api"
	"github.com/sambawy01/Retail-Compliance-System/internal/auth"
	"github.com/sambawy01/Retail-Compliance-System/internal/config"
	"github.com/sambawy01/Retail-Compliance-System/internal/event"
	"github.com/sambawy01/Retail-Compliance-System/internal/identity"
	"github.com/sambawy01/Retail-Compliance-System/internal/vision"
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

	// Create DB pool
	poolCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := database.NewPool(poolCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	slog.Info("database connected")

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

	// Create identity service
	identitySvc := identity.New(pool, bus)

	// Create auth service — prefer base64 env vars, fall back to file paths
	var authSvc *auth.Service
	if cfg.JWTPrivateKeyB64 != "" && cfg.JWTPublicKeyB64 != "" {
		authSvc, err = auth.NewFromBase64(cfg.JWTPrivateKeyB64, cfg.JWTPublicKeyB64)
	} else if cfg.JWTPrivateKey != "" && cfg.JWTPublicKey != "" {
		authSvc, err = auth.New(cfg.JWTPrivateKey, cfg.JWTPublicKey)
	}
	if err != nil {
		slog.Warn("auth service not fully initialized", "error", err)
	}

	// Create API server + router
	apiServer := api.NewServer(pool, bus, visionSvc, identitySvc, authSvc, api.APIConfig{
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

	// Suppress unused-import lint for pgxpool (used for type inference via api/identity).
	var _ *pgxpool.Pool = pool
	return nil
}