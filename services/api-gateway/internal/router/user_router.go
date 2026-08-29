package router

import (
	"github.com/gin-gonic/gin"
)

// registerUserRoutes 注册用户自助路由（/api/v1/user）。
//
// 全部接口只需登录即可访问，不涉及权限码校验。
func registerUserRoutes(v1 *gin.RouterGroup, h *handlers, auth gin.HandlerFunc) {
	group := v1.Group("/user", auth)
	{
		// 个人资料
		group.GET("/profile", h.user.Profile)
		group.PUT("/profile", h.userManage.UpdateProfile)
		group.PUT("/password", h.userManage.ChangePassword)

		// 权限
		group.GET("/permissions", h.userManage.MyPermissions)

		// 会话管理
		group.GET("/sessions", h.userManage.ListSessions)
		group.DELETE("/sessions/:token", h.userManage.RevokeSession)

		// 第三方账号绑定
		group.GET("/oauth", h.userManage.ListOAuthBindings)
		group.DELETE("/oauth/:provider", h.userManage.UnbindOAuth)
	}
}
