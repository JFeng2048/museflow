// Package router 负责 api-gateway 的路由注册。
//
// 约定：按 API 版本分目录维护（当前只有 v1/），每个版本目录下有一个集中注册的
// 文件（v1/v1_router.go）负责注册该版本的全部路由；本文件只做全局总装，
// 如健康检查、Swagger 与版本路由组的挂载。
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/museflow/api-gateway/docs" // 注册 Swagger 文档
	"github.com/museflow/api-gateway/internal/client"
	"github.com/museflow/api-gateway/internal/config"
	"github.com/museflow/api-gateway/internal/middleware"
	"github.com/museflow/api-gateway/internal/router/v1"
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

	// 各版本 API 路由
	v1.Register(r.Group("/api/v1"), cfg, userClient)

	return r
}
