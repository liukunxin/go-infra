package log

import (
	"context"
	"fmt"
	"github.com/liukunxin/go-infra/pkg/log/core"
	"go.opentelemetry.io/otel/trace"
	"sync"
)

// =================== 对象池优化 ===================

// mapPool 用于复用 map[string]interface{} 对象，减少内存分配
var mapPool = sync.Pool{
	New: func() interface{} {
		// 预分配一定容量，减少扩容
		return make(map[string]interface{}, 8)
	},
}

// getMapFromPool 从对象池获取 map
func getMapFromPool() map[string]interface{} {
	return mapPool.Get().(map[string]interface{})
}

// putMapToPool 归还 map 到对象池（清空后归还）
func putMapToPool(m map[string]interface{}) {
	// 清空 map
	for k := range m {
		delete(m, k)
	}
	mapPool.Put(m)
}

// =================== Context 支持 ===================

// WithContext 返回一个绑定 ctx 的 Logger
// 只提取 TraceID，SpanID 通常只在详细的分布式追踪中需要
func WithContext(ctx context.Context) *ContextLogger {
	if ctx == nil {
		ctx = context.Background()
	}
	span := trace.SpanFromContext(ctx)
	spanCtx := span.SpanContext()
	var traceID string
	var spanID string
	if spanCtx.HasTraceID() {
		traceID = spanCtx.TraceID().String()
	}
	if spanCtx.HasSpanID() {
		spanID = spanCtx.SpanID().String()
	}
	// 注意：TraceID用于关联整个请求的日志，SpanID用于详细的链路追踪
	// 即使不使用分布式追踪工具，保留SpanID也没有性能影响，且保留了扩展性
	return &ContextLogger{
		l:       logger,
		traceId: traceID,
		spanId:  spanID,
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
// 重要：这个方法在高并发场景下频繁调用，需要避免共享状态导致的竞态条件
// 解决方案：不修改底层 Logger，而是在 ContextLogger 层面保存字段
func (cl *ContextLogger) WithFields(fields map[string]interface{}) *ContextLogger {
	if cl.l == nil {
		return cl
	}

	// 创建新的 ContextLogger，携带额外字段（不修改原 Logger）
	// 注意：这里不调用 core.Logger.WithFields，避免并发竞态和内存泄漏
	return &ContextLogger{
		l:           cl.l,
		traceId:     cl.traceId,
		spanId:      cl.spanId,
		extraFields: fields, // 保存额外字段，在实际输出时合并
	}
}

// ContextLogger 支持链路日志
type ContextLogger struct {
	l           *core.Logger
	traceId     string
	spanId      string
	extraFields map[string]interface{} // 额外的字段，避免修改共享的 Logger
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

	// 如果有额外字段，需要临时合并到消息中
	// 这里采用简单的方式：将字段序列化到消息中
	if len(cl.extraFields) > 0 {
		// 从对象池获取 map 用于临时合并
		combined := getMapFromPool()
		for k, v := range cl.extraFields {
			combined[k] = v
		}
		
		// 将字段添加到消息中（简化处理）
		fieldsStr := ""
		for k, v := range combined {
			fieldsStr += fmt.Sprintf(" %s=%v", k, v)
		}
		if fieldsStr != "" {
			msg = msg + fieldsStr
		}
		
		// 归还到对象池
		putMapToPool(combined)
	}

	cl.l.Log(level, msg, cl.traceId, cl.spanId)
}
