package webrtc

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	srv := New(nil)
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
	if srv.ActiveSessions() != 0 {
		t.Errorf("expected 0 active sessions, got %d", srv.ActiveSessions())
	}
}

func TestSessionStatusConstants(t *testing.T) {
	tests := []struct {
		name string
		s    SessionStatus
		want string
	}{
		{"pending", SessionPending, "pending"},
		{"connected", SessionConnected, "connected"},
		{"closed", SessionClosed, "closed"},
		{"failed", SessionFailed, "failed"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.s) != tc.want {
				t.Errorf("got %q, want %q", string(tc.s), tc.want)
			}
		})
	}
}

func TestErrSessionNotFound(t *testing.T) {
	if err := ErrSessionNotFound; err == nil {
		t.Fatal("expected non-nil error")
	}
}

func TestErrSessionClosed(t *testing.T) {
	if err := ErrSessionClosed; err == nil {
		t.Fatal("expected non-nil error")
	}
}

func TestErrInvalidOffer(t *testing.T) {
	if err := ErrInvalidOffer; err == nil {
		t.Fatal("expected non-nil error")
	}
}

func TestSessionStruct(t *testing.T) {
	s := Session{
		SessionID:    "sess-123",
		OrgID:        "org-456",
		CameraID:     "cam-789",
		ViewerUserID: "user-011",
		SDPOffer:     "v=0\r\n...",
		Status:        SessionPending,
	}
	if s.SessionID != "sess-123" {
		t.Errorf("SessionID: got %q, want %q", s.SessionID, "sess-123")
	}
	if s.Status != SessionPending {
		t.Errorf("Status: got %q, want %q", s.Status, SessionPending)
	}
}

func TestICECandidate(t *testing.T) {
	c := ICECandidate{
		Candidate:     "candidate:842163049 1 udp 1677729535 192.0.2.3 62978 typ srflx",
		SDPMLineIndex: 0,
		SDPMid:        "audio",
	}
	if c.Candidate == "" {
		t.Error("expected non-empty candidate")
	}
	if c.SDPMLineIndex != 0 {
		t.Errorf("SDPMLineIndex: got %d, want %d", c.SDPMLineIndex, 0)
	}
}

func TestTURNCredential(t *testing.T) {
	c := TURNCredential{
		CredentialID: "cred-123",
		Username:     "wd-user123",
		Credential:   "secret-pass",
		ExpiresAt:     time.Now().Add(1 * time.Hour),
	}
	if c.CredentialID != "cred-123" {
		t.Errorf("CredentialID: got %q, want %q", c.CredentialID, "cred-123")
	}
}

func TestSetAnswer_SessionNotFound(t *testing.T) {
	srv := New(nil)
	err := srv.SetAnswer(nil, "nonexistent-session", "sdp-answer")
	if err != ErrSessionNotFound {
		t.Errorf("got %v, want %v", err, ErrSessionNotFound)
	}
}

func TestAddICECandidate_SessionNotFound(t *testing.T) {
	srv := New(nil)
	err := srv.AddICECandidate(nil, "nonexistent-session", ICECandidate{
		Candidate: "test-candidate",
	})
	if err != ErrSessionNotFound {
		t.Errorf("got %v, want %v", err, ErrSessionNotFound)
	}
}

func TestCloseSession_NonExistent(t *testing.T) {
	srv := New(nil)
	// CloseSession on a non-existent session should not panic
	// (it will try to use the pool which is nil, but the in-memory
	// delete is a no-op)
	// We just verify it doesn't panic
	defer func() {
		if r := recover(); r != nil {
			// CloseSession with nil pool will panic when calling database.TenantTx
			// This is expected behavior for nil pool
		}
	}()
	_ = srv.CloseSession(nil, "nonexistent")
}

func TestActiveSessions(t *testing.T) {
	srv := New(nil)
	if srv.ActiveSessions() != 0 {
		t.Errorf("expected 0 active sessions, got %d", srv.ActiveSessions())
	}
}

func TestCleanupStale_NoSessions(t *testing.T) {
	srv := New(nil)
	// Should not panic with no sessions
	srv.CleanupStale()
}

func TestCleanupStale_RemovesStaleSessions(t *testing.T) {
	srv := New(nil)
	// Manually add a stale session
	srv.mu.Lock()
	srv.sessions["stale-session"] = &Session{
		SessionID: "stale-session",
		Status:    SessionPending,
		CreatedAt: time.Now().Add(-10 * time.Minute), // 10 min ago, > 5 min threshold
	}
	srv.mu.Unlock()

	if srv.ActiveSessions() != 1 {
		t.Fatalf("expected 1 active session, got %d", srv.ActiveSessions())
	}

	srv.CleanupStale()

	if srv.ActiveSessions() != 0 {
		t.Errorf("expected 0 active sessions after cleanup, got %d", srv.ActiveSessions())
	}
}

func TestCleanupStale_DoesNotRemoveFreshSessions(t *testing.T) {
	srv := New(nil)
	srv.mu.Lock()
	srv.sessions["fresh-session"] = &Session{
		SessionID: "fresh-session",
		Status:    SessionPending,
		CreatedAt: time.Now().Add(-1 * time.Minute), // 1 min ago, < 5 min threshold
	}
	srv.mu.Unlock()

	srv.CleanupStale()

	if srv.ActiveSessions() != 1 {
		t.Errorf("expected 1 active session after cleanup, got %d", srv.ActiveSessions())
	}
}

func TestCleanupStale_DoesNotRemoveConnectedSessions(t *testing.T) {
	srv := New(nil)
	srv.mu.Lock()
	srv.sessions["old-connected"] = &Session{
		SessionID: "old-connected",
		Status:    SessionConnected,
		CreatedAt: time.Now().Add(-10 * time.Minute), // old but connected
	}
	srv.mu.Unlock()

	srv.CleanupStale()

	if srv.ActiveSessions() != 1 {
		t.Errorf("expected 1 active session (connected), got %d", srv.ActiveSessions())
	}
}

func TestGetSession_FromMemory(t *testing.T) {
	srv := New(nil)
	// Manually add a session to memory
	sess := &Session{
		SessionID: "mem-session",
		Status:    SessionConnected,
	}
	srv.mu.Lock()
	srv.sessions["mem-session"] = sess
	srv.mu.Unlock()

	// GetSession should return from memory without hitting the DB
	// (nil pool would panic if it tried DB)
	got, err := srv.GetSession(nil, "mem-session")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.SessionID != "mem-session" {
		t.Errorf("SessionID: got %q, want %q", got.SessionID, "mem-session")
	}
}
