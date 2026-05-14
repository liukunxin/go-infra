package llm

import (
	"context"
	"sync"
	"time"

	baselog "github.com/liukunxin/go-infra/pkg/base/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var (
	llmTracer = otel.Tracer("go-infra/llm")

	llmMetricsOnce sync.Once

	llmRequestCounter metric.Int64Counter
	llmLatencyMs      metric.Float64Histogram
)

func initMetrics() {
	llmMetricsOnce.Do(func() {
		m := otel.Meter("go-infra/llm")
		llmRequestCounter, _ = m.Int64Counter(
			"llm_requests_total",
			metric.WithDescription("Total LLM requests by provider/model/operation/status."),
		)
		llmLatencyMs, _ = m.Float64Histogram(
			"llm_latency_ms",
			metric.WithDescription("LLM request latency in milliseconds."),
			metric.WithUnit("ms"),
		)
	})
}

func recordMetrics(ctx context.Context, provider, model, operation, status string, latency time.Duration) {
	initMetrics()
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
		attribute.String("operation", operation),
		attribute.String("status", status),
	}
	if llmRequestCounter != nil {
		llmRequestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if llmLatencyMs != nil {
		llmLatencyMs.Record(ctx, float64(latency.Milliseconds()), metric.WithAttributes(attrs...))
	}
}

func logAttempt(ctx context.Context, msg string, fields map[string]interface{}) {
	baselog.WithContext(ctx).WithFields(fields).Info(msg)
}

func startSpan(ctx context.Context, operation, provider, model string) (context.Context, trace.Span) {
	return llmTracer.Start(ctx, "llm."+operation, trace.WithAttributes(
		attribute.String("llm.operation", operation),
		attribute.String("llm.provider", provider),
		attribute.String("llm.model", model),
	))
}
