// Package middleware provides reusable HTTP middleware for Watch Dog.
package middleware

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// MaxBodySize limits the size of request bodies to prevent DoS via large payloads.
// Default: 1MB for normal endpoints, 10MB for WebRTC offers (SDP can be large).
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeaders adds standard security headers to all responses.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		// CSP allows API JSON responses and the same origin for WebSocket
		w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self' wss: ws:; img-src 'self' data: blob:; media-src 'self' blob:; style-src 'self' 'unsafe-inline'; font-src 'self' https://fonts.googleapis.com https://fonts.gstatic.com")
		next.ServeHTTP(w, r)
	})
}

// RateLimiter provides per-IP rate limiting using a token bucket algorithm.
// Returns 429 Too Many Requests when the rate is exceeded.
type rateBucket struct {
	tokens   float64
	lastSeen int64 // unix timestamp
}

type RateLimiter struct {
	buckets    map[string]*rateBucket
	rate       float64 // tokens per second
	burst      int     // max tokens
}

// NewRateLimiter creates a rate limiter that allows `rate` requests per second
// with a maximum burst of `burst` requests.
func NewRateLimiter(rate float64, burst int) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*rateBucket),
		rate:    rate,
		burst:   burst,
	}
}

// Allow checks if the given key (e.g. IP address) is allowed to make a request.
func (rl *RateLimiter) Allow(key string) bool {
	b, exists := rl.buckets[key]
	if !exists {
		rl.buckets[key] = &rateBucket{
			tokens:   float64(rl.burst) - 1,
			lastSeen: timeNow(),
		}
		return true
	}

	now := timeNow()
	elapsed := float64(now - b.lastSeen)
	b.tokens += elapsed * rl.rate
	if b.tokens > float64(rl.burst) {
		b.tokens = float64(rl.burst)
	}
	b.lastSeen = now

	if b.tokens >= 1 {
		b.tokens -= 1
		return true
	}
	return false
}

// RateLimit middleware returns 429 when the rate limit is exceeded.
// The key function extracts the rate-limiting key from the request (default: client IP).
func RateLimit(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := clientIP(r)
			if !rl.Allow(key) {
				w.Header().Set("Retry-After", "60")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				io.WriteString(w, `{"error":"rate limit exceeded","code":"RATE_LIMITED"}`)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP extracts the client IP from the request, respecting X-Forwarded-For.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	// Remove port from RemoteAddr
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// timeNow returns the current Unix timestamp.
func timeNow() int64 {
	return time.Now().Unix()
}

// RequestID middleware sets a request ID in the response header and context.
// Uses chi's built-in RequestID if available, otherwise generates one.
func RequestIDInResponse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := chi.RequestIDFromContext(r.Context())
		if rid != "" {
			w.Header().Set("X-Request-ID", fmt.Sprintf("%v", rid))
		}
		next.ServeHTTP(w, r)
	})
}

// ctxKey is used for storing the rate limiter in context if needed.
type ctxKey string

const rateLimitKey ctxKey = "rate_limiter"

// WithRateLimiter stores the rate limiter in the request context.
func WithRateLimiter(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), rateLimitKey, rl)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// --- Server helper methods (called from Router()) ---

// securityHeadersMiddleware adds standard security headers.
func (s *Server) securityHeadersMiddleware(next http.Handler) http.Handler {
	return SecurityHeaders(next)
}

// requestIDResponseMiddleware sets the X-Request-ID header on responses.
func (s *Server) requestIDResponseMiddleware(next http.Handler) http.Handler {
	return RequestIDInResponse(next)
}

// maxBodyMiddleware limits request body size to 1MB by default.
// WebRTC offer endpoints can override with a larger limit.
func (s *Server) maxBodyMiddleware(next http.Handler) http.Handler {
	return MaxBodySize(1 << 20)(next) // 1MB default
}