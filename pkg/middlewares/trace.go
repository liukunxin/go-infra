package middlewares

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func GinTraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 提取上游 trace 信息（如果有）
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		tracer := otel.Tracer("gin-server")

		// 每个请求创建一个 root span
		ctx, span := tracer.Start(ctx, c.FullPath())
		defer span.End()

		// 将 ctx 写回 request，让后续 handler 能拿到 traceID
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
