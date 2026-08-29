// Package main 是 user-service 的入口，启动 gRPC 服务端。
package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	userpb "github.com/museflow/proto/user"
	"github.com/museflow/pkg/logger"
	"github.com/museflow/user-service/internal/config"
	"github.com/museflow/user-service/internal/handler"
	"github.com/museflow/user-service/internal/repository"
	"github.com/museflow/user-service/internal/service/admin"
	"github.com/museflow/user-service/internal/service/auth"
	"github.com/museflow/user-service/internal/service/rbac"
	"github.com/museflow/user-service/internal/service/token"
)

func main() {
	// 配置从仓库根目录 .env（前缀 USER_）读取，系统环境变量可覆盖
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化统一日志（输出到文件 + 控制台，按 LOG_ 前缀配置）
	if err := logger.Init(cfg.Log); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}

	db, err := initDB(cfg.DBDSN)
	if err != nil {
		logger.Error("连接数据库失败", logger.Err(err))
		log.Fatalf("连接数据库失败: %v", err)
	}

	rdb, err := initRedis(cfg)
	if err != nil {
		logger.Error("连接 Redis 失败", logger.Err(err))
		log.Fatalf("连接 Redis 失败: %v", err)
	}
	defer rdb.Close()

	// 依赖注入：repository -> service -> handler
	userRepo := repository.NewUserRepository(db)
	rbacRepo := repository.NewRBACRepository(db)
	tokenStore := repository.NewTokenStore(rdb)
	tokenManager := token.NewTokenManager(cfg.JWTSecret, cfg.AccessTTL, cfg.RefreshTTL)
	// 权限缓存 TTL 与 Refresh Token 一致（7 天）
	rbacService := rbac.NewService(rbacRepo, tokenStore, cfg.RefreshTTL)
	authService := auth.NewAuthService(userRepo, tokenStore, tokenManager, rbacService, cfg.BcryptCost)
	adminService := admin.NewService(userRepo, rbacService)
	userHandler := handler.NewUserHandler(authService, adminService)

	// 预置角色 / 权限（库为空时插入），失败仅告警不阻断启动
	if err := rbacService.EnsureSeeded(context.Background()); err != nil {
		logger.Warn("预置 RBAC 数据失败", logger.Err(err))
	}

	grpcServer := grpc.NewServer()
	userpb.RegisterUserServiceServer(grpcServer, userHandler)

	// 健康检查，供容器编排探活
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("user.UserService", grpc_health_v1.HealthCheckResponse_SERVING)

	// 反射服务，便于用 grpcurl 调试
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		logger.Error("监听端口失败", "port", cfg.Port, logger.Err(err))
		log.Fatalf("监听端口 %s 失败: %v", cfg.Port, err)
	}

	logger.Info("MuseFlow user-service 已启动",
		"port", cfg.Port,
		"access_ttl", cfg.AccessTTL.String(),
		"refresh_ttl", cfg.RefreshTTL.String(),
	)
	logger.Info("gRPC 监听地址", "addr", "grpc://localhost:"+cfg.Port)

	// 优雅关闭：收到信号后停止接收新请求并等待存量请求结束
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		logger.Info("正在关闭 user-service ...")
		grpcServer.GracefulStop()
		logger.Info("user-service 已退出")
	}()

	if err := grpcServer.Serve(lis); err != nil {
		logger.Error("gRPC 服务异常退出", logger.Err(err))
		log.Fatalf("gRPC 服务异常退出: %v", err)
	}
}

// initDB 初始化 GORM 连接池。
// 不执行 AutoMigrate：schema 由 database/user_svc.sql 维护。
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

// initRedis 初始化 Redis 客户端并验证连通性。
func initRedis(cfg *config.Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB:       cfg.RedisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return rdb, nil
}
