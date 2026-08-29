// Package router 负责 api-gateway 的路由注册。
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

	authHandler := handler.NewAuthHandler(userClient, cfg)
	userHandler := handler.NewUserHandler(userClient)
	userManageHandler := handler.NewUserManageHandler(userClient)
	adminHandler := handler.NewAdminHandler(userClient)

	// 鉴权中间件：解析 access token 并从 user-service 校验（含黑名单）
	auth := middleware.Auth(userClient)

	v1 := r.Group("/api/v1")
	{
		authGroup := v1.Group("/auth")
		{
			// 无需认证
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
			// 走 Cookie 校验，不需要 access token
			authGroup.POST("/refresh", authHandler.Refresh)
			// 需要 access token
			authGroup.POST("/logout", auth, authHandler.Logout)
		}

		// 用户自助接口：只需登录
		userGroup := v1.Group("/user", auth)
		{
			userGroup.GET("/profile", userHandler.Profile)
			userGroup.PUT("/profile", userManageHandler.UpdateProfile)
			userGroup.PUT("/password", userManageHandler.ChangePassword)
			userGroup.GET("/permissions", userManageHandler.MyPermissions)

			userGroup.GET("/sessions", userManageHandler.ListSessions)
			userGroup.DELETE("/sessions/:token", userManageHandler.RevokeSession)

			userGroup.GET("/oauth", userManageHandler.ListOAuthBindings)
			userGroup.DELETE("/oauth/:provider", userManageHandler.UnbindOAuth)
		}

		// 管理后台接口：需登录 + user:admin 权限
		adminGroup := v1.Group("/admin", auth, middleware.RequirePermission(userClient, "user:admin"))
		{
			adminGroup.GET("/users", adminHandler.ListUsers)
			adminGroup.GET("/users/:uuid", adminHandler.GetUserDetail)
			adminGroup.PUT("/users/:uuid/status", adminHandler.UpdateUserStatus)
			adminGroup.PUT("/users/:uuid/role", adminHandler.AssignRole)

			adminGroup.GET("/roles", adminHandler.ListRoles)
			adminGroup.POST("/roles", adminHandler.CreateRole)
			adminGroup.PUT("/roles/:id", adminHandler.UpdateRole)
			adminGroup.DELETE("/roles/:id", adminHandler.DeleteRole)
			adminGroup.PUT("/roles/:id/permissions", adminHandler.SetRolePermissions)

			adminGroup.GET("/permissions", adminHandler.ListPermissions)

			adminGroup.GET("/audit-logs", adminHandler.ListAuditLogs)
		}
	}

	return r
}
