// Package vision — vision.go wires the vision Service with its dependencies
// (database pool + event bus) and registers event handlers.
package vision

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sambawy01/Retail-Compliance-System/internal/event"
)

// Service is the core vision-pipeline service. It owns the database pool and
// event bus and registers handlers for retail compliance events.
type Service struct {
	pool *pgxpool.Pool
	bus  *event.Bus
	log  *slog.Logger
}

// New creates a vision Service bound to the given pool and bus.
func New(pool *pgxpool.Pool, bus *event.Bus, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		pool: pool,
		bus:  bus,
		log:  log,
	}
}

// Pool returns the underlying connection pool.
func (s *Service) Pool() *pgxpool.Pool { return s.pool }

// Bus returns the underlying event bus.
func (s *Service) Bus() *event.Bus { return s.bus }

// RegisterHandlers subscribes the service's handlers to all retail event
// subjects on the bus. It uses a single dispatch handler that routes by the
// envelope's EventType.
func (s *Service) RegisterHandlers() {
	if s.bus == nil {
		return
	}
	// Wildcard subscription: vision.> matches every retail subject.
	s.bus.Subscribe("vision.>", func(ctx context.Context, env event.Envelope) error {
		return s.handleEvent(ctx, env)
	})
}

// handleEvent is the central dispatch for vision events. It logs the event at
// a level derived from the resolved severity and is a hook point for
// persistence/escalation.
func (s *Service) handleEvent(ctx context.Context, env event.Envelope) error {
	sev := ResolveSeverity(env.EventType)
	switch sev {
	case SeverityCritical:
		s.log.Error("vision.critical_event",
			"subject", env.EventType,
			"org_id", env.OrgID,
			"location_id", env.LocationID,
			"severity", sev,
			"event_id", env.EventID,
		)
	case SeverityWarning:
		s.log.Warn("vision.warning_event",
			"subject", env.EventType,
			"org_id", env.OrgID,
			"location_id", env.LocationID,
			"severity", sev,
			"event_id", env.EventID,
		)
	default:
		s.log.Info("vision.info_event",
			"subject", env.EventType,
			"org_id", env.OrgID,
			"location_id", env.LocationID,
			"severity", sev,
			"event_id", env.EventID,
		)
	}
	return nil
}