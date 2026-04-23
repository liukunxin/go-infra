package trace

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// GetTraceID extracts the TraceID string from ctx.
// Returns an empty string when ctx is nil or no valid span is present,
// allowing callers to use a simple `if id == ""` guard.
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return ""
	}
	return sc.TraceID().String()
}

// GetSpanID extracts the SpanID string from ctx.
// Returns an empty string when ctx is nil or no valid span is present.
func GetSpanID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return ""
	}
	return sc.SpanID().String()
}

// HasTrace reports whether ctx carries a valid, sampled trace span.
func HasTrace(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	sc := trace.SpanFromContext(ctx).SpanContext()
	return sc.IsValid() && sc.IsSampled()
}
