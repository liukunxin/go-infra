package metrics

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var defaultHTTPSkipPaths = []string{"/health", "/metrics"}

var (
	meter              metric.Meter
	HttpRequests       metric.Int64Counter
	HttpDurationSeconds metric.Float64Histogram
)

func initHTTPMetrics() error {
	meter = otel.Meter("http-server")

	var err error

	HttpRequests, err = meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests."),
	)
	if err != nil {
		return fmt.Errorf("metrics: create http_requests_total counter: %w", err)
	}

	// Unit follows Prometheus convention: base unit (seconds) with no suffix abbreviation.
	HttpDurationSeconds, err = meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request latency in seconds."),
		metric.WithUnit("s"),
	)
	if err != nil {
		return fmt.Errorf("metrics: create http_request_duration_seconds histogram: %w", err)
	}

	return nil
}

func buildHTTPSkipSet(extra ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(defaultHTTPSkipPaths)+len(extra))
	for _, p := range defaultHTTPSkipPaths {
		out[normalizeHTTPPath(p)] = struct{}{}
	}
	for _, p := range extra {
		if p = normalizeHTTPPath(p); p != "" {
			out[p] = struct{}{}
		}
	}
	return out
}

func normalizeHTTPPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func ginMetricsMiddleware(skip map[string]struct{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := skip[c.Request.URL.Path]; ok {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		duration := time.Since(start).Seconds()
		ctx := c.Request.Context()

		path := c.FullPath()
		if path == "" {
			// Unregistered routes (e.g. 404). Use a fixed label to avoid unbounded cardinality
			// from arbitrary user-supplied paths.
			path = "unknown"
		}

		attrs := []attribute.KeyValue{
			attribute.String("method", c.Request.Method),
			attribute.String("path", path),
			attribute.Int("status", c.Writer.Status()),
		}

		HttpRequests.Add(ctx, 1, metric.WithAttributes(attrs...))
		HttpDurationSeconds.Record(ctx, duration, metric.WithAttributes(attrs...))
	}
}
