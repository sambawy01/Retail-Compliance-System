package webrtc

import "errors"

var (
	// ErrSessionNotFound is returned when a WebRTC session does not exist.
	ErrSessionNotFound = errors.New("webrtc session not found")
	// ErrSessionClosed is returned when operating on a closed session.
	ErrSessionClosed = errors.New("webrtc session is closed")
	// ErrInvalidOffer is returned when the SDP offer is empty or malformed.
	ErrInvalidOffer = errors.New("invalid SDP offer")
)