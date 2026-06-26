package event

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestNew(t *testing.T) {
	bus := New()
	if bus == nil {
		t.Fatal("expected non-nil bus")
	}
	if len(bus.DLQ()) != 0 {
		t.Errorf("expected empty DLQ, got %v", bus.DLQ())
	}
}

func TestNew_WithMiddlewares(t *testing.T) {
	var mwCalled int32
	mw := func(h Handler) Handler {
		return func(ctx context.Context, env Envelope) error {
			atomic.AddInt32(&mwCalled, 1)
			return h(ctx, env)
		}
	}
	bus := New(mw)
	var handlerCalled int32
	bus.Subscribe("test.subject", func(ctx context.Context, env Envelope) error {
		atomic.AddInt32(&handlerCalled, 1)
		return nil
	})

	bus.Publish(context.Background(), Envelope{EventType: "test.subject"})

	if atomic.LoadInt32(&mwCalled) != 1 {
		t.Errorf("middleware called %d times, want 1", mwCalled)
	}
	if atomic.LoadInt32(&handlerCalled) != 1 {
		t.Errorf("handler called %d times, want 1", handlerCalled)
	}
}

func TestPublish_NoSubscribers(t *testing.T) {
	bus := New()
	// Should not panic
	bus.Publish(context.Background(), Envelope{EventType: "no.subscribers"})
}

func TestPublish_SingleSubscriber(t *testing.T) {
	bus := New()
	var called int32
	bus.Subscribe("vision.test.event", func(ctx context.Context, env Envelope) error {
		atomic.AddInt32(&called, 1)
		if env.EventType != "vision.test.event" {
			t.Errorf("EventType: got %q, want %q", env.EventType, "vision.test.event")
		}
		return nil
	})
	bus.Publish(context.Background(), Envelope{EventType: "vision.test.event"})
	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("handler called %d times, want 1", called)
	}
}

func TestPublish_MultipleSubscribers(t *testing.T) {
	bus := New()
	var called int32
	for i := 0; i < 3; i++ {
		bus.Subscribe("vision.multi", func(ctx context.Context, env Envelope) error {
			atomic.AddInt32(&called, 1)
			return nil
		})
	}
	bus.Publish(context.Background(), Envelope{EventType: "vision.multi"})
	if atomic.LoadInt32(&called) != 3 {
		t.Errorf("handlers called %d times, want 3", called)
	}
}

func TestPublish_HandlerError_GoesToDLQ(t *testing.T) {
	bus := New()
	errSentinel := errors.New("handler failed")
	bus.Subscribe("error.subject", func(ctx context.Context, env Envelope) error {
		return errSentinel
	})

	env := Envelope{EventType: "error.subject", EventID: "evt-123"}
	bus.Publish(context.Background(), env)

	dlq := bus.DLQ()
	if len(dlq) != 1 {
		t.Fatalf("DLQ length: got %d, want 1", len(dlq))
	}
	if dlq[0].EventID != "evt-123" {
		t.Errorf("DLQ EventID: got %q, want %q", dlq[0].EventID, "evt-123")
	}
}

func TestPublish_MixedSuccessAndError(t *testing.T) {
	bus := New()
	var successCalled int32
	var errorCalled int32

	bus.Subscribe("mixed.subject", func(ctx context.Context, env Envelope) error {
		atomic.AddInt32(&successCalled, 1)
		return nil
	})
	bus.Subscribe("mixed.subject", func(ctx context.Context, env Envelope) error {
		atomic.AddInt32(&errorCalled, 1)
		return errors.New("fail")
	})

	bus.Publish(context.Background(), Envelope{EventType: "mixed.subject"})

	if atomic.LoadInt32(&successCalled) != 1 {
		t.Errorf("success handler called %d times, want 1", successCalled)
	}
	if atomic.LoadInt32(&errorCalled) != 1 {
		t.Errorf("error handler called %d times, want 1", errorCalled)
	}
	if len(bus.DLQ()) != 1 {
		t.Errorf("DLQ length: got %d, want 1", len(bus.DLQ()))
	}
}

func TestDLQ_ReturnsCopy(t *testing.T) {
	bus := New()
	bus.Subscribe("err", func(ctx context.Context, env Envelope) error {
		return errors.New("fail")
	})
	bus.Publish(context.Background(), Envelope{EventType: "err"})

	dlq1 := bus.DLQ()
	dlq2 := bus.DLQ()
	if len(dlq1) != 1 || len(dlq2) != 1 {
		t.Fatalf("DLQ lengths: %d, %d", len(dlq1), len(dlq2))
	}
	// Modify copy, ensure original unchanged
	dlq1[0] = Envelope{EventID: "modified"}
	dlq3 := bus.DLQ()
	if dlq3[0].EventID == "modified" {
		t.Error("DLQ should return a copy")
	}
}

func TestDLQ_Cap(t *testing.T) {
	bus := New()
	bus.Subscribe("err", func(ctx context.Context, env Envelope) error {
		return errors.New("fail")
	})
	// Publish more than 1000 events
	for i := 0; i < 1100; i++ {
		bus.Publish(context.Background(), Envelope{EventType: "err"})
	}
	dlq := bus.DLQ()
	if len(dlq) > 1000 {
		t.Errorf("DLQ length: got %d, want <= 1000", len(dlq))
	}
}

