package router

import (
	"github.com/gin-gonic/gin"

	"github.com/museflow/api-gateway/internal/client"
	"github.com/museflow/api-gateway/internal/middleware"
)

// adminPermission 管理后台统一要求的权限码。
const adminPermission = "user:admin"

// registerAdminRoutes 注册管理后台路由（/api/v1/admin）。
//
// 全部接口在登录校验之外，统一附加 RequirePermission 中间件：
// 以 token 中的 user_uuid 调用 user-service 的 CheckPermission 校验 user:admin。
func registerAdminRoutes(v1 *gin.RouterGroup, h *handlers, userClient *client.UserClient, auth gin.HandlerFunc) {
	group := v1.Group("/admin", auth, middleware.RequirePermission(userClient, adminPermission))
	{
		// 用户管理
		group.GET("/users", h.admin.ListUsers)
		group.GET("/users/:uuid", h.admin.GetUserDetail)
		group.PUT("/users/:uuid/status", h.admin.UpdateUserStatus)
		group.PUT("/users/:uuid/role", h.admin.AssignRole)

		// 角色管理
		group.GET("/roles", h.admin.ListRoles)
		group.POST("/roles", h.admin.CreateRole)
		group.PUT("/roles/:id", h.admin.UpdateRole)
		group.DELETE("/roles/:id", h.admin.DeleteRole)

		// 权限管理
		group.GET("/permissions", h.admin.ListPermissions)
		group.PUT("/roles/:id/permissions", h.admin.SetRolePermissions)

		// 审计日志
		group.GET("/audit-logs", h.admin.ListAuditLogs)
	}
}
