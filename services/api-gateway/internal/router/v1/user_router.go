package v1

import (
	"github.com/gin-gonic/gin"
)

// registerUserRoutes 注册用户自助路由（/api/v1/user）。
//
// 全部接口只需登录即可访问，不涉及权限码校验。
func registerUserRoutes(r *gin.RouterGroup, h *Handlers, auth gin.HandlerFunc) {
	group := r.Group("/user", auth)
	{
		// 个人资料
		group.GET("/profile", h.User.Profile)
		group.PUT("/profile", h.UserManage.UpdateProfile)
		group.PUT("/password", h.UserManage.ChangePassword)
		group.POST("/email/change", h.UserManage.ChangeEmail)

		// 权限
		group.GET("/permissions", h.UserManage.MyPermissions)

		// 会话管理
		group.GET("/sessions", h.UserManage.ListSessions)
		group.DELETE("/sessions/:token", h.UserManage.RevokeSession)

		// 第三方账号绑定
		group.GET("/oauth", h.UserManage.ListOAuthBindings)
		group.DELETE("/oauth/:provider", h.UserManage.UnbindOAuth)
	}
}
