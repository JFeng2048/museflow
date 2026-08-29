package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/museflow/api-gateway/internal/client"
	"github.com/museflow/api-gateway/internal/middleware"
)

// AdminPermission 管理后台统一要求的权限码。
const AdminPermission = "user:admin"

// registerAdminRoutes 注册管理后台路由（/api/v1/admin）。
//
// 全部接口在登录校验之外，统一附加 RequirePermission 中间件：
// 以 token 中的 user_uuid 调用 user-service 的 CheckPermission 校验 user:admin。
func registerAdminRoutes(r *gin.RouterGroup, h *Handlers, userClient *client.UserClient, auth gin.HandlerFunc) {
	group := r.Group("/admin", auth, middleware.RequirePermission(userClient, AdminPermission))
	{
		// 用户管理
		group.GET("/users", h.Admin.ListUsers)
		group.GET("/users/:uuid", h.Admin.GetUserDetail)
		group.PUT("/users/:uuid/status", h.Admin.UpdateUserStatus)
		group.PUT("/users/:uuid/role", h.Admin.AssignRole)

		// 角色管理
		group.GET("/roles", h.Admin.ListRoles)
		group.POST("/roles", h.Admin.CreateRole)
		group.PUT("/roles/:id", h.Admin.UpdateRole)
		group.DELETE("/roles/:id", h.Admin.DeleteRole)

		// 权限管理
		group.GET("/permissions", h.Admin.ListPermissions)
		group.PUT("/roles/:id/permissions", h.Admin.SetRolePermissions)

		// 审计日志
		group.GET("/audit-logs", h.Admin.ListAuditLogs)
	}
}
