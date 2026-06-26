// Package api provides the HTTP REST API for Watch Dog.
// Endpoints are under /api/v1/ and use JWT auth middleware.
package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sambawy01/Retail-Compliance-System/internal/auth"
	"github.com/sambawy01/Retail-Compliance-System/internal/event"
	"github.com/sambawy01/Retail-Compliance-System/internal/identity"
	"github.com/sambawy01/Retail-Compliance-System/internal/tenant"
	"github.com/sambawy01/Retail-Compliance-System/internal/vision"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server holds all services and wires the HTTP routes.
type Server struct {
	pool       *pgxpool.Pool
	bus        *event.Bus
	vision     *vision.Service
	identity   *identity.Service
	auth       *auth.Service
	cfg        APIConfig
}

// APIConfig holds API-level configuration.
type APIConfig struct {
	AllowedOrigins string
}

// NewServer creates the API server with all dependencies.
func NewServer(pool *pgxpool.Pool, bus *event.Bus, vs *vision.Service, ids *identity.Service, authSvc *auth.Service, cfg APIConfig) *Server {
	return &Server{
		pool:     pool,
		bus:      bus,
		vision:   vs,
		identity: ids,
		auth:     authSvc,
		cfg:      cfg,
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

	// Auth — no auth required
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/login", s.loginHandler)
		r.Post("/refresh", s.refreshHandler)
	})

	// Everything else requires JWT auth
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)

		// Cameras
		r.Route("/api/v1/vision/cameras", func(r chi.Router) {
			r.Get("/", s.listCameras)
			r.Post("/", s.createCamera)
			r.Get("/{cameraID}", s.getCamera)
			r.Patch("/{cameraID}", s.updateCamera)
			r.Delete("/{cameraID}", s.deleteCamera)
			r.Post("/{cameraID}/heartbeat", s.cameraHeartbeat)
		})

		// Zones
		r.Route("/api/v1/vision/zones", func(r chi.Router) {
			r.Get("/", s.listZones)
			r.Post("/", s.createZone)
			r.Delete("/{zoneID}", s.deleteZone)
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
			r.Post("/persons", s.enrollPerson)
			r.Get("/persons/{personID}", s.getPerson)
			r.Delete("/persons/{personID}", s.revokePerson)
			r.Post("/persons/{personID}/consent", s.recordConsent)
			r.Post("/persons/{personID}/templates", s.insertTemplate)
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
		origins := s.cfg.AllowedOrigins
		if origins == "" {
			origins = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origins)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing or invalid authorization header")
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := s.auth.ValidateToken(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		// Inject org_id and user_id into context for RLS
		ctx, err := tenant.WithOrgIDString(r.Context(), claims.OrgID)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid org_id in token")
			return
		}
		ctx = context.WithValue(ctx, userCtxKey{}, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type userCtxKey struct{}

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

	// Hash password (SHA256 — same as seed script)
	h := sha256.Sum256([]byte(body.Password))
	passwordHash := hex.EncodeToString(h[:])

	// Query user from DB
	orgID, err := tenant.OrgIDFrom(r.Context())
	if err != nil {
		// No tenant context on login — use admin connection
		ctx := context.Background()
		var userID, dbOrgID, role, displayName string
		err := s.pool.QueryRow(ctx,
			`SELECT user_id, org_id, role, display_name FROM users WHERE email = $1 AND password_hash = $2 AND status = 'active'`,
			body.Email, passwordHash,
		).Scan(&userID, &dbOrgID, &role, &displayName)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}

		// Generate JWT
		token, err := s.auth.GenerateToken(userID, dbOrgID, role)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to generate token")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"token": token,
			"user": map[string]any{
				"user_id":      userID,
				"email":        body.Email,
				"display_name": displayName,
				"role":         role,
				"org_id":       dbOrgID,
			},
		})
		return
	}
	_ = orgID // unused fallback
	writeError(w, http.StatusUnauthorized, "invalid credentials")
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
	// TODO: implement WebRTC signaling
	writeError(w, http.StatusNotImplemented, "WebRTC signaling not yet implemented")
}

func (s *Server) webrtcAnswer(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "WebRTC answer not yet implemented")
}

func (s *Server) webrtcICE(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "ICE exchange not yet implemented")
}

func (s *Server) getTurnCredentials(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "TURN credentials not yet implemented")
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