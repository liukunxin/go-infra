package log

import (
	"context"
	"fmt"
	"go-infra/pkg/log/core"
)

// =================== Context 支持 ===================

// contextKey 用于存储 traceId/spanId
type contextKey string

const (
	traceIDKey = contextKey("traceId")
	spanIDKey  = contextKey("spanId")
)

// InjectTrace 把 traceId/spanId 写入 context
func InjectTrace(ctx context.Context, traceId, spanId string) context.Context {
	ctx = context.WithValue(ctx, traceIDKey, traceId)
	ctx = context.WithValue(ctx, spanIDKey, spanId)
	return ctx
}

// WithContext 返回一个绑定 ctx 的 Logger
func WithContext(ctx context.Context) *ContextLogger {
	if logger == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	traceId, _ := ctx.Value(traceIDKey).(string)
	spanId, _ := ctx.Value(spanIDKey).(string)
	return &ContextLogger{
		l:       logger,
		traceId: traceId,
		spanId:  spanId,
	}
}

func New() *ContextLogger {
	if logger == nil {
		return nil
	}
	return &ContextLogger{
		l: logger,
	}
}

// WithFields 返回一个新的 ContextLogger，带上额外的默认字段
func (cl *ContextLogger) WithFields(fields map[string]interface{}) *ContextLogger {
	if cl.l == nil {
		return cl
	}

	// 合并 traceId/spanId
	newFields := make(map[string]interface{})
	for k, v := range fields {
		newFields[k] = v
	}

	return &ContextLogger{
		l:       cl.l.WithFields(newFields), // 调用核心 Logger 的 WithFields
		traceId: cl.traceId,
		spanId:  cl.spanId,
	}
}

// ContextLogger 支持链路日志
type ContextLogger struct {
	l       *core.Logger
	traceId string
	spanId  string
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

	// 拼接消息
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}

	cl.l.Log(level, msg, cl.traceId, cl.spanId)
}
