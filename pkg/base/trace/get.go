package trace

import (
	"context"
	"go.opentelemetry.io/otel/trace"
)

// GetTraceID 从context中提取TraceID
// 每个请求在经过 GinTraceMiddleware 后都会有唯一的TraceID
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return "unknown_trace_id"
	}
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	if !sc.IsValid() {
		return "unknown_trace_id"
	}
	return sc.TraceID().String()
}

// GetSpanID 从context中提取SpanID
func GetSpanID(ctx context.Context) string {
	if ctx == nil {
		return "unknown_span_id"
	}
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	if !sc.IsValid() {
		return "unknown_span_id"
	}
	return sc.SpanID().String()
}
