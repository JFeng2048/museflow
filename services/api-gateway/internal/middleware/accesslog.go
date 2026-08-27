package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/museflow/pkg/logger"
)

// RequestIDHeader 在响应头与 context 中携带请求唯一标识的键。
const RequestIDHeader = "X-Request-ID"

// AccessLog HTTP 访问日志中间件。
//
// 为每个请求生成 request_id 并注入 context，供下游 handler 通过
// logger.WithRequestID 透传；同时记录方法、路径、状态码、耗时与客户端 IP。
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 优先复用客户端带上的 request_id，否则生成新的
		reqID := c.GetHeader(RequestIDHeader)
		if reqID == "" {
			reqID = newRequestID()
		}
		c.Header(RequestIDHeader, reqID)

		// 将 request_id 注入 context，使后续业务日志自动带上
		ctx := logger.WithRequestID(c.Request.Context(), reqID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		latency := time.Since(start)
		if raw != "" {
			path = path + "?" + raw
		}

		logger.InfoContext(ctx, "HTTP 访问",
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		)
	}
}
