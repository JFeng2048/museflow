package v1

import (
	"github.com/gin-gonic/gin"
)

// registerAuthRoutes 注册认证相关路由（/api/v1/auth）。
//
// 注册与登录无需认证；refresh 走 HttpOnly Cookie 校验，不需要 access token；
// logout 需要 access token。
func registerAuthRoutes(r *gin.RouterGroup, h *Handlers, auth gin.HandlerFunc) {
	group := r.Group("/auth")
	{
		// 无需认证
		group.POST("/register", h.Auth.Register)
		group.POST("/login", h.Auth.Login)
		// 走 Cookie 校验，不需要 access token
		group.POST("/refresh", h.Auth.Refresh)
		// 需要 access token
		group.POST("/logout", auth, h.Auth.Logout)
	}
}
