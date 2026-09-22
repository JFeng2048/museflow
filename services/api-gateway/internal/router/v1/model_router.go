package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/museflow/api-gateway/internal/client"
	"github.com/museflow/api-gateway/internal/middleware"
)

// ModelPermission 模型与系统配置管理要求的权限码。
//
// 该权限只授予 super_admin：渠道的 api_key 是平台资产，一旦放开，
// 任何管理员都能读取到通往上游厂商的凭证。角色权限的增配本身
// 也由 super_admin 在「角色管理」里控制。
const ModelPermission = "system:admin"

// registerModelRoutes 注册模型目录与系统配置路由。
//
// 分两类挂载：
//   - /admin 前缀：平台渠道、平台模型、系统配置的维护，需 system:admin 权限；
//   - /user 前缀：用户自带渠道与模型、可用模型清单、公开配置，只需登录。
//
// 两类的鉴权强度不同：前者持有平台密钥，后者只操作自己的数据。
func registerModelRoutes(r *gin.RouterGroup, h *Handlers, userClient *client.UserClient, auth gin.HandlerFunc) {
	admin := r.Group("/admin", auth, middleware.RequirePermission(userClient, ModelPermission))
	{
		// 平台渠道（厂商 + base_url + api_key，平台侧提供）
		admin.GET("/model-providers", h.Model.ListProviders)
		admin.POST("/model-providers", h.Model.CreateProvider)
		admin.PUT("/model-providers/:id", h.Model.UpdateProvider)
		admin.PUT("/model-providers/:id/active", h.Model.SetProviderActive)
		admin.DELETE("/model-providers/:id", h.Model.DeleteProvider)

		// 平台模型（挂在平台渠道下，用户端可见的主体）
		admin.GET("/models", h.Model.ListModels)
		admin.POST("/models", h.Model.CreateModel)
		admin.PUT("/models/:id", h.Model.UpdateModel)
		admin.PUT("/models/:id/active", h.Model.SetModelActive)
		admin.DELETE("/models/:id", h.Model.DeleteModel)

		// 通用系统配置（密钥只写不读）
		admin.GET("/settings", h.Model.ListSettings)
		admin.GET("/settings/:config_group/:key", h.Model.GetSetting)
		admin.PUT("/settings/:config_group/:key", h.Model.UpsertSetting)
		admin.DELETE("/settings/:config_group/:key", h.Model.DeleteSetting)
	}

	user := r.Group("/user", auth)
	{
		// 我的自定义渠道
		user.GET("/model-providers", h.Model.ListUserProviders)
		user.POST("/model-providers", h.Model.CreateUserProvider)
		user.PUT("/model-providers/:id", h.Model.UpdateUserProvider)
		user.DELETE("/model-providers/:id", h.Model.DeleteUserProvider)

		// 我的自定义模型
		user.GET("/models", h.Model.ListUserModels)
		user.POST("/models", h.Model.CreateUserModel)
		user.GET("/models/available", h.Model.ListAvailableModels)
		user.PUT("/models/:id", h.Model.UpdateUserModel)
		user.DELETE("/models/:id", h.Model.DeleteUserModel)

		// 公开系统配置（只读）
		user.GET("/settings", h.Model.ListPublicSettings)
	}
}