func TestMatchSubject(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		subject string
		want    bool
	}{
		// Exact matches
		{"exact match", "vision.compliance.phone_usage", "vision.compliance.phone_usage", true},
		{"exact no match", "vision.compliance.phone_usage", "vision.compliance.uniform_violation", false},

		// Single token wildcard *
		{"wildcard * matches single token", "vision.*.phone_usage", "vision.compliance.phone_usage", true},
		{"wildcard * does not match multiple tokens", "vision.*.phone_usage", "vision.compliance.sub.phone_usage", false},
		{"wildcard * matches any token", "vision.*.phone_usage", "vision.security.phone_usage", true},

		// Trailing wildcard >
		{"wildcard > matches trailing tokens", "vision.>", "vision.compliance.phone_usage", true},
		{"wildcard > matches single trailing", "vision.>", "vision", true}, // > matches zero or more remaining
		{"wildcard > matches deeper nesting", "vision.compliance.>", "vision.compliance.phone_usage", true},
		{"wildcard > does not match different prefix", "vision.compliance.>", "vision.security.loitering", false},
		{"wildcard > at end matches remaining", "vision.compliance.>", "vision.compliance.a.b.c", true},

		// Mixed
		{"mixed * and exact", "vision.*.sub.>", "vision.compliance.sub.event", true},
		{"mixed * and > with no trailing", "vision.*.sub.>", "vision.compliance.sub", true}, // > matches zero remaining
		{"multiple wildcards", "*.*.phone_usage", "vision.compliance.phone_usage", true},

		// Edge cases
		{"empty pattern and subject", "", "", true},
		{"empty pattern non-empty subject", "", "vision", false},
		{"pattern longer than subject", "vision.compliance.phone_usage", "vision", false},
		{"subject longer than pattern", "vision", "vision.compliance", false},
		{"just >", ">", "anything.here", true},
		{"just > with single token", ">", "single", true},
		{"wildcard at start", "*.compliance.>", "vision.compliance.phone_usage", true},
		{"wildcard at start no match", "*.compliance.>", "alert.security.breach", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := matchSubject(tc.pattern, tc.subject)
			if got != tc.want {
				t.Errorf("matchSubject(%q, %q) = %v, want %v", tc.pattern, tc.subject, got, tc.want)
			}
		})
	}
}

func TestMatchTokens(t *testing.T) {
	tests := []struct {
		name    string
		pattern []string
		subject []string
		want    bool
	}{
		{"both empty", []string{}, []string{}, true},
		{"pattern empty subject not", []string{}, []string{"a"}, false},
		{"exact single token", []string{"a"}, []string{"a"}, true},
		{"exact single token no match", []string{"a"}, []string{"b"}, false},
		{"wildcard > matches everything after", []string{"a", ">", "b"}, []string{"a", "x", "y"}, true},
		{"wildcard > at start", []string{">"}, []string{"a", "b"}, true},
		{"wildcard * skips one", []string{"a", "*", "c"}, []string{"a", "b", "c"}, true},
		{"wildcard * does not skip two", []string{"a", "*", "c"}, []string{"a", "b", "x", "c"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := matchTokens(tc.pattern, tc.subject)
			if got != tc.want {
				t.Errorf("matchTokens(%v, %v) = %v, want %v", tc.pattern, tc.subject, got, tc.want)
			}
		})
	}
}

func TestPublish_Concurrent(t *testing.T) {
	bus := New()
	var counter int64
	bus.Subscribe("concurrent.test", func(ctx context.Context, env Envelope) error {
		atomic.AddInt64(&counter, 1)
		return nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(context.Background(), Envelope{EventType: "concurrent.test"})
		}()
	}
	wg.Wait()

	if atomic.LoadInt64(&counter) != 100 {
		t.Errorf("handler called %d times, want 100", counter)
	}
}

func TestSubscribe_Concurrent(t *testing.T) {
	bus := New()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Subscribe("concurrent.sub", func(ctx context.Context, env Envelope) error {
				return nil
			})
		}()
	}
	wg.Wait()

	// Should not panic and should deliver to all subscribers
	bus.Publish(context.Background(), Envelope{EventType: "concurrent.sub"})
}

func TestPublish_PassesEnvelope(t *testing.T) {
	bus := New()
	env := Envelope{
		EventID:       "evt-001",
		EventType:     "vision.test.pass",
		OrgID:         "org-123",
		LocationID:    "loc-456",
		Source:        "test",
		SchemaVersion: 2,
		Payload:       map[string]any{"key": "value"},
	}

	var received Envelope
	bus.Subscribe("vision.test.pass", func(ctx context.Context, env Envelope) error {
		received = env
		return nil
	})

	bus.Publish(context.Background(), env)

	if received.EventID != env.EventID {
		t.Errorf("EventID: got %q, want %q", received.EventID, env.EventID)
	}
	if received.OrgID != env.OrgID {
		t.Errorf("OrgID: got %q, want %q", received.OrgID, env.OrgID)
	}
	if received.LocationID != env.LocationID {
		t.Errorf("LocationID: got %q, want %q", received.LocationID, env.LocationID)
	}
	if received.Source != env.Source {
		t.Errorf("Source: got %q, want %q", received.Source, env.Source)
	}
	if received.SchemaVersion != env.SchemaVersion {
		t.Errorf("SchemaVersion: got %d, want %d", received.SchemaVersion, env.SchemaVersion)
	}
}

func TestPublish_ContextPropagation(t *testing.T) {
	bus := New()
	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "test-value")

	var gotCtx context.Context
	bus.Subscribe("ctx.test", func(ctx context.Context, env Envelope) error {
		gotCtx = ctx
		return nil
	})
	bus.Publish(ctx, Envelope{EventType: "ctx.test"})

	if gotCtx == nil {
		t.Fatal("expected non-nil context")
	}
	if v := gotCtx.Value(ctxKey{}); v != "test-value" {
		t.Errorf("context value: got %v, want %v", v, "test-value")
	}
}
