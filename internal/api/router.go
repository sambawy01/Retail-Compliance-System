// Package api provides the HTTP REST API for Watch Dog.
// Endpoints are under /api/v1/ and use JWT auth middleware.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sambawy01/Retail-Compliance-System/internal/auth"
	"github.com/sambawy01/Retail-Compliance-System/internal/event"
	"github.com/sambawy01/Retail-Compliance-System/internal/identity"
	"github.com/sambawy01/Retail-Compliance-System/internal/tenant"
	"github.com/sambawy01/Retail-Compliance-System/internal/vision"
	"github.com/sambawy01/Retail-Compliance-System/internal/webrtc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sambawy01/Retail-Compliance-System/pkg/database"
)

// Server holds all services and wires the HTTP routes.
type Server struct {
	pool       *pgxpool.Pool
	bus        *event.Bus
	vision     *vision.Service
	identity   *identity.Service
	auth       *auth.Service
	signaling  *webrtc.SignalingServer
	cfg        APIConfig
}

// APIConfig holds API-level configuration.
type APIConfig struct {
	AllowedOrigins string
}

// NewServer creates the API server with all dependencies.
func NewServer(pool *pgxpool.Pool, bus *event.Bus, vs *vision.Service, ids *identity.Service, authSvc *auth.Service, sig *webrtc.SignalingServer, cfg APIConfig) *Server {
	return &Server{
		pool:      pool,
		bus:       bus,
		vision:    vs,
		identity:  ids,
		auth:      authSvc,
		signaling: sig,
		cfg:       cfg,
	}
}

// Router returns the full chi router with all routes wired.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(s.corsMiddleware)

	// Health — no auth
	r.Get("/health", s.healthHandler)

	// Auth — no auth required for login
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/login", s.loginHandler)
		r.Post("/logout", s.logoutHandler)
		r.Post("/refresh", s.refreshHandler)
	})
	// /me endpoint — requires auth, returns current user info
	r.With(s.authMiddleware).Get("/api/v1/auth/me", s.meHandler)

	// Everything else requires JWT auth
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)

		// Cameras
		r.Route("/api/v1/vision/cameras", func(r chi.Router) {
			r.Get("/", s.listCameras)
			r.With(s.requireRole("owner", "admin", "manager")).Post("/", s.createCamera)
			r.Get("/{cameraID}", s.getCamera)
			r.With(s.requireRole("owner", "admin", "manager")).Patch("/{cameraID}", s.updateCamera)
			r.With(s.requireRole("owner", "admin")).Delete("/{cameraID}", s.deleteCamera)
			r.Post("/{cameraID}/heartbeat", s.cameraHeartbeat)
		})

		// Zones
		r.Route("/api/v1/vision/zones", func(r chi.Router) {
			r.Get("/", s.listZones)
			r.With(s.requireRole("owner", "admin", "manager")).Post("/", s.createZone)
			r.With(s.requireRole("owner", "admin")).Delete("/{zoneID}", s.deleteZone)
		})

		// Detections
		r.Route("/api/v1/vision/detections", func(r chi.Router) {
			r.Get("/", s.listDetections)
			r.Post("/", s.insertDetection)
		})

		// Clips
		r.Route("/api/v1/vision/clips", func(r chi.Router) {
			r.Get("/", s.listClips)
			r.Post("/", s.insertClip)
			r.Get("/{clipID}", s.getClip)
			r.Get("/{clipID}/url", s.getClipURL)
		})

		// Identity / Face ID
		r.Route("/api/v1/identity", func(r chi.Router) {
			r.Get("/persons", s.listPersons)
			r.With(s.requireRole("owner", "admin", "manager")).Post("/persons", s.enrollPerson)
			r.Get("/persons/{personID}", s.getPerson)
			r.With(s.requireRole("owner", "admin")).Delete("/persons/{personID}", s.revokePerson)
			r.With(s.requireRole("owner", "admin", "manager")).Post("/persons/{personID}/consent", s.recordConsent)
			r.With(s.requireRole("owner", "admin", "manager")).Post("/persons/{personID}/templates", s.insertTemplate)
			r.Post("/match", s.matchFace)
			r.Get("/audit", s.listAuditLog)
		})

		// WebRTC signaling
		r.Route("/api/v1/webrtc", func(r chi.Router) {
			r.Post("/offer", s.webrtcOffer)
			r.Post("/answer", s.webrtcAnswer)
			r.Post("/ice", s.webrtcICE)
			r.Post("/turn", s.getTurnCredentials)
		})

		// Notifications
		r.Route("/api/v1/notifications", func(r chi.Router) {
			r.Get("/rules", s.listNotificationRules)
			r.Post("/rules", s.createNotificationRule)
		})
	})

	return r
}

