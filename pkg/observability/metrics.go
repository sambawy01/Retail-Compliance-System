// Package observability — metrics.go provides Prometheus-compatible metrics.
// Exposes /metrics endpoint with request count, latency, and error rate.
// Uses a simple in-memory counter/histogram (no external Prometheus library
// needed — the output is Prometheus text format).
package observability

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds Prometheus-compatible metrics.
type Metrics struct {
	mu              sync.Mutex
	requestCount    map[string]*atomic.Int64 // key: method:path
	requestDuration map[string]*histogram   // key: method:path
	errorCount      atomic.Int64
	totalRequests   atomic.Int64
}

type histogram struct {
	buckets map[string]*atomic.Int64 // bucket label -> count
	sum     atomic.Int64              // sum in milliseconds
}

var globalMetrics = &Metrics{
	requestCount:    make(map[string]*atomic.Int64),
	requestDuration: make(map[string]*histogram),
}

// RecordRequest increments the request counter and records duration.
func RecordRequest(method, path string, durationMs int64, isError bool) {
	key := method + ":" + path
	globalMetrics.mu.Lock()
	counter, ok := globalMetrics.requestCount[key]
	if !ok {
		counter = &atomic.Int64{}
		globalMetrics.requestCount[key] = counter
	}
	counter.Add(1)
	globalMetrics.totalRequests.Add(1)

	hist, ok := globalMetrics.requestDuration[key]
	if !ok {
		hist = &histogram{buckets: make(map[string]*atomic.Int64)}
		globalMetrics.requestDuration[key] = hist
	}
	bucket := bucketFor(durationMs)
	b, ok := hist.buckets[bucket]
	if !ok {
		b = &atomic.Int64{}
		hist.buckets[bucket] = b
	}
	b.Add(1)
	hist.sum.Add(durationMs)
	globalMetrics.mu.Unlock()

	if isError {
		globalMetrics.errorCount.Add(1)
	}
}

func bucketFor(ms int64) string {
	switch {
	case ms < 10:
		return "10"
	case ms < 50:
		return "50"
	case ms < 100:
		return "100"
	case ms < 500:
		return "500"
	case ms < 1000:
		return "1000"
	case ms < 5000:
		return "5000"
	default:
		return "+Inf"
	}
}

// MetricsHandler returns an http.HandlerFunc that serves Prometheus text format.
func MetricsHandler(w http.ResponseWriter, r *http.Request) {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()

	var sb strings.Builder
	sb.WriteString("# TYPE watchdog_requests_total counter\n")
	for key, counter := range globalMetrics.requestCount {
		parts := strings.SplitN(key, ":", 2)
		sb.WriteString(fmt.Sprintf("watchdog_requests_total{method=%q,path=%q} %d\n", parts[0], parts[1], counter.Load()))
	}

	sb.WriteString("\n# TYPE watchdog_request_duration_ms histogram\n")
	for key, hist := range globalMetrics.requestDuration {
		parts := strings.SplitN(key, ":", 2)
		for bucket, count := range hist.buckets {
			sb.WriteString(fmt.Sprintf("watchdog_request_duration_ms_bucket{method=%q,path=%q,le=%q} %d\n", parts[0], parts[1], bucket, count.Load()))
		}
		sb.WriteString(fmt.Sprintf("watchdog_request_duration_ms_sum{method=%q,path=%q} %d\n", parts[0], parts[1], hist.sum.Load()))
	}

	sb.WriteString("\n# TYPE watchdog_errors_total counter\n")
	sb.WriteString(fmt.Sprintf("watchdog_errors_total %d\n", globalMetrics.errorCount.Load()))

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(sb.String()))
}

// MetricsMiddleware wraps an http.Handler and records metrics for each request.
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)
		duration := time.Since(start).Milliseconds()
		isError := rec.status >= 400
		RecordRequest(r.Method, r.URL.Path, duration, isError)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}