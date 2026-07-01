// Package api — integration_test.go provides full request-to-database integration tests
// using testcontainers-go to spin up a real PostgreSQL instance with pgvector.
// These tests verify the full HTTP stack: routing, auth, RLS, and DB operations.
//
// Prerequisites: Docker must be available on the test runner.
// Run: go test -v -tags integration ./internal/api/...
//
//go:build integration

package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/sambawy01/Retail-Compliance-System/internal/auth"
	"github.com/sambawy01/Retail-Compliance-System/internal/event"
	"github.com/sambawy01/Retail-Compliance-System/internal/identity"
	"github.com/sambawy01/Retail-Compliance-System/internal/migrations"
	"github.com/sambawy01/Retail-Compliance-System/internal/notifications"
	"github.com/sambawy01/Retail-Compliance-System/internal/staff"
	"github.com/sambawy01/Retail-Compliance-System/internal/vision"
	"github.com/sambawy01/Retail-Compliance-System/internal/webrtc"
	"github.com/sambawy01/Retail-Compliance-System/pkg/database"
)

// testEnv holds the integration test environment.
type testEnv struct {
	server   *Server
	pool     *database.Pool
	authSvc  *auth.Service
	token    string
	orgID    string
	userID   string
	cleanup  func()
}

// setupTestEnv creates a test environment with a real PostgreSQL container,
// runs migrations, seeds test data, and returns a ready-to-use testEnv.
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	ctx := context.Background()

	// Start PostgreSQL container with pgvector
	pgContainer, err := postgres.Run(ctx,
		"pgvector/pgvector:pg16",
		postgres.WithDatabase("watchdog_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	// Run migrations
	migDB, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("failed to open migration db: %v", err)
	}
	if err := migrations.Run(ctx, migDB); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}
	migDB.Close()

	// Create pgxpool
	pool, err := database.NewPool(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	// Create services
	bus := event.New()
	visionSvc := vision.New(pool, bus, slog.Default())
	identitySvc := identity.New(pool, bus)
	notifSvc := notifications.New(pool)
	staffSvc := staff.New(pool)

	// Generate RSA keys for auth
	privPEM, pubPEM := generateTestKeys(t)
	authSvc, err := auth.NewFromBytes(privPEM, pubPEM)
	if err != nil {
		t.Fatalf("failed to create auth service: %v", err)
	}

	signalingServer := webrtc.New(pool)

	// Create API server
	apiServer := NewServer(pool, bus, visionSvc, identitySvc, authSvc, notifSvc, staffSvc, signalingServer, APIConfig{
		AllowedOrigins: "*",
	})

	// Seed test organization and user directly in DB (bypass RLS)
	_, err = pool.Exec(ctx, `
		INSERT INTO organizations (org_id, name, slug) VALUES ('550e8400-e29b-41d4-a716-446655440000', 'Test Org', 'test-org')
	`)
	if err != nil {
		t.Fatalf("failed to seed org: %v", err)
	}

	// Hash password
	hashed, _ := bcryptHash([]byte("TestPassword123!"))
	_, err = pool.Exec(ctx, `
		INSERT INTO users (user_id, org_id, email, password_hash, display_name, role)
		VALUES ('660e8400-e29b-41d4-a716-446655440000', '550e8400-e29b-41d4-a716-446655440000', 'test@test.com', $1, 'Test User', 'owner')
	`, string(hashed))
	if err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	// Generate auth token
	token, err := authSvc.GenerateToken("660e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440000", "owner")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	cleanup := func() {
		pool.Close()
		pgContainer.Terminate(ctx)
	}

	return &testEnv{
		server:  apiServer,
		pool:    pool,
		authSvc: authSvc,
		token:   token,
		orgID:   "550e8400-e29b-41d4-a716-446655440000",
		userID:  "660e8400-e29b-41d4-a716-446655440000",
		cleanup: cleanup,
	}
}

// request is a helper to make HTTP requests to the test server.
func (e *testEnv) request(t *testing.T, method, path string, body any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if e.token != "" {
		req.Header.Set("Authorization", "Bearer "+e.token)
	}
	rec := httptest.NewRecorder()
	e.server.Router().ServeHTTP(rec, req)
	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	return rec, resp
}

// --- Tests ---

func TestIntegration_HealthCheck(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	rec, resp := env.request(t, "GET", "/health", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("health: got %d, want 200", rec.Code)
	}
	if resp["status"] != "ok" {
		t.Errorf("health status: got %v, want 'ok'", resp["status"])
	}
}

func TestIntegration_LoginFlow(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	// Override token to test login (don't use pre-set token)
	savedToken := env.token
	env.token = ""

	// Wrong credentials
	rec, _ := env.request(t, "POST", "/api/v1/auth/login", map[string]string{
		"email": "wrong@test.com", "password": "wrong",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("login wrong creds: got %d, want 401", rec.Code)
	}

	// Correct credentials
	rec, resp := env.request(t, "POST", "/api/v1/auth/login", map[string]string{
		"email": "test@test.com", "password": "TestPassword123!",
	})
	if rec.Code != http.StatusOK {
		t.Errorf("login correct: got %d, want 200", rec.Code)
	}
	if resp["token"] == nil || resp["token"] == "" {
		t.Error("expected non-empty token")
	}
	user := resp["user"].(map[string]any)
	if user["display_name"] != "Test User" {
		t.Errorf("display_name: got %v, want 'Test User'", user["display_name"])
	}

	// Restore token for subsequent tests
	env.token = savedToken
}

func TestIntegration_AuthMe(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	rec, resp := env.request(t, "GET", "/api/v1/auth/me", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("me: got %d, want 200", rec.Code)
	}
	user := resp["user"].(map[string]any)
	if user["email"] != "test@test.com" {
		t.Errorf("email: got %v, want test@test.com", user["email"])
	}
}

func TestIntegration_AuthGuard(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	savedToken := env.token
	env.token = ""

	rec, _ := env.request(t, "GET", "/api/v1/vision/cameras", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("no auth: got %d, want 401", rec.Code)
	}

	env.token = savedToken
}

func TestIntegration_CameraCRUD(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	// Create camera
	rec, resp := env.request(t, "POST", "/api/v1/vision/cameras", map[string]string{
		"name": "Test Camera", "rtsp_url": "rtsp://test:554/stream",
	})
	if rec.Code != http.StatusCreated {
		t.Errorf("create camera: got %d, want 201", rec.Code)
	}
	cameraID := resp["camera_id"].(string)

	// Get camera
	rec, resp = env.request(t, "GET", "/api/v1/vision/cameras/"+cameraID, nil)
	if rec.Code != http.StatusOK {
		t.Errorf("get camera: got %d, want 200", rec.Code)
	}
	if resp["name"] != "Test Camera" {
		t.Errorf("camera name: got %v, want 'Test Camera'", resp["name"])
	}

	// List cameras
	rec, resp = env.request(t, "GET", "/api/v1/vision/cameras", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("list cameras: got %d, want 200", rec.Code)
	}
	cameras := resp["cameras"].([]any)
	if len(cameras) != 1 {
		t.Errorf("camera count: got %d, want 1", len(cameras))
	}

	// Update camera
	rec, resp = env.request(t, "PATCH", "/api/v1/vision/cameras/"+cameraID, map[string]string{
		"name": "Updated Camera",
	})
	if rec.Code != http.StatusOK {
		t.Errorf("update camera: got %d, want 200", rec.Code)
	}
	if resp["name"] != "Updated Camera" {
		t.Errorf("updated name: got %v, want 'Updated Camera'", resp["name"])
	}

	// Delete camera
	rec, _ = env.request(t, "DELETE", "/api/v1/vision/cameras/"+cameraID, nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("delete camera: got %d, want 204", rec.Code)
	}

	// Verify deleted
	rec, _ = env.request(t, "GET", "/api/v1/vision/cameras/"+cameraID, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("deleted camera: got %d, want 404", rec.Code)
	}
}

func TestIntegration_NotificationRules(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	// List (empty)
	rec, resp := env.request(t, "GET", "/api/v1/notifications/rules", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("list rules: got %d, want 200", rec.Code)
	}

	// Create rule
	rec, resp = env.request(t, "POST", "/api/v1/notifications/rules", map[string]string{
		"event_type": "slip_fall", "severity": "critical",
		"channel": "telegram", "target": "@alerts",
	})
	if rec.Code != http.StatusCreated {
		t.Errorf("create rule: got %d, want 201", rec.Code)
	}
	ruleID := resp["rule_id"].(string)

	// Update rule
	rec, resp = env.request(t, "PATCH", "/api/v1/notifications/rules/"+ruleID, map[string]any{
		"enabled": false,
	})
	if rec.Code != http.StatusOK {
		t.Errorf("update rule: got %d, want 200", rec.Code)
	}

	// Delete rule
	rec, _ = env.request(t, "DELETE", "/api/v1/notifications/rules/"+ruleID, nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("delete rule: got %d, want 204", rec.Code)
	}
}

func TestIntegration_IdentityEnrollment(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	// Enroll person
	rec, resp := env.request(t, "POST", "/api/v1/identity/persons", map[string]string{
		"display_name": "John Doe", "kind": "employee",
		"job_role": "Cashier", "phone": "+201001234567",
	})
	if rec.Code != http.StatusCreated {
		t.Errorf("enroll person: got %d, want 201", rec.Code)
	}
	personID := resp["person_id"].(string)

	// Get person
	rec, resp = env.request(t, "GET", "/api/v1/identity/persons/"+personID, nil)
	if rec.Code != http.StatusOK {
		t.Errorf("get person: got %d, want 200", rec.Code)
	}
	if resp["display_name"] != "John Doe" {
		t.Errorf("person name: got %v, want 'John Doe'", resp["display_name"])
	}
	if resp["job_role"] != "Cashier" {
		t.Errorf("job_role: got %v, want 'Cashier'", resp["job_role"])
	}

	// List persons
	rec, resp = env.request(t, "GET", "/api/v1/identity/persons", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("list persons: got %d, want 200", rec.Code)
	}
	persons := resp["persons"].([]any)
	if len(persons) != 1 {
		t.Errorf("person count: got %d, want 1", len(persons))
	}

	// Revoke person
	rec, _ = env.request(t, "DELETE", "/api/v1/identity/persons/"+personID, nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("revoke person: got %d, want 204", rec.Code)
	}
}

func TestIntegration_TokenRefresh(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	// Refresh with valid token
	rec, resp := env.request(t, "POST", "/api/v1/auth/refresh", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("refresh: got %d, want 200", rec.Code)
	}
	if resp["token"] == nil || resp["token"] == "" {
		t.Error("expected non-empty refreshed token")
	}

	// Refresh with invalid token
	savedToken := env.token
	env.token = "invalid.token.here"
	rec, _ = env.request(t, "POST", "/api/v1/auth/refresh", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("refresh invalid: got %d, want 401", rec.Code)
	}
	env.token = savedToken
}

func TestIntegration_StaffListing(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	// Enroll a staff member first
	env.request(t, "POST", "/api/v1/identity/persons", map[string]string{
		"display_name": "Staff Test", "kind": "employee",
		"job_role": "Manager",
	})

	// List staff
	rec, resp := env.request(t, "GET", "/api/v1/staff/", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("list staff: got %d, want 200", rec.Code)
	}
	staffList := resp["staff"].([]any)
	if len(staffList) < 1 {
		t.Errorf("staff count: got %d, want >= 1", len(staffList))
	}
}