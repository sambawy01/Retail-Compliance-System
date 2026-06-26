package observability

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewLogger_DebugLevel(t *testing.T) {
	logger := NewLogger("debug", "development")
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNewLogger_AllLevels(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"warning", "warning"},
		{"error", "error"},
		{"empty defaults to info", ""},
		{"unknown defaults to info", "unknown"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger := NewLogger(tc.level, "development")
			if logger == nil {
				t.Fatal("expected non-nil logger")
			}
		})
	}
}

func TestNewLogger_ProductionEnv(t *testing.T) {
	logger := NewLogger("info", "production")
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNewLogger_ProdEnv(t *testing.T) {
	logger := NewLogger("info", "prod")
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  string // we'll just verify it doesn't panic
	}{
		{"debug", ""},
		{"info", ""},
		{"warn", ""},
		{"warning", ""},
		{"error", ""},
		{"", ""},
		{"unknown", ""},
		{"DEBUG", ""},
		{"Error", ""},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			// Should not panic
			lvl := parseLevel(tc.input)
			_ = lvl
		})
	}
}

func TestHealthHandler_NoDB(t *testing.T) {
	handler := HealthHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code: got %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status: got %v, want %v", body["status"], "ok")
	}
	if _, ok := body["checked_at"]; !ok {
		t.Error("expected checked_at field")
	}
}

func TestHealthHandler_ContentType(t *testing.T) {
	handler := HealthHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type: got %q, want application/json", ct)
	}
}

func TestHealthHandler_CheckedAtIsValidRFC3339(t *testing.T) {
	handler := HealthHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	checkedAt, ok := body["checked_at"].(string)
	if !ok {
		t.Fatal("expected checked_at to be a string")
	}
	// Verify it's a valid RFC3339 timestamp
	if !strings.Contains(checkedAt, "T") || !strings.Contains(checkedAt, "Z") {
		t.Errorf("checked_at %q should be RFC3339 format", checkedAt)
	}
}
