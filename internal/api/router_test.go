package api

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sambawy01/Retail-Compliance-System/internal/auth"
	"github.com/sambawy01/Retail-Compliance-System/internal/event"
	"github.com/sambawy01/Retail-Compliance-System/internal/identity"
	"github.com/sambawy01/Retail-Compliance-System/internal/vision"
)

// generateTestKeys creates a fresh RSA key pair for API tests.
func generateTestKeys(t *testing.T) (privPEM, pubPEM []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	privDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	privPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})

	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	pubPEM = pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	return privPEM, pubPEM
}

func TestNewServer(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil, nil, APIConfig{AllowedOrigins: "*"})
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
}

func TestHealthHandler(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil, nil, APIConfig{})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.healthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status: got %v, want %v", body["status"], "ok")
	}
	if body["service"] != "watch-dog" {
		t.Errorf("service: got %v, want %v", body["service"], "watch-dog")
	}
	if body["version"] != "0.1.0" {
		t.Errorf("version: got %v, want %v", body["version"], "0.1.0")
	}
}

func TestCORSMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins string
		wantOrigin     string
	}{
		{"custom origins", "https://example.com", "https://example.com"},
		{"empty defaults to *", "", "*"},
		{"wildcard", "*", "*"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := NewServer(nil, nil, nil, nil, nil, APIConfig{AllowedOrigins: tc.allowedOrigins})
			handler := srv.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			origin := rec.Header().Get("Access-Control-Allow-Origin")
			if origin != tc.wantOrigin {
				t.Errorf("CORS origin: got %q, want %q", origin, tc.wantOrigin)
			}
		})
	}
}

func TestCORSMiddleware_OptionsRequest(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil, nil, APIConfig{AllowedOrigins: "*"})
	handler := srv.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called for OPTIONS")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestCORSMiddleware_Headers(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil, nil, APIConfig{AllowedOrigins: "*"})
	handler := srv.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if methods := rec.Header().Get("Access-Control-Allow-Methods"); methods == "" {
		t.Error("expected non-empty Access-Control-Allow-Methods")
	}
	if headers := rec.Header().Get("Access-Control-Allow-Headers"); headers == "" {
		t.Error("expected non-empty Access-Control-Allow-Headers")
	}
}

func TestAuthMiddleware_NoAuthHeader(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil, nil, APIConfig{})
	handler := srv.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_InvalidAuthHeader(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil, nil, APIConfig{})
	handler := srv.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	tests := []struct {
		name   string
		header string
	}{
		{"no Bearer prefix", "Basic abc123"},
		{"empty bearer", "Bearer "},
		{"just Bearer", "Bearer"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", tc.header)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil, &auth.Service{}, APIConfig{})
	handler := srv.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRouter_HealthEndpoint(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil, nil, APIConfig{})
	r := srv.Router()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRouter_ProtectedEndpointRequiresAuth(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil, nil, APIConfig{})
	r := srv.Router()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vision/cameras", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRouter_LoginEndpoint_BadJSON(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil, nil, APIConfig{})
	r := srv.Router()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRouter_LoginEndpoint_EmptyFields(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil, nil, APIConfig{})
	r := srv.Router()

	body := `{"email":"","password":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRouter_LoginEndpoint_MissingFields(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil, nil, APIConfig{})
	r := srv.Router()

	tests := []struct {
		name string
		body string
	}{
		{"missing password", `{"email":"test@example.com","password":""}`},
		{"missing email", `{"email":"","password":"secret"}`},
		{"both empty", `{"email":"","password":""}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status: got %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestRouter_RefreshEndpoint(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil, nil, APIConfig{})
	r := srv.Router()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusNotImplemented)
	}
}