// --- Middleware ---

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := s.cfg.AllowedOrigins
		if allowed == "" || allowed == "*" {
			// In credentials mode, we can't use wildcard. Reflect the origin
			// if it's present, otherwise allow all (no credentials).
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			} else {
				// No Origin header — not a CORS request. Skip Allow-Origin entirely
				// to avoid returning "*" alongside Allow-Credentials: true (spec violation).
			}
		} else {
			// Specific origins configured — always set the first one.
			// If Origin header matches, reflect it. Otherwise set the configured origin.
			matched := false
			for _, o := range strings.Split(allowed, ",") {
				if strings.TrimSpace(o) == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
					matched = true
					break
				}
			}
			if !matched {
				// No Origin header or not in list — set the configured origin
				w.Header().Set("Access-Control-Allow-Origin", strings.TrimSpace(strings.Split(allowed, ",")[0]))
			}
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "3600")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read token from httpOnly cookie first, then fall back to Bearer header
		// (Bearer header is for edge agent / API clients that can't use cookies)
		var token string
		if cookie, err := r.Cookie("watchdog_session"); err == nil && cookie.Value != "" {
			token = cookie.Value
		} else {
			header := r.Header.Get("Authorization")
			if strings.HasPrefix(header, "Bearer ") {
				token = strings.TrimPrefix(header, "Bearer ")
			}
		}
		if token == "" {
			writeError(w, http.StatusUnauthorized, "missing or invalid authentication")
			return
		}
		if s.auth == nil {
			slog.Error("auth service not initialized — rejecting request")
			writeError(w, http.StatusServiceUnavailable, "authentication service unavailable")
			return
		}
		claims, err := s.auth.ValidateToken(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		// Inject org_id, user_id, and role into context for RLS + RBAC
		ctx, err := tenant.WithOrgIDString(r.Context(), claims.OrgID)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid org_id in token")
			return
		}
		ctx = context.WithValue(ctx, userCtxKey{}, claims.UserID)
		ctx = context.WithValue(ctx, roleCtxKey{}, claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type userCtxKey struct{}
type roleCtxKey struct{}

// requireRole returns middleware that checks the JWT role claim.
// Pass allowed roles; if the user's role is not in the set, 403.
func (s *Server) requireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(roleCtxKey{}).(string)
			if !allowed[role] {
				writeError(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// --- Handlers ---

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "watch-dog",
		"version": "0.1.0",
	})
}

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Email == "" || body.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password required")
		return
	}

	// Use SECURITY DEFINER function to look up user by email (bypasses RLS safely)
	// Returns password_hash for Go-side bcrypt verification
	ctx := context.Background()
	var userID, dbOrgID, role, displayName, dbPasswordHash string
	err := s.pool.QueryRow(ctx,
		`SELECT user_id, org_id, role, display_name, password_hash FROM fn_login_lookup($1)`,
		body.Email,
	).Scan(&userID, &dbOrgID, &role, &displayName, &dbPasswordHash)
	if err != nil {
		writeError(w, http.StatusUnauthorized, fmt.Sprintf("lookup error: %v", err))
		return
	}

	// Verify password with bcrypt (constant-time comparison)
	if err := bcryptCompare([]byte(dbPasswordHash), []byte(body.Password)); err != nil {
		slog.Error("login_bcrypt_failed", "email", body.Email, "hash_len", len(dbPasswordHash), "error", err)
		writeError(w, http.StatusUnauthorized, "invalid credentials - bcrypt failed")
		return
	}

	// Generate JWT
	if s.auth == nil {
		writeError(w, http.StatusServiceUnavailable, "authentication service unavailable")
		return
	}
	token, err := s.auth.GenerateToken(userID, dbOrgID, role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	// Set httpOnly cookie — not accessible to JavaScript, prevents XSS token theft
	// SameSite=None + Secure=true required for cross-origin (Vercel→Railway) cookie sending.
	// Both Railway and Vercel terminate TLS at their proxy, so r.TLS is always nil and
	// X-Forwarded-Proto is unreliable on Railway's hikari proxy. Since production is
	// always HTTPS, we unconditionally set Secure=true. Without Secure, Go net/http
	// silently downgrades SameSite=None to SameSite=Lax, which browsers then withhold
	// on cross-site fetch requests — breaking the entire auth flow.
	http.SetCookie(w, &http.Cookie{
		Name:     "watchdog_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		MaxAge:   86400, // 24h
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"user_id":      userID,
			"email":        body.Email,
			"display_name": displayName,
			"role":         role,
			"org_id":       dbOrgID,
		},
	})
}

