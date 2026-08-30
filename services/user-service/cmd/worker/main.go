// Command worker 是 user-service 的异步任务消费端（asynq Worker）。
//
// 与 gRPC 服务（cmd/server）分离部署：
//   - 服务进程只处理在线请求，不因慢速外部依赖（SMTP）而堆积；
//   - Worker 可独立扩容，用并发数控制发信吞吐；
//   - 任一进程崩溃互不影响。
//
// 启动：cd services/user-service && go run ./cmd/worker
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"github.com/museflow/pkg/logger"
	"github.com/museflow/user-service/internal/config"
	"github.com/museflow/user-service/internal/pkg/email"
	"github.com/museflow/user-service/internal/pkg/queue"
	"github.com/museflow/user-service/internal/repository"
	"github.com/museflow/user-service/internal/worker/handlers"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if err := logger.Init(cfg.Log); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}

	rdb, err := initRedis(cfg)
	if err != nil {
		logger.Error("连接 Redis 失败", logger.Err(err))
		log.Fatalf("连接 Redis 失败: %v", err)
	}
	defer rdb.Close()

	renderer, err := email.NewRenderer()
	if err != nil {
		log.Fatalf("初始化邮件模板失败: %v", err)
	}

	// Worker 只回写任务进度，不承载业务逻辑
	taskStore := repository.NewTaskStore(rdb)
	emailHandler := handlers.NewEmailHandler(
		email.NewSender(cfg.SMTP),
		renderer,
		handlers.EmailHandlerConfig{
			StatusStore: taskStore,
			StatusTTL:   cfg.Queue.StatusTTL,
		},
	)

	mux := asynq.NewServeMux()
	mux.Handle(queue.TypeEmailVerifyCode, emailHandler)
	mux.Handle(queue.TypeEmailWelcome, emailHandler)

	srv := asynq.NewServer(
		cfg.Queue.RedisOpt(),
		asynq.Config{
			// 并发数决定同时能处理多少封邮件，是发信吞吐的主开关
			Concurrency: cfg.Queue.Concurrency,
			Queues:      map[string]int{cfg.Queue.QueueName: 1},
			// 优雅关闭时等待在建任务的最长时间
			ShutdownTimeout: 30 * time.Second,
			ErrorHandler:    newErrorHandler(),
			Logger:          asynqLogger{},
		},
	)

	logger.Info("MuseFlow user-service worker 已启动",
		"queue", cfg.Queue.QueueName,
		"concurrency", cfg.Queue.Concurrency,
	)

	// 优雅关闭：收到信号后停止领取新任务，等待在建任务结束
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		logger.Info("正在关闭 user-service worker ...")
		srv.Stop()
		srv.Shutdown()
		logger.Info("user-service worker 已退出")
	}()

	if err := srv.Run(mux); err != nil {
		logger.Error("asynq Worker 异常退出", logger.Err(err))
		log.Fatalf("asynq Worker 异常退出: %v", err)
	}
}

// newErrorHandler 统一记录任务处理失败（如重试耗尽）。
//
// 任务失败通常由 SMTP 抖动引起，重试机制会兜底，这里只做可观测性输出。
func newErrorHandler() asynq.ErrorHandler {
	return asynq.ErrorHandlerFunc(func(ctx context.Context, t *asynq.Task, err error) {
		retried, _ := asynq.GetRetryCount(ctx)
		maxRetry, _ := asynq.GetMaxRetry(ctx)
		logger.ErrorContext(ctx, "异步任务处理失败",
			"type", t.Type(), "retried", retried, "max_retry", maxRetry, logger.Err(err))
	})
}

// initRedis 初始化 Redis 客户端并验证连通性（asynq 队列与任务状态共用）。
func initRedis(cfg *config.Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Queue.RedisAddr,
		Password: cfg.Queue.RedisPass,
		DB:       cfg.Queue.RedisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return rdb, nil
}

// asynqLogger 把 asynq 内部日志桥接到统一的 logger。
type asynqLogger struct{}

func (asynqLogger) Debug(args ...any) { logger.Info("[asynq] " + fmt.Sprint(args...)) }
func (asynqLogger) Info(args ...any)  { logger.Info("[asynq] " + fmt.Sprint(args...)) }
func (asynqLogger) Warn(args ...any)  { logger.Warn("[asynq] " + fmt.Sprint(args...)) }
func (asynqLogger) Error(args ...any) { logger.Error("[asynq] " + fmt.Sprint(args...)) }
func (asynqLogger) Fatal(args ...any) { logger.Error("[asynq][fatal] " + fmt.Sprint(args...)) }
