package middlewares

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/liukunxin/go-infra/internal/consts"
	"github.com/liukunxin/go-infra/pkg/base/log"
	"github.com/liukunxin/go-infra/pkg/base/trace"
	"github.com/spf13/cast"
)

func HttpLogRecord() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录开始时间
		startTime := time.Now()
		// 处理请求
		c.Next()
		// 记录结束时间
		endTime := time.Now()
		latency := endTime.Sub(startTime)

		// 获取其他信息
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path
		userAgent := c.Request.UserAgent()
		bodySize := c.Writer.Size()
		respCode, ok := c.Get(consts.ResponseCode)
		if !ok {
			respCode = -1
		}
		respMsg := c.GetString(consts.ResponseMsg)

		// 封装主要信息
		logMessage := fmt.Sprintf("[%s_%s]%s | %d | %d | %v | %s",
			method,
			path,
			respMsg,
			statusCode,
			respCode,
			latency,
			clientIP)
		// 打印日志
		fields := map[string]interface{}{
			"time":       time.Now().Format(time.DateTime),
			"status":     statusCode,
			"code":       cast.ToInt32(respCode),
			"latency":    latency,
			"client_ip":  clientIP,
			"method":     method,
			"path":       path,
			"user_agent": userAgent,
			"body_size":  bodySize,
		}
		if traceID := trace.GetTraceID(c.Request.Context()); traceID != "" {
			fields["trace_id"] = traceID
		}
		lg := log.WithContext(c.Request.Context()).WithFields(fields)
		sc := statusCode / 100
		if sc == 5 {
			lg.Error(logMessage)
		} else if sc == 4 {
			lg.Warn(logMessage)
		} else {
			lg.Info(logMessage)
		}
	}
}
