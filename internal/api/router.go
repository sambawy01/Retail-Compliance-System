// Package api provides the HTTP REST API for Watch Dog.
// Endpoints are under /api/v1/ and use JWT auth middleware.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sambawy01/Retail-Compliance-System/internal/auth"
	"github.com/sambawy01/Retail-Compliance-System/internal/event"
	"github.com/sambawy01/Retail-Compliance-System/internal/identity"
	"github.com/sambawy01/Retail-Compliance-System/internal/notifications"
	"github.com/sambawy01/Retail-Compliance-System/internal/staff"
	"github.com/sambawy01/Retail-Compliance-System/internal/tenant"
	"github.com/sambawy01/Retail-Compliance-System/internal/vision"
	"github.com/sambawy01/Retail-Compliance-System/internal/webrtc"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server holds all services and wires the HTTP routes.
type Server struct {
	pool          *pgxpool.Pool
	bus           *event.Bus
	vision        *vision.Service
	identity      *identity.Service
	auth          *auth.Service
	notifications *notifications.Service
	staff         *staff.Service
	signaling     *webrtc.SignalingServer
	cfg           APIConfig
}

// APIConfig holds API-level configuration.
type APIConfig struct {
	AllowedOrigins string
}

// NewServer creates the API server with all dependencies.
func NewServer(pool *pgxpool.Pool, bus *event.Bus, vs *vision.Service, ids *identity.Service, authSvc *auth.Service, notifSvc *notifications.Service, staffSvc *staff.Service, sig *webrtc.SignalingServer, cfg APIConfig) *Server {
	return &Server{
		pool:          pool,
		bus:           bus,
		vision:        vs,
		identity:      ids,
		auth:          authSvc,
		notifications: notifSvc,
		staff:         staffSvc,
		signaling:     sig,
		cfg:           cfg,
	}
}

// Router returns the full chi router with all routes wired.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(s.corsMiddleware)
	r.Use(s.securityHeadersMiddleware)
	r.Use(s.requestIDResponseMiddleware)
	r.Use(s.maxBodyMiddleware)

	// Health — no auth
	r.Get("/health", s.healthHandler)

	// Auth — no auth required for login
	loginLimiter := NewRateLimiter(0.1, 5) // 5 requests burst, refill 1 per 10s
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.With(RateLimit(loginLimiter)).Post("/login", s.loginHandler)
		r.Post("/logout", s.logoutHandler)
		r.With(RateLimit(NewRateLimiter(0.5, 10))).Post("/refresh", s.refreshHandler)
	})
	// /me endpoint — requires auth
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
			r.Get("/persons/{personID}/consent", s.getPersonConsent)
			r.With(s.requireRole("owner", "admin", "manager")).Post("/persons/{personID}/templates", s.insertTemplate)
			r.Get("/persons/{personID}/audit", s.getPersonAudit)
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
			r.With(s.requireRole("owner", "admin", "manager")).Post("/rules", s.createNotificationRule)
			r.With(s.requireRole("owner", "admin", "manager")).Patch("/rules/{ruleID}", s.updateNotificationRule)
			r.With(s.requireRole("owner", "admin")).Delete("/rules/{ruleID}", s.deleteNotificationRule)
		})

		// Staff performance & management
		r.Route("/api/v1/staff", func(r chi.Router) {
			r.Get("/", s.listStaff)
			r.Get("/{personID}", s.getStaff)
			r.Get("/{personID}/report", s.getStaffReport)
			r.Get("/{personID}/attendance", s.getStaffAttendance)
			r.With(s.requireRole("owner", "admin", "manager")).Post("/{personID}/attendance", s.recordAttendance)
			r.Get("/{personID}/holidays", s.getStaffHolidays)
			r.With(s.requireRole("owner", "admin", "manager")).Post("/{personID}/holidays", s.requestHoliday)
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
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
		} else {
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

type userCtxKey struct{}
type roleCtxKey struct{}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			writeError(w, http.StatusServiceUnavailable, "authentication service unavailable")
			return
		}
		claims, err := s.auth.ValidateToken(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
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

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}