// Package api — webrtc_handlers.go contains WebRTC signaling HTTP handlers.
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sambawy01/Retail-Compliance-System/internal/webrtc"
)

func (s *Server) webrtcOffer(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CameraID string `json:"camera_id"`
		SDPOffer string `json:"sdp_offer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.CameraID == "" || body.SDPOffer == "" {
		writeError(w, http.StatusBadRequest, "camera_id and sdp_offer are required")
		return
	}
	userID, _ := r.Context().Value(userCtxKey{}).(string)
	sess, err := s.signaling.CreateSession(r.Context(), body.CameraID, userID, body.SDPOffer)
	if err != nil {
		slog.Error("webrtc_create_session_failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	writeJSON(w, http.StatusCreated, sess)
}

func (s *Server) webrtcAnswer(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SessionID  string `json:"session_id"`
		SDPAnswer  string `json:"sdp_answer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.SessionID == "" || body.SDPAnswer == "" {
		writeError(w, http.StatusBadRequest, "session_id and sdp_answer are required")
		return
	}
	if err := s.signaling.SetAnswer(r.Context(), body.SessionID, body.SDPAnswer); err != nil {
		if errors.Is(err, webrtc.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to set answer")
		return
	}
	sess, err := s.signaling.GetSession(r.Context(), body.SessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get session")
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (s *Server) webrtcICE(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SessionID    string `json:"session_id"`
		Candidate    string `json:"candidate"`
		SDPMLineIndex int   `json:"sdp_mline_index"`
		SDPMid       string `json:"sdp_mid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.SessionID == "" || body.Candidate == "" {
		writeError(w, http.StatusBadRequest, "session_id and candidate are required")
		return
	}
	if err := s.signaling.AddICECandidate(r.Context(), body.SessionID, webrtc.ICECandidate{
		Candidate:     body.Candidate,
		SDPMLineIndex: body.SDPMLineIndex,
		SDPMid:       body.SDPMid,
	}); err != nil {
		if errors.Is(err, webrtc.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to add ICE candidate")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getTurnCredentials(w http.ResponseWriter, r *http.Request) {
	cred, err := s.signaling.GenerateTURNCredential(r.Context(), "", 24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate TURN credentials")
		return
	}
	writeJSON(w, http.StatusOK, cred)
}