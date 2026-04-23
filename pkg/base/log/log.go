package log

import (
	"context"
	"fmt"

	"github.com/liukunxin/go-infra/pkg/base/log/core"
	"go.opentelemetry.io/otel/trace"
)

// WithContext returns a ContextLogger enriched with the TraceID and SpanID extracted from ctx.
func WithContext(ctx context.Context) *ContextLogger {
	if ctx == nil {
		ctx = context.Background()
	}
	var traceID, spanID string
	sc := trace.SpanFromContext(ctx).SpanContext()
	if sc.HasTraceID() {
		traceID = sc.TraceID().String()
	}
	if sc.HasSpanID() {
		spanID = sc.SpanID().String()
	}
	return &ContextLogger{l: loadLogger(), traceId: traceID, spanId: spanID}
}

// New returns a plain ContextLogger without trace information.
func New() *ContextLogger {
	l := loadLogger()
	if l == nil {
		return nil
	}
	return &ContextLogger{l: l}
}

// ContextLogger is a thin wrapper around core.Logger that carries per-request state
// (trace IDs, extra fields) without mutating the shared Logger.
type ContextLogger struct {
	l           *core.Logger
	traceId     string
	spanId      string
	extraFields map[string]interface{}
}

// WithFields returns a new ContextLogger that includes the given fields in every log entry.
// The input map is copied so callers can safely modify or reuse it after this call.
func (cl *ContextLogger) WithFields(fields map[string]interface{}) *ContextLogger {
	if cl.l == nil {
		return cl
	}
	// Shallow-copy the map so mutations by the caller cannot affect this logger.
	copied := make(map[string]interface{}, len(cl.extraFields)+len(fields))
	for k, v := range cl.extraFields {
		copied[k] = v
	}
	for k, v := range fields {
		copied[k] = v
	}
	return &ContextLogger{
		l:           cl.l,
		traceId:     cl.traceId,
		spanId:      cl.spanId,
		extraFields: copied,
	}
}

func (cl *ContextLogger) Debug(msg string, args ...interface{}) {
	cl.log(core.LevelDebug, msg, args...)
}
func (cl *ContextLogger) Info(msg string, args ...interface{}) {
	cl.log(core.LevelInfo, msg, args...)
}
func (cl *ContextLogger) Warn(msg string, args ...interface{}) {
	cl.log(core.LevelWarn, msg, args...)
}
func (cl *ContextLogger) Error(msg string, args ...interface{}) {
	cl.log(core.LevelError, msg, args...)
}
func (cl *ContextLogger) Fatal(msg string, args ...interface{}) {
	cl.log(core.LevelFatal, msg, args...)
}

func (cl *ContextLogger) log(level int, format string, args ...interface{}) {
	if cl.l == nil {
		return
	}
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	// extraFields is passed directly to the core logger so the formatter receives them
	// as proper structured fields — they appear as top-level JSON keys, not baked into msg.
	// The map is already a copy owned by this ContextLogger, so it is safe to share.
	cl.l.Log(level, msg, cl.traceId, cl.spanId, cl.extraFields)
}
