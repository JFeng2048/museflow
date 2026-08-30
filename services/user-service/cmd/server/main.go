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

	"github.com/museflow/pkg/logger"
	userpb "github.com/museflow/proto/user"
	"github.com/museflow/user-service/internal/config"
	"github.com/museflow/user-service/internal/handler"
	"github.com/museflow/user-service/internal/pkg/queue"
	"github.com/museflow/user-service/internal/pkg/turnstile"
	"github.com/museflow/user-service/internal/repository"
	"github.com/museflow/user-service/internal/service/admin"
	"github.com/museflow/user-service/internal/service/audit"
	"github.com/museflow/user-service/internal/service/auth"
	"github.com/museflow/user-service/internal/service/oauth"
	"github.com/museflow/user-service/internal/service/rbac"
	"github.com/museflow/user-service/internal/service/task"
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
	auditRepo := repository.NewAuditRepository(db)
	oauthRepo := repository.NewOAuthRepository(db)
	tokenStore := repository.NewTokenStore(rdb)
	codeStore := repository.NewVerifyCodeStore(rdb)
	taskStore := repository.NewTaskStore(rdb)
	// 邮件等慢速操作走 asynq 队列：这里的客户端只负责投递，消费在 cmd/worker
	queueClient := queue.New(cfg.Queue, taskStore)
	defer queueClient.Close()
	// 人机验证：未配置密钥时返回恒通过的客户端（开发态自动跳过）
	captcha := captchaVerifier(cfg.Turnstile)
	taskService := task.NewService(taskStore)
	tokenManager := token.NewTokenManager(cfg.JWTSecret, cfg.AccessTTL, cfg.RefreshTTL, cfg.MFATicketTTL)
	// 权限缓存 TTL 与 Refresh Token 一致（7 天）
	rbacService := rbac.NewService(rbacRepo, tokenStore, cfg.RefreshTTL)
	auditService := audit.NewService(auditRepo)
	oauthService := oauth.NewService(oauthRepo, userRepo, rbacService, auditService)
	authService := auth.NewAuthService(
		userRepo, tokenStore, tokenManager,
		rbacService, auditService, oauthService,
		codeStore, queueClient, captcha,
		auth.ResetServiceConfig{
			CodeTTL:      cfg.CodeTTL,
			CodeLength:   cfg.CodeLength,
			CodeResendCD: cfg.CodeResendCD,
		},
		auth.EmailCodeConfig{
			CodeTTL:      cfg.CodeTTL,
			CodeLength:   cfg.CodeLength,
			CodeResendCD: cfg.CodeResendCD,
		},
		auth.MFAConfig{
			Issuer:             cfg.MFAIssuer,
			CodeSkew:           cfg.MFASkew,
			RecoveryCodeCount:  cfg.MFARecoveryCodes,
			RecoveryCodeLength: cfg.MFARecoveryCodeLen,
		},
		cfg.BcryptCost,
	)
	adminService := admin.NewService(userRepo, rbacService, auditService)
	userHandler := handler.NewUserHandler(authService, adminService, taskService)

	// 角色与权限数据由数据库维护（database/user_svc.sql），代码不做种子写入
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

// captchaVerifier 构造人机验证器。
//
// 未配置密钥时返回一个恒通过的客户端：本地开发没有 Cloudflare 配置也能走通流程，
// 但此时发送验证码接口不受保护，启动时会有明确告警。
func captchaVerifier(cfg turnstile.Config) auth.CaptchaVerifier {
	if !cfg.Enabled() {
		logger.Warn("人机验证未配置（USER_TURNSTILE_SECRET 为空），发送验证码接口将不受保护；生产环境必须配置")
		return turnstile.NewNoop()
	}
	logger.Info("人机验证已启用", "endpoint", cfg.Endpoint, "timeout", cfg.Timeout.String())
	return turnstile.New(cfg)
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
