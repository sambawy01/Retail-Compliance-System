// Package event provides the in-process event bus with NATS-compatible
// subject naming. Subscribers can wildcard with * (single token) and >
// (trailing tokens).
package event

import (
	"context"
	"log/slog"
	"strings"
	"sync"
)

// Envelope wraps every event with metadata for tracing and tenant scoping.
type Envelope struct {
	EventID       string `json:"event_id"`
	EventType     string `json:"event_type"`     // NATS-compatible subject: "vision.compliance.phone_usage"
	OrgID         string `json:"org_id"`         // tenant scope
	LocationID    string `json:"location_id"`    // optional location scope
	Source        string `json:"source"`         // originating module/agent
	SchemaVersion int   `json:"schema_version"`  // payload schema version
	Payload       any   `json:"payload"`         // event-specific data
}

// Handler processes an event envelope. Return error to send to dead letter queue.
type Handler func(ctx context.Context, env Envelope) error

type subscription struct {
	subject string
	handler Handler
}

// Bus is an in-process event bus with NATS-compatible subject naming.
type Bus struct {
	mu          sync.RWMutex
	subs        []subscription
	dlq         []Envelope
	dlqMu       sync.Mutex
	middlewares []Middleware
}

// Middleware wraps a handler with cross-cutting concerns (logging, metrics, etc).
type Middleware func(Handler) Handler

// New creates a new event bus.
func New(middlewares ...Middleware) *Bus {
	return &Bus{middlewares: middlewares}
}

// Subscribe registers a handler for a subject pattern.
func (b *Bus) Subscribe(subject string, h Handler) {
	wrapped := h
	for i := len(b.middlewares) - 1; i >= 0; i-- {
		wrapped = b.middlewares[i](wrapped)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs = append(b.subs, subscription{subject: subject, handler: wrapped})
}

// Publish dispatches an envelope to all matching subscribers.
func (b *Bus) Publish(ctx context.Context, env Envelope) {
	b.mu.RLock()
	subs := make([]subscription, len(b.subs))
	copy(subs, b.subs)
	b.mu.RUnlock()

	for _, sub := range subs {
		if matchSubject(sub.subject, env.EventType) {
			if err := sub.handler(ctx, env); err != nil {
				slog.Error("event_handler_failed",
					"subject", env.EventType,
					"error", err)
				b.dlqMu.Lock()
				b.dlq = append(b.dlq, env)
				if len(b.dlq) > 1000 {
					b.dlq = b.dlq[1:]
				}
				b.dlqMu.Unlock()
			}
		}
	}
}

// DLQ returns the dead letter queue contents.
func (b *Bus) DLQ() []Envelope {
	b.dlqMu.Lock()
	defer b.dlqMu.Unlock()
	cp := make([]Envelope, len(b.dlq))
	copy(cp, b.dlq)
	return cp
}

// matchSubject checks if a published subject matches a subscription pattern.
func matchSubject(pattern, subject string) bool {
	pTokens := strings.Split(pattern, ".")
	sTokens := strings.Split(subject, ".")
	return matchTokens(pTokens, sTokens)
}

func matchTokens(pattern, subject []string) bool {
	for i, p := range pattern {
		if p == ">" {
			return true // matches remaining
		}
		if i >= len(subject) {
			return false
		}
		if p == "*" {
			continue
		}
		if p != subject[i] {
			return false
		}
	}
	return len(pattern) == len(subject)
}