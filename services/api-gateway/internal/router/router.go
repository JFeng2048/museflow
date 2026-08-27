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

	// 鉴权中间件
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

		userGroup := v1.Group("/user", auth)
		{
			userGroup.GET("/profile", userHandler.Profile)
		}
	}

	return r
}
