package http_client

import (
	"context"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const metricPathUnknown = "unknown"

var (
	clientMetricsOnce sync.Once

	clientRequestsTotal   metric.Int64Counter
	clientDurationSeconds metric.Float64Histogram
)

func initClientMetrics() {
	clientMetricsOnce.Do(func() {
		m := otel.Meter("go-infra/http_client")
		clientRequestsTotal, _ = m.Int64Counter(
			"http_client_requests_total",
			metric.WithDescription("Total outbound HTTP client requests."),
		)
		clientDurationSeconds, _ = m.Float64Histogram(
			"http_client_request_duration_seconds",
			metric.WithDescription("Outbound HTTP client request latency in seconds."),
			metric.WithUnit("s"),
		)
	})
}

func recordClientMetrics(ctx context.Context, req *http.Request, metricPath string, status int, latency time.Duration) {
	initClientMetrics()
	if clientRequestsTotal == nil && clientDurationSeconds == nil {
		return
	}
	path := resolveMetricPath(req, metricPath)
	host := ""
	method := ""
	if req != nil {
		method = req.Method
		if req.URL != nil {
			host = req.URL.Host
		}
	}
	attrs := []attribute.KeyValue{
		attribute.String("method", method),
		attribute.String("host", host),
		attribute.String("path", path),
		attribute.Int("status", status),
	}
	if clientRequestsTotal != nil {
		clientRequestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if clientDurationSeconds != nil {
		clientDurationSeconds.Record(ctx, latency.Seconds(), metric.WithAttributes(attrs...))
	}
}

// resolveMetricPath prefers an explicit template from WithMetricPath.
// Otherwise it uses the real URL path; if any path segment is purely numeric
// (typical resource IDs like "1001"), it falls back to "unknown".
func resolveMetricPath(req *http.Request, explicit string) string {
	if explicit != "" {
		return explicit
	}
	if req == nil || req.URL == nil {
		return metricPathUnknown
	}
	path := req.URL.Path
	if path == "" {
		path = "/"
	}
	if pathHasPureDigitSegment(path) {
		return metricPathUnknown
	}
	return path
}

// pathHasPureDigitSegment reports whether any /-separated segment is all digits.
// "/users/1001/orders" → true; "/api/v1/ping" → false (v1 is not pure digits).
func pathHasPureDigitSegment(path string) bool {
	start := 0
	for i := 0; i <= len(path); i++ {
		if i < len(path) && path[i] != '/' {
			continue
		}
		if i > start && isPureDigits(path[start:i]) {
			return true
		}
		start = i + 1
	}
	return false
}

func isPureDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
