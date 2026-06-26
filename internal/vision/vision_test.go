package vision

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sambawy01/Retail-Compliance-System/internal/event"
)

func TestNew_WithNilLogger(t *testing.T) {
	svc := New(nil, nil, nil)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.log == nil {
		t.Error("expected non-nil default logger")
	}
}

func TestNew_WithCustomLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(nil, nil))
	svc := New(nil, nil, logger)
	if svc.log != logger {
		t.Error("expected custom logger to be used")
	}
}

func TestNew_WithBus(t *testing.T) {
	bus := event.New()
	svc := New(nil, bus, nil)
	if svc.Bus() != bus {
		t.Error("expected bus to be stored")
	}
}

func TestNew_WithPool(t *testing.T) {
	svc := New(nil, nil, nil)
	if svc.Pool() != nil {
		t.Error("expected nil pool")
	}
}

func TestRegisterHandlers_NilBus(t *testing.T) {
	svc := New(nil, nil, nil)
	// Should not panic
	svc.RegisterHandlers()
}

func TestRegisterHandlers_Subscribes(t *testing.T) {
	bus := event.New()
	svc := New(nil, bus, nil)
	svc.RegisterHandlers()

	// Publishing a vision event should reach the handler
	// Since the handler just logs, we verify it doesn't panic
	bus.Publish(context.Background(), event.Envelope{
		EventType: EventSafetySlipFall,
		EventID:   "test-evt",
		OrgID:     "org-123",
	})
}

func TestRegisterHandlers_DispatchesBySeverity(t *testing.T) {
	// Capture log output
	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	bus := event.New()
	svc := New(nil, bus, logger)
	svc.RegisterHandlers()

	tests := []struct {
		name       string
		eventType  string
		wantLogMsg string
	}{
		{"critical", EventSafetySlipFall, "vision.critical_event"},
		{"warning", EventCompliancePhoneUsage, "vision.warning_event"},
		{"info", EventOccupancyUpdate, "vision.info_event"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf.Reset()
			bus.Publish(context.Background(), event.Envelope{
				EventType: tc.eventType,
				EventID:   "evt-" + tc.name,
			})
			output := buf.String()
			if !strings.Contains(output, tc.wantLogMsg) {
				t.Errorf("expected log message %q in output %q", tc.wantLogMsg, output)
			}
		})
	}
}

func TestHandleEvent_DoesNotReturnError(t *testing.T) {
	svc := New(nil, nil, nil)
	tests := []struct {
		name      string
		eventType string
	}{
		{"critical", EventSafetySlipFall},
		{"warning", EventCompliancePhoneUsage},
		{"info", EventOccupancyUpdate},
		{"unknown", "vision.unknown.event"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := svc.handleEvent(context.Background(), event.Envelope{
				EventType: tc.eventType,
			})
			if err != nil {
				t.Errorf("handleEvent returned error: %v", err)
			}
		})
	}
}

// --- Presign URL tests ---

func TestGeneratePresignURL(t *testing.T) {
	svc := New(nil, nil, nil)
	tests := []struct {
		name          string
		clip          Clip
		expiryMinutes int
		wantBucket    string
		wantKey       string
		wantErr       bool
		errContains   string
	}{
		{
			name: "valid clip with bucket",
			clip: Clip{
				S3Bucket: "mybucket",
				S3Key:    "clips/abc123.mp4",
			},
			expiryMinutes: 30,
			wantBucket:    "mybucket",
			wantKey:       "clips/abc123.mp4",
		},
		{
			name: "valid clip without bucket uses default",
			clip: Clip{
				S3Key: "clips/xyz.mp4",
			},
			expiryMinutes: 60,
			wantBucket:    "watchdog-clips",
			wantKey:       "clips/xyz.mp4",
		},
		{
			name: "empty S3 key returns error",
			clip: Clip{
				S3Bucket: "mybucket",
				S3Key:    "",
			},
			expiryMinutes: 30,
			wantErr:       true,
			errContains:   "clip has no S3 key",
		},
		{
			name: "zero expiry defaults to 60",
			clip: Clip{
				S3Bucket: "mybucket",
				S3Key:    "clips/test.mp4",
			},
			expiryMinutes: 0,
			wantBucket:    "mybucket",
			wantKey:       "clips/test.mp4",
		},
		{
			name: "negative expiry defaults to 60",
			clip: Clip{
				S3Bucket: "mybucket",
				S3Key:    "clips/test.mp4",
			},
			expiryMinutes: -5,
			wantBucket:    "mybucket",
			wantKey:       "clips/test.mp4",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			url, err := svc.GeneratePresignURL(context.Background(), tc.clip, tc.expiryMinutes)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("expected error containing %q, got %q", tc.errContains, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(url, tc.wantBucket) {
				t.Errorf("URL %q does not contain bucket %q", url, tc.wantBucket)
			}
			if !strings.Contains(url, tc.wantKey) {
				t.Errorf("URL %q does not contain key %q", url, tc.wantKey)
			}
			if !strings.Contains(url, "expires=") {
				t.Errorf("URL %q does not contain expires= parameter", url)
			}
			if !strings.HasPrefix(url, "https://") {
				t.Errorf("URL should start with https://, got %q", url)
			}
		})
	}
}

func TestGeneratePresignURL_URLFormat(t *testing.T) {
	svc := New(nil, nil, nil)
	clip := Clip{S3Bucket: "testbucket", S3Key: "path/to/clip.mp4"}
	url, err := svc.GeneratePresignURL(context.Background(), clip, 15)
	if err != nil {
		t.Fatalf("GeneratePresignURL: %v", err)
	}
	expectedPrefix := "https://f000.backblazeb2.com/file/testbucket/path/to/clip.mp4?expires="
	if !strings.HasPrefix(url, expectedPrefix) {
		t.Errorf("URL prefix: got %q, want prefix %q", url, expectedPrefix)
	}
}

func TestSetB2Bucket(t *testing.T) {
	original := B2BucketName
	defer func() { B2BucketName = original }()

	SetB2Bucket("custom-bucket")
	if B2BucketName != "custom-bucket" {
		t.Errorf("B2BucketName: got %q, want %q", B2BucketName, "custom-bucket")
	}

	// Verify it's used in presign URL when clip has no bucket
	svc := New(nil, nil, nil)
	clip := Clip{S3Key: "test.mp4"}
	url, err := svc.GeneratePresignURL(context.Background(), clip, 30)
	if err != nil {
		t.Fatalf("GeneratePresignURL: %v", err)
	}
	if !strings.Contains(url, "custom-bucket") {
		t.Errorf("URL %q should contain custom bucket name", url)
	}
}

// --- Clips nil-pool tests ---

func TestInsertClip_NilPool(t *testing.T) {
	svc := New(nil, nil, nil)
	_, err := svc.InsertClip(context.Background(), InsertClipInput{
		S3Bucket: "test",
		S3Key:    "key",
	})
	if err == nil {
		t.Fatal("expected error for nil pool")
	}
	if err != ErrClipNotFound {
		t.Errorf("expected ErrClipNotFound, got %v", err)
	}
}

func TestGetClip_NilPool(t *testing.T) {
	svc := New(nil, nil, nil)
	_, err := svc.GetClip(context.Background(), "550e8400-e29b-41d4-a716-446655440000")
	if err == nil {
		t.Fatal("expected error for nil pool")
	}
	if err != ErrClipNotFound {
		t.Errorf("expected ErrClipNotFound, got %v", err)
	}
}

func TestListClips_NilPool(t *testing.T) {
	svc := New(nil, nil, nil)
	_, err := svc.ListClips(context.Background(), "550e8400-e29b-41d4-a716-446655440000", 10)
	if err == nil {
		t.Fatal("expected error for nil pool")
	}
	if err != ErrClipNotFound {
		t.Errorf("expected ErrClipNotFound, got %v", err)
	}
}

func TestUpdateRetentionTier_NilPool(t *testing.T) {
	svc := New(nil, nil, nil)
	err := svc.UpdateRetentionTier(context.Background(), "550e8400-e29b-41d4-a716-446655440000", RetentionWarm)
	if err == nil {
		t.Fatal("expected error for nil pool")
	}
	if err != ErrClipNotFound {
		t.Errorf("expected ErrClipNotFound, got %v", err)
	}
}

func TestGetClip_InvalidUUID_NilPool(t *testing.T) {
	svc := New(nil, nil, nil)
	_, err := svc.GetClip(context.Background(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
}

func TestListClips_InvalidUUID_NilPool(t *testing.T) {
	svc := New(nil, nil, nil)
	_, err := svc.ListClips(context.Background(), "not-a-uuid", 10)
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
}

func TestUpdateRetentionTier_InvalidUUID_NilPool(t *testing.T) {
	svc := New(nil, nil, nil)
	err := svc.UpdateRetentionTier(context.Background(), "not-a-uuid", RetentionWarm)
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
}

// --- Vision Service constructor and accessors ---

func TestService_PoolAccessor(t *testing.T) {
	svc := New(nil, nil, nil)
	if svc.Pool() != nil {
		t.Error("expected nil pool")
	}
}

func TestService_BusAccessor(t *testing.T) {
	bus := event.New()
	svc := New(nil, bus, nil)
	if svc.Bus() != bus {
		t.Error("expected bus to match")
	}
}

// --- HTTP-level tests for presign URL ---

func TestGeneratePresignURL_HTTPContext(t *testing.T) {
	svc := New(nil, nil, nil)
	clip := Clip{S3Bucket: "test", S3Key: "clip.mp4"}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := req.Context()

	url, err := svc.GeneratePresignURL(ctx, clip, 30)
	if err != nil {
		t.Fatalf("GeneratePresignURL: %v", err)
	}
	if url == "" {
		t.Error("expected non-empty URL")
	}
}
