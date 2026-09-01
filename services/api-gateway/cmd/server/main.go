// Package main 是 MuseFlow api-gateway 的入口。
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/museflow/api-gateway/internal/client"
	"github.com/museflow/api-gateway/internal/config"
	"github.com/museflow/api-gateway/internal/router"
	"github.com/museflow/pkg/logger"
)

// @title			MuseFlow API Gateway
// @version		1.0
// @description	MuseFlow AI 小说生成平台的 API 网关，统一对外暴露 HTTP 接口并转发到后端 gRPC 微服务。
// @description
// @description	认证采用双令牌机制：
// @description	- access token：短期（默认 1 小时），通过响应 body 返回，请求时放入 Authorization 头
// @description	- refresh token：长期（默认 30 天），存于 HttpOnly Cookie，用于换取新的 access token
//
// @contact.name	MuseFlow Team
// @license.name	MIT
// @license.url	https://opensource.org/licenses/MIT
//
// @host		localhost:5001
// @BasePath	/api/v1
// @schemes	http https
//
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				请填入 "Bearer {access_token}"
func main() {
	// 配置从仓库根目录 .env（前缀 GATEWAY_）读取，系统环境变量可覆盖
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化统一日志（输出到文件 + 控制台，按 LOG_ 前缀配置）
	if err := logger.Init(cfg.Log); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}

	// 建立到 user-service 的 gRPC 连接（惰性连接，不阻塞启动）
	userClient, err := client.NewUserClient(cfg.UserServiceURL)
	if err != nil {
		logger.Error("初始化 user-service 客户端失败", logger.Err(err))
		log.Fatalf("初始化 user-service 客户端失败: %v", err)
	}
	defer userClient.Close()

	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := router.Setup(cfg, userClient)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           engine,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logger.Info("MuseFlow api-gateway 已启动", "port", cfg.Port)
	logger.Info("服务地址", "addr", "http://localhost:"+cfg.Port)
	logger.Info("Swagger 文档", "addr", "http://localhost:"+cfg.Port+"/swagger/index.html")
	logger.Info("user-service", "target", cfg.UserServiceURL)

	// 优雅关闭
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		logger.Info("正在关闭 api-gateway ...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("关闭 api-gateway 失败", logger.Err(err))
		}
		logger.Info("api-gateway 已退出")
	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("HTTP 服务异常退出", logger.Err(err))
		log.Fatalf("HTTP 服务异常退出: %v", err)
	}
}
