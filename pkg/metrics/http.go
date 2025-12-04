package metrics

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"log"
	"time"
)

// ------------------ 全局指标 ------------------

var (
	meter         metric.Meter
	HttpRequests  metric.Int64Counter
	HttpLatencyMs metric.Float64Histogram
)

// 默认自带的http指标
func initHTTPMetrics() {
	meter = otel.Meter("http-server")

	var err error

	HttpRequests, err = meter.Int64Counter(
		"http_requests_total",
	)
	if err != nil {
		log.Fatalf("failed to create HttpRequests counter: %v", err)
	}

	HttpLatencyMs, err = meter.Float64Histogram(
		"http_request_duration_ms",
	)
	if err != nil {
		log.Fatalf("failed to create HttpLatencyMs histogram: %v", err)
	}
}

func ginMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Milliseconds()
		ctx := c.Request.Context()

		attrs := []attribute.KeyValue{
			attribute.String("method", c.Request.Method),
			attribute.String("path", c.FullPath()),
			attribute.Int("status", c.Writer.Status()),
		}

		// 使用 metric.WithAttributes 包装
		HttpRequests.Add(ctx, 1, metric.WithAttributes(attrs...))
		HttpLatencyMs.Record(ctx, float64(duration), metric.WithAttributes(attrs...))
	}
}
