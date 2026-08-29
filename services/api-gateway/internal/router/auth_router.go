package router

import (
	"github.com/gin-gonic/gin"
)

// registerAuthRoutes 注册认证相关路由（/api/v1/auth）。
//
// 注册与登录无需认证；refresh 走 HttpOnly Cookie 校验，不需要 access token；
// logout 需要 access token。
func registerAuthRoutes(v1 *gin.RouterGroup, h *handlers, auth gin.HandlerFunc) {
	group := v1.Group("/auth")
	{
		// 无需认证
		group.POST("/register", h.auth.Register)
		group.POST("/login", h.auth.Login)
		// 走 Cookie 校验，不需要 access token
		group.POST("/refresh", h.auth.Refresh)
		// 需要 access token
		group.POST("/logout", auth, h.auth.Logout)
	}
}
