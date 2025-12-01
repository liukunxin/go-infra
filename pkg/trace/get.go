package trace

import (
	"context"
	"go.opentelemetry.io/otel/trace"
)

func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	if !sc.IsValid() {
		return "unknown_trace_id"
	}
	return sc.TraceID().String()
}