func TestRouter_WebRTCEndpoints_RequireAuth(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil, nil, APIConfig{})
	r := srv.Router()

	tests := []struct {
		name string
		path string
	}{
		{"offer", "/api/v1/webrtc/offer"},
		{"answer", "/api/v1/webrtc/answer"},
		{"ice", "/api/v1/webrtc/ice"},
		{"turn", "/api/v1/webrtc/turn"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status: got %d, want %d (auth required)", rec.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestRouter_NotificationRules_RequireAuth(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil, nil, APIConfig{})
	r := srv.Router()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/rules", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusTeapot, map[string]string{"key": "value"})

	if rec.Code != http.StatusTeapot {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusTeapot)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type: got %q, want application/json", ct)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["key"] != "value" {
		t.Errorf("body[key]: got %q, want %q", body["key"], "value")
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, http.StatusBadRequest, "something went wrong")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "something went wrong" {
		t.Errorf("error: got %q, want %q", body["error"], "something went wrong")
	}
}

func TestRouter_AllProtectedRoutes(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil, nil, APIConfig{})
	r := srv.Router()

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/vision/cameras"},
		{http.MethodPost, "/api/v1/vision/cameras"},
		{http.MethodGet, "/api/v1/vision/cameras/550e8400-e29b-41d4-a716-446655440000"},
		{http.MethodPatch, "/api/v1/vision/cameras/550e8400-e29b-41d4-a716-446655440000"},
		{http.MethodDelete, "/api/v1/vision/cameras/550e8400-e29b-41d4-a716-446655440000"},
		{http.MethodPost, "/api/v1/vision/cameras/550e8400-e29b-41d4-a716-446655440000/heartbeat"},
		{http.MethodGet, "/api/v1/vision/zones"},
		{http.MethodPost, "/api/v1/vision/zones"},
		{http.MethodDelete, "/api/v1/vision/zones/550e8400-e29b-41d4-a716-446655440000"},
		{http.MethodGet, "/api/v1/vision/detections"},
		{http.MethodPost, "/api/v1/vision/detections"},
		{http.MethodGet, "/api/v1/vision/clips"},
		{http.MethodPost, "/api/v1/vision/clips"},
		{http.MethodGet, "/api/v1/vision/clips/550e8400-e29b-41d4-a716-446655440000"},
		{http.MethodGet, "/api/v1/vision/clips/550e8400-e29b-41d4-a716-446655440000/url"},
		{http.MethodGet, "/api/v1/identity/persons"},
		{http.MethodPost, "/api/v1/identity/persons"},
		{http.MethodGet, "/api/v1/identity/persons/550e8400-e29b-41d4-a716-446655440000"},
		{http.MethodDelete, "/api/v1/identity/persons/550e8400-e29b-41d4-a716-446655440000"},
		{http.MethodPost, "/api/v1/identity/persons/550e8400-e29b-41d4-a716-446655440000/consent"},
		{http.MethodPost, "/api/v1/identity/persons/550e8400-e29b-41d4-a716-446655440000/templates"},
		{http.MethodPost, "/api/v1/identity/match"},
		{http.MethodGet, "/api/v1/identity/audit"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("route %s %s: expected 401, got %d", route.method, route.path, rec.Code)
			}
		})
	}
}

func TestRouter_WithValidToken(t *testing.T) {
	privPEM, pubPEM := generateTestKeys(t)
	authSvc, err := auth.NewFromBytes(privPEM, pubPEM)
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}

	bus := event.New()
	visionSvc := vision.New(nil, bus, nil)
	identitySvc := identity.New(nil, bus)

	srv := NewServer(nil, bus, visionSvc, identitySvc, authSvc, APIConfig{AllowedOrigins: "*"})
	r := srv.Router()

	token, err := authSvc.GenerateToken("user-123", "550e8400-e29b-41d4-a716-446655440000", "admin")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vision/cameras", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	// Valid token should pass auth (not 401), but handler will fail due to nil pool
	if rec.Code == http.StatusUnauthorized {
		t.Error("valid token should not return 401")
	}
}

func TestRouter_WithInvalidOrgIDInToken(t *testing.T) {
	privPEM, pubPEM := generateTestKeys(t)
	authSvc, err := auth.NewFromBytes(privPEM, pubPEM)
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}

	srv := NewServer(nil, nil, nil, nil, authSvc, APIConfig{})
	r := srv.Router()

	token, err := authSvc.GenerateToken("user-123", "not-a-uuid", "admin")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vision/cameras", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// Dummy to avoid unused import
var _ = context.Background
