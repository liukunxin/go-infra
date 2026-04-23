package middlewares

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func GinTraceMiddleware() gin.HandlerFunc {
	tracer := otel.Tracer("gin-server")
	
	return func(c *gin.Context) {
		// 提取上游 trace 信息（如果有）
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// 获取span名称，优先使用FullPath，如果为空则使用Method + Path
		spanName := c.FullPath()
		if spanName == "" {
			// 404或未匹配到路由的情况
			spanName = fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path)
		}

		// 创建 server span，每个请求一个唯一的 TraceID
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer), // 标记为服务端span
			trace.WithAttributes(
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.url", c.Request.URL.String()),
				attribute.String("http.target", c.Request.URL.Path),
				attribute.String("http.host", c.Request.Host),
				attribute.String("http.scheme", c.Request.URL.Scheme),
				attribute.String("http.user_agent", c.Request.UserAgent()),
				attribute.String("http.client_ip", c.ClientIP()),
			),
		)
		defer span.End()

		// 将 ctx 写回 request，让后续 handler 能拿到 traceID
		c.Request = c.Request.WithContext(ctx)

		// 处理请求
		c.Next()

		// 记录响应状态码和错误信息
		statusCode := c.Writer.Status()
		span.SetAttributes(
			attribute.Int("http.status_code", statusCode),
			attribute.Int("http.response_size", c.Writer.Size()),
		)

		// 如果是错误状态码，标记span为错误
		if statusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
			// 如果有错误信息，记录下来
			if len(c.Errors) > 0 {
				span.RecordError(c.Errors.Last().Err)
			}
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}
}
