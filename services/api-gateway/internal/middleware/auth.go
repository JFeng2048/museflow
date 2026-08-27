// Package middleware 提供 api-gateway 的 HTTP 中间件。
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/museflow/api-gateway/internal/client"
	"github.com/museflow/pkg/errcode"
	userpb "github.com/museflow/proto/user"
)

// ContextUserUUID 鉴权通过后写入 gin.Context 的用户标识键。
const ContextUserUUID = "user_uuid"

// Auth JWT 鉴权中间件。
//
// 校验委托给 user-service 的 ValidateToken：
// access token 虽为无状态 JWT，但登出后会进入 Redis 黑名单，
// 仅在网关本地验签无法感知吊销状态，因此必须回调用户服务。
func Auth(userClient *client.UserClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ExtractBearerToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errcode.ErrorGin(c, errcode.CodeMissingToken))
			return
		}

		resp, err := userClient.Service().ValidateToken(c.Request.Context(), &userpb.ValidateTokenRequest{
			AccessToken: token,
		})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, errcode.ErrorGin(c, errcode.CodeServiceUnavail))
			return
		}
		if !resp.GetValid() {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errcode.ErrorGin(c, errcode.CodeTokenInvalid))
			return
		}

		// 供下游处理器获取当前用户
		c.Set(ContextUserUUID, resp.GetUuid())
		c.Next()
	}
}

// ExtractBearerToken 从 Authorization 头解析 Bearer 令牌。
func ExtractBearerToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if header == "" {
		return ""
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// CurrentUserUUID 从上下文读取当前用户 uuid。
func CurrentUserUUID(c *gin.Context) string {
	if v, ok := c.Get(ContextUserUUID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
