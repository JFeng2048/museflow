// Package main 是 config-service 的入口，启动 gRPC 服务端。
//
// 本服务承载「模型目录 + 凭证 + 通用系统配置」：平台渠道 / 平台模型 / 用户自定义
// 渠道 / 用户自定义模型 / 系统配置共五张表。schema 由 database/config_svc.sql
// 维护，启动时不执行 AutoMigrate。
package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/museflow/config-service/internal/config"
	"github.com/museflow/config-service/internal/handler"
	"github.com/museflow/config-service/internal/pkg/secret"
	"github.com/museflow/config-service/internal/pkg/upstream"
	"github.com/museflow/config-service/internal/repository"
	"github.com/museflow/config-service/internal/service"
	"github.com/museflow/pkg/logger"
	modelpb "github.com/museflow/proto/model"
)

func main() {
	// 配置从 config-service 服务目录 .env（前缀 CONFIG_）读取，系统环境变量可覆盖
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化统一日志（输出到 stdout，按 LOG_ 前缀配置）
	if err := logger.Init(cfg.Log); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}

	// 凭证加密器：密钥长度不符时 fail fast，不让服务带着错误的密钥起来
	encrypter, err := secret.NewEncrypter(cfg.SecretKey)
	if err != nil {
		logger.Error("初始化凭证加密器失败", logger.Err(err))
		log.Fatalf("初始化凭证加密器失败: %v", err)
	}

	db, err := initDB(cfg.DBDSN)
	if err != nil {
		logger.Error("连接数据库失败", logger.Err(err))
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 依赖注入：repository -> service -> handler
	providerRepo := repository.NewProviderRepository(db)
	modelRepo := repository.NewModelRepository(db)
	userProviderRepo := repository.NewUserProviderRepository(db)
	userModelRepo := repository.NewUserModelRepository(db)
	settingRepo := repository.NewSettingRepository(db)

	// 上游模型目录客户端：超时取包内默认值，实例全局复用（内部只有无状态的 http.Client）
	upstreamClient := upstream.NewClient(upstream.DefaultTimeout)

	modelService := service.NewService(service.Deps{
		Providers:     providerRepo,
		Models:        modelRepo,
		UserProviders: userProviderRepo,
		UserModels:    userModelRepo,
		Settings:      settingRepo,
		Secrets:       encrypter,
		Upstream:      upstreamClient,
	})
	modelHandler := handler.NewModelHandler(modelService)

	grpcServer := grpc.NewServer()
	modelpb.RegisterModelServiceServer(grpcServer, modelHandler)

	// 健康检查，供容器编排探活
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("model.ModelService", grpc_health_v1.HealthCheckResponse_SERVING)

	// 反射服务，便于用 grpcurl 调试
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		logger.Error("监听端口失败", "port", cfg.Port, logger.Err(err))
		log.Fatalf("监听端口 %s 失败: %v", cfg.Port, err)
	}

	logger.Info("MuseFlow config-service 已启动", "port", cfg.Port)
	logger.Info("gRPC 监听地址", "addr", "grpc://localhost:"+cfg.Port)

	// 优雅关闭：收到信号后停止接收新请求并等待存量请求结束
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		logger.Info("正在关闭 config-service ...")
		grpcServer.GracefulStop()
		logger.Info("config-service 已退出")
	}()

	if err := grpcServer.Serve(lis); err != nil {
		logger.Error("gRPC 服务异常退出", logger.Err(err))
		log.Fatalf("gRPC 服务异常退出: %v", err)
	}
}

// initDB 初始化 GORM 连接池。
// 不执行 AutoMigrate：schema 由 database/config_svc.sql 维护。
func initDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}
