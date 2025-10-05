package log

import (
	"backend/go-infra/pkg/log/core"
	"context"
)

var logger *core.Logger

// Init 初始化日志
func Init(cfg Config) {
	var formatter core.Formatter
	if cfg.Formatter == "json" {
		formatter = &core.JSONFormatter{}
	} else {
		formatter = &core.TxtLineFormatter{}
	}

	provider := core.NewStdProvider()
	buffer := cfg.BufferSize
	if buffer <= 0 {
		buffer = 1000
	}
	logger = core.NewLogger(cfg.Level, provider, formatter, buffer)
}

// Close 关闭日志（确保异步队列写完）
func Close() {
	if logger != nil {
		logger.Close()
	}
}

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

// ContextLogger 支持链路日志
type ContextLogger struct {
	l       *core.Logger
	traceId string
	spanId  string
}

func (cl *ContextLogger) Debug(msg string, fields ...map[string]interface{}) {
	cl.log(core.LevelDebug, msg, fields...)
}
func (cl *ContextLogger) Info(msg string, fields ...map[string]interface{}) {
	cl.log(core.LevelInfo, msg, fields...)
}
func (cl *ContextLogger) Warn(msg string, fields ...map[string]interface{}) {
	cl.log(core.LevelWarn, msg, fields...)
}
func (cl *ContextLogger) Error(msg string, fields ...map[string]interface{}) {
	cl.log(core.LevelError, msg, fields...)
}
func (cl *ContextLogger) Fatal(msg string, fields ...map[string]interface{}) {
	cl.log(core.LevelFatal, msg, fields...)
}

func (cl *ContextLogger) log(level int, msg string, fields ...map[string]interface{}) {
	if cl.l == nil {
		return
	}
	combined := make(map[string]interface{})
	for _, m := range fields {
		if m != nil {
			for k, v := range m {
				combined[k] = v
			}
		}
	}
	cl.l.Log(level, msg, combined, cl.traceId, cl.spanId)
}
