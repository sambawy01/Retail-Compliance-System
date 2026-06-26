// Package webrtc provides the WebRTC signaling server for live camera streaming.
// The edge agent publishes a WebRTC stream from each camera; the dashboard
// subscribes to view it. This server brokers the SDP offer/answer + ICE exchange.
package webrtc

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sambawy01/Retail-Compliance-System/internal/tenant"
	"github.com/sambawy01/Retail-Compliance-System/pkg/database"
)

// SessionStatus tracks the lifecycle of a WebRTC session.
type SessionStatus string

const (
	SessionPending   SessionStatus = "pending"
	SessionConnected  SessionStatus = "connected"
	SessionClosed     SessionStatus = "closed"
	SessionFailed     SessionStatus = "failed"
)

// Session represents an active WebRTC streaming session between the edge
// agent (publisher) and a dashboard viewer (subscriber).
type Session struct {
	SessionID     string        `json:"session_id"`
	OrgID         string        `json:"org_id"`
	CameraID      string        `json:"camera_id"`
	ViewerUserID  string        `json:"viewer_user_id,omitempty"`
	SDPOffer      string        `json:"sdp_offer"`
	SDPAnswer     string        `json:"sdp_answer,omitempty"`
	ICECandidates []ICECandidate `json:"ice_candidates,omitempty"`
	Status        SessionStatus `json:"status"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// ICECandidate is a single ICE candidate exchanged between peers.
type ICECandidate struct {
	Candidate     string `json:"candidate"`
	SDPMLineIndex int    `json:"sdp_mline_index"`
	SDPMid        string `json:"sdp_mid,omitempty"`
}

// TURNCredential is a time-limited credential for the TURN server.
type TURNCredential struct {
	CredentialID string    `json:"credential_id"`
	Username     string    `json:"username"`
	Credential   string    `json:"credential"`
	ExpiresAt    time.Time  `json:"expires_at"`
}

// SignalingServer manages WebRTC sessions in memory and persists to the DB.
type SignalingServer struct {
	pool    *pgxpool.Pool
	mu      sync.RWMutex
	sessions map[string]*Session
}

// New creates a new SignalingServer.
func New(pool *pgxpool.Pool) *SignalingServer {
	return &SignalingServer{
		pool:    pool,
		sessions: make(map[string]*Session),
	}
}

// CreateSession starts a new WebRTC session for a camera stream.
func (s *SignalingServer) CreateSession(ctx context.Context, cameraID, viewerUserID, sdpOffer string) (*Session, error) {
	orgID, err := tenant.OrgIDFrom(ctx)
	if err != nil {
		return nil, err
	}

	session := &Session{
		SessionID:    uuid.NewString(),
		OrgID:       orgID.String(),
		CameraID:     cameraID,
		ViewerUserID: viewerUserID,
		SDPOffer:     sdpOffer,
		Status:       SessionPending,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	// Persist to database
	err = database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO webrtc_sessions (session_id, org_id, camera_id, viewer_user_id, sdp_offer, status)
			VALUES ($1, $2, $3, NULLIF($4,'')::uuid, $5, $6)`,
			session.SessionID, session.OrgID, session.CameraID,
			session.ViewerUserID, session.SDPOffer, string(session.Status))
		return err
	})
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.sessions[session.SessionID] = session
	s.mu.Unlock()

	slog.Info("webrtc_session_created", "session_id", session.SessionID, "camera_id", cameraID)
	return session, nil
}

// SetAnswer stores the SDP answer from the edge agent and marks the session connected.
func (s *SignalingServer) SetAnswer(ctx context.Context, sessionID, sdpAnswer string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}

	sess.SDPAnswer = sdpAnswer
	sess.Status = SessionConnected
	sess.UpdatedAt = time.Now().UTC()

	// Persist update
	err := database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE webrtc_sessions SET sdp_answer = $2, status = $3, updated_at = now()
			WHERE session_id = $1`,
			sessionID, sdpAnswer, string(SessionConnected))
		return err
	})
	return err
}

// AddICECandidate appends an ICE candidate to the session.
func (s *SignalingServer) AddICECandidate(ctx context.Context, sessionID string, candidate ICECandidate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}

	sess.ICECandidates = append(sess.ICECandidates, candidate)
	sess.UpdatedAt = time.Now().UTC()
	return nil
}

// GetSession retrieves a session by ID (from memory if active, else from DB).
func (s *SignalingServer) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	s.mu.RLock()
	if sess, ok := s.sessions[sessionID]; ok {
		s.mu.RUnlock()
		return sess, nil
	}
	s.mu.RUnlock()

	// Fall back to database
	var sess Session
	err := database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT session_id, org_id, camera_id,
			       COALESCE(viewer_user_id::text,''), sdp_offer,
			       COALESCE(sdp_answer,''), status, created_at, updated_at
			FROM webrtc_sessions WHERE session_id = $1`,
			sessionID,
		).Scan(&sess.SessionID, &sess.OrgID, &sess.CameraID,
			&sess.ViewerUserID, &sess.SDPOffer, &sess.SDPAnswer,
			&sess.Status, &sess.CreatedAt, &sess.UpdatedAt)
	})
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

// CloseSession marks a session as closed and removes from memory.
func (s *SignalingServer) CloseSession(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[sessionID]; ok {
		delete(s.sessions, sessionID)
	}

	err := database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE webrtc_sessions SET status = $2, updated_at = now()
			WHERE session_id = $1`,
			sessionID, string(SessionClosed))
		return err
	})
	return err
}

// GenerateTURNCredential creates a time-limited TURN credential for a viewer.
func (s *SignalingServer) GenerateTURNCredential(ctx context.Context, orgID string, ttl time.Duration) (*TURNCredential, error) {
	cred := &TURNCredential{
		CredentialID: uuid.NewString(),
		Username:     "wd-" + uuid.NewString()[:8],
		Credential:   uuid.NewString(),
		ExpiresAt:    time.Now().Add(ttl).UTC(),
	}

	err := database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO webrtc_turn_credentials (credential_id, org_id, username, credential, expires_at)
			VALUES ($1, $2, $3, $4, $5)`,
			cred.CredentialID, orgID, cred.Username, cred.Credential, cred.ExpiresAt)
		return err
	})
	if err != nil {
		return nil, err
	}
	return cred, nil
}

// ActiveSessions returns the count of currently active sessions.
func (s *SignalingServer) ActiveSessions() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}

// CleanupStale removes sessions that have been pending for more than 5 minutes.
func (s *SignalingServer) CleanupStale() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-5 * time.Minute)
	for id, sess := range s.sessions {
		if sess.Status == SessionPending && sess.CreatedAt.Before(cutoff) {
			sess.Status = SessionFailed
			delete(s.sessions, id)
			slog.Info("webrtc_session_stale_removed", "session_id", id)
		}
	}
}