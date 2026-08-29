// Package router 负责 api-gateway 的路由注册。
//
// 约定：每个业务域的路由单独一个文件维护（auth_router.go / user_router.go /
// admin_router.go），本文件只做总装，避免在单文件里堆砌全部路由。
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/museflow/api-gateway/docs" // 注册 Swagger 文档
	"github.com/museflow/api-gateway/internal/client"
	"github.com/museflow/api-gateway/internal/config"
	"github.com/museflow/api-gateway/internal/handler"
	"github.com/museflow/api-gateway/internal/middleware"
)

// handlers 聚合各业务域的处理器，便于在路由注册函数间传递。
type handlers struct {
	auth       *handler.AuthHandler
	user       *handler.UserHandler
	userManage *handler.UserManageHandler
	admin      *handler.AdminHandler
}

// Setup 创建 Gin 引擎并注册所有路由。
func Setup(cfg *config.Config, userClient *client.UserClient) *gin.Engine {
	r := gin.New()
	r.Use(middleware.AccessLog(), gin.Recovery())
	r.Use(middleware.CORS(cfg.AllowOrigins))

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Swagger 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	h := &handlers{
		auth:       handler.NewAuthHandler(userClient, cfg),
		user:       handler.NewUserHandler(userClient),
		userManage: handler.NewUserManageHandler(userClient),
		admin:      handler.NewAdminHandler(userClient),
	}

	// 鉴权中间件：解析 access token 并从 user-service 校验（含黑名单）
	auth := middleware.Auth(userClient)

	v1 := r.Group("/api/v1")
	registerAuthRoutes(v1, h, auth)
	registerUserRoutes(v1, h, auth)
	registerAdminRoutes(v1, h, userClient, auth)

	return r
}
