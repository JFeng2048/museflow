// Package v1 存放 API v1 版本的路由注册。
//
// 约定：本文件统一注册 v1 的全部路由，每个业务域的路由单独一个文件维护
// （auth_router.go / user_router.go / admin_router.go），
// 新增业务域时只需在该域文件中注册，并在 Register 里补一行调用。
package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/museflow/api-gateway/internal/client"
	"github.com/museflow/api-gateway/internal/config"
	"github.com/museflow/api-gateway/internal/handler"
	"github.com/museflow/api-gateway/internal/middleware"
)

// Handlers 聚合各业务域的处理器，便于在各域路由注册函数间传递。
type Handlers struct {
	Auth       *handler.AuthHandler
	Common     *handler.CommonHandler
	User       *handler.UserHandler
	UserManage *handler.UserManageHandler
	Admin      *handler.AdminHandler
}

// NewHandlers 构造全部业务域的处理器。
func NewHandlers(cfg *config.Config, userClient *client.UserClient) *Handlers {
	return &Handlers{
		Auth:       handler.NewAuthHandler(userClient, cfg),
		Common:     handler.NewCommonHandler(userClient, cfg),
		User:       handler.NewUserHandler(userClient),
		UserManage: handler.NewUserManageHandler(userClient),
		Admin:      handler.NewAdminHandler(userClient),
	}
}

// Register 在 /api/v1 路由组下注册 v1 的全部路由。
//
// auth 为鉴权中间件（解析 access token 并从 user-service 校验，含黑名单检查），
// 需要登录的接口会在各域路由中按需挂载。
func Register(r *gin.RouterGroup, cfg *config.Config, userClient *client.UserClient) {
	h := NewHandlers(cfg, userClient)
	auth := middleware.Auth(userClient)

	registerAuthRoutes(r, h, auth)
	registerCommonRoutes(r, h)
	registerUserRoutes(r, h, auth)
	registerAdminRoutes(r, h, userClient, auth)
}