// logoutHandler clears the session cookie.
func (s *Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "watchdog_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		MaxAge:   -1, // immediately expire
	})
	w.WriteHeader(http.StatusNoContent)
}

// meHandler returns the current authenticated user's info.
// Returns the same shape as loginHandler so the frontend AuthContext
// can use the user object consistently after login and after refresh.
// Uses TenantTx so RLS policies allow reading the users table.
func (s *Server) meHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(userCtxKey{}).(string)
	role, _ := r.Context().Value(roleCtxKey{}).(string)
	orgID, err := tenant.OrgIDFrom(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid context")
		return
	}
	var email, displayName string
	err = database.TenantTx(r.Context(), s.pool, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT email, display_name FROM users WHERE user_id = $1`, userID,
		).Scan(&email, &displayName)
	})
	if err != nil {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"user_id":      userID,
			"email":        email,
			"display_name": displayName,
			"role":         role,
			"org_id":       orgID.String(),
		},
	})
}

func (s *Server) refreshHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: refresh token
	writeError(w, http.StatusNotImplemented, "refresh not yet implemented")
}

// --- Camera handlers ---

func (s *Server) listCameras(w http.ResponseWriter, r *http.Request) {
	locationID := r.URL.Query().Get("location_id")
	cams, err := s.vision.ListCameras(r.Context(), locationID)
	if err != nil {
		slog.Error("list_cameras_failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list cameras")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cameras": cams})
}

func (s *Server) createCamera(w http.ResponseWriter, r *http.Request) {
	var in vision.CreateCameraInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	cam, err := s.vision.CreateCamera(r.Context(), in)
	if err != nil {
		slog.Error("create_camera_failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create camera")
		return
	}
	writeJSON(w, http.StatusCreated, cam)
}

func (s *Server) getCamera(w http.ResponseWriter, r *http.Request) {
	cameraID := chi.URLParam(r, "cameraID")
	cam, err := s.vision.GetCamera(r.Context(), cameraID)
	if err != nil {
		writeError(w, http.StatusNotFound, "camera not found")
		return
	}
	writeJSON(w, http.StatusOK, cam)
}

func (s *Server) updateCamera(w http.ResponseWriter, r *http.Request) {
	cameraID := chi.URLParam(r, "cameraID")
	var in vision.UpdateCameraInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	cam, err := s.vision.UpdateCamera(r.Context(), cameraID, in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update camera")
		return
	}
	writeJSON(w, http.StatusOK, cam)
}

func (s *Server) deleteCamera(w http.ResponseWriter, r *http.Request) {
	cameraID := chi.URLParam(r, "cameraID")
	if err := s.vision.DeleteCamera(r.Context(), cameraID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete camera")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) cameraHeartbeat(w http.ResponseWriter, r *http.Request) {
	cameraID := chi.URLParam(r, "cameraID")
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := s.vision.UpdateCameraStatus(r.Context(), cameraID, vision.CameraStatus(body.Status)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update heartbeat")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Zone handlers ---

func (s *Server) listZones(w http.ResponseWriter, r *http.Request) {
	cameraID := r.URL.Query().Get("camera_id")
	zones, err := s.vision.ListZones(r.Context(), cameraID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list zones")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"zones": zones})
}

func (s *Server) createZone(w http.ResponseWriter, r *http.Request) {
	var in vision.CreateZoneInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	z, err := s.vision.CreateZone(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create zone")
		return
	}
	writeJSON(w, http.StatusCreated, z)
}

func (s *Server) deleteZone(w http.ResponseWriter, r *http.Request) {
	zoneID := chi.URLParam(r, "zoneID")
	if err := s.vision.DeleteZone(r.Context(), zoneID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete zone")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Detection handlers ---

func (s *Server) listDetections(w http.ResponseWriter, r *http.Request) {
	// Parse query params into filter
	// TODO: parse all filter params
	dets, err := s.vision.ListDetections(r.Context(), vision.ListDetectionsFilter{
		CameraID:  r.URL.Query().Get("camera_id"),
		EventType: r.URL.Query().Get("event_type"),
		Limit:     100,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list detections")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"detections": dets})
}

func (s *Server) insertDetection(w http.ResponseWriter, r *http.Request) {
	var in vision.InsertDetectionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	det, err := s.vision.InsertDetection(r.Context(), in)
	if err != nil {
		slog.Error("insert_detection_failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to insert detection")
		return
	}
	writeJSON(w, http.StatusCreated, det)
}

// --- Clip handlers ---

func (s *Server) listClips(w http.ResponseWriter, r *http.Request) {
	cameraID := r.URL.Query().Get("camera_id")
	clips, err := s.vision.ListClips(r.Context(), cameraID, 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list clips")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"clips": clips})
}

func (s *Server) insertClip(w http.ResponseWriter, r *http.Request) {
	var in vision.InsertClipInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	clip, err := s.vision.InsertClip(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to insert clip")
		return
	}
	writeJSON(w, http.StatusCreated, clip)
}

func (s *Server) getClip(w http.ResponseWriter, r *http.Request) {
	clipID := chi.URLParam(r, "clipID")
	clip, err := s.vision.GetClip(r.Context(), clipID)
	if err != nil {
		writeError(w, http.StatusNotFound, "clip not found")
		return
	}
	writeJSON(w, http.StatusOK, clip)
}

func (s *Server) getClipURL(w http.ResponseWriter, r *http.Request) {
	clipID := chi.URLParam(r, "clipID")
	clip, err := s.vision.GetClip(r.Context(), clipID)
	if err != nil {
		writeError(w, http.StatusNotFound, "clip not found")
		return
	}
	url, err := s.vision.GeneratePresignURL(r.Context(), clip, 15)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate URL")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

// --- Identity handlers ---

func (s *Server) listPersons(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")
	persons, err := s.identity.ListPersons(r.Context(), kind)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list persons")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"persons": persons})
}

func (s *Server) enrollPerson(w http.ResponseWriter, r *http.Request) {
	var in identity.EnrollPersonInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	person, err := s.identity.EnrollPerson(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to enroll person")
		return
	}
	writeJSON(w, http.StatusCreated, person)
}

func (s *Server) getPerson(w http.ResponseWriter, r *http.Request) {
	personID := chi.URLParam(r, "personID")
	person, err := s.identity.GetPerson(r.Context(), personID)
	if err != nil {
		writeError(w, http.StatusNotFound, "person not found")
		return
	}
	writeJSON(w, http.StatusOK, person)
}

func (s *Server) revokePerson(w http.ResponseWriter, r *http.Request) {
	personID := chi.URLParam(r, "personID")
	reason := r.URL.Query().Get("reason")
	if err := s.identity.RevokePerson(r.Context(), personID, reason); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to revoke person")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) recordConsent(w http.ResponseWriter, r *http.Request) {
	personID := chi.URLParam(r, "personID")
	var in identity.ConsentInput
	in.PersonID = personID
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := s.identity.RecordConsent(r.Context(), in); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record consent")
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) insertTemplate(w http.ResponseWriter, r *http.Request) {
	personID := chi.URLParam(r, "personID")
	var in identity.TemplateInput
	in.PersonID = personID
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := s.identity.InsertTemplate(r.Context(), in); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store template")
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) matchFace(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Embedding []float64 `json:"embedding"`
		Threshold float64   `json:"threshold"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	result, err := s.identity.MatchFace(r.Context(), in.Embedding, in.Threshold)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "match failed")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) listAuditLog(w http.ResponseWriter, r *http.Request) {
	// TODO: implement
	writeJSON(w, http.StatusOK, map[string]any{"audit": []any{}})
}

// --- WebRTC handlers ---

func (s *Server) webrtcOffer(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CameraID  string `json:"camera_id"`
		SDPOffer  string `json:"sdp_offer"`
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
	// Return the updated session so the viewer gets the answer
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
		SDPMid:        body.SDPMid,
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
	// Generate time-limited TURN credentials (24h TTL for pilot)
	cred, err := s.signaling.GenerateTURNCredential(r.Context(), "", 24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate TURN credentials")
		return
	}
	writeJSON(w, http.StatusOK, cred)
}

// --- Notification handlers ---

func (s *Server) listNotificationRules(w http.ResponseWriter, r *http.Request) {
	// TODO: implement
	writeJSON(w, http.StatusOK, map[string]any{"rules": []any{}})
}

func (s *Server) createNotificationRule(w http.ResponseWriter, r *http.Request) {
	// TODO: implement
	writeError(w, http.StatusNotImplemented, "notification rules not yet implemented")
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}