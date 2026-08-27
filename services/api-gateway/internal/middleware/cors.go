package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS 跨域中间件。
//
// 由于 refresh token 通过 Cookie 传输，必须携带凭证，
// 因此 Access-Control-Allow-Origin 不能为 "*"，需按白名单回显具体来源。
func CORS(allowOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowOrigins))
	allowAll := false
	for _, o := range allowOrigins {
		if o == "*" {
			allowAll = true
		}
		allowed[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if origin != "" {
			_, ok := allowed[origin]
			if ok || allowAll {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				c.Header("Access-Control-Allow-Headers", strings.Join([]string{
					"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With",
				}, ", "))
				c.Header("Access-Control-Max-Age", "86400")
				// 来源随请求变化，需声明 Vary 避免缓存污染
				c.Header("Vary", "Origin")
			}
		}

		// 预检请求直接结束
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
