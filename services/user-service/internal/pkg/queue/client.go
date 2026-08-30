package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

// Config 队列配置。
type Config struct {
	RedisAddr string
	RedisPass string
	RedisDB   int

	// QueueName 默认队列名，Worker 按队列设置并发与优先级。
	QueueName string
	// MaxRetry 任务失败后的最大重试次数。
	MaxRetry int
	// Timeout 单次任务执行超时。
	Timeout time.Duration
	// Concurrency Worker 并发消费协程数。
	Concurrency int
	// Retention 已完成任务在队列中的保留时长，便于排查问题。
	Retention time.Duration
	// StatusTTL 任务状态在 Redis 中的保留时长，超时后客户端无法再订阅进度。
	StatusTTL time.Duration
}

// applyDefaults 补全默认值。
func (c *Config) applyDefaults() {
	if c.QueueName == "" {
		c.QueueName = "default"
	}
	if c.MaxRetry < 0 {
		c.MaxRetry = 0
	}
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}
	if c.Concurrency <= 0 {
		c.Concurrency = 20
	}
	if c.Retention <= 0 {
		c.Retention = time.Hour
	}
	if c.StatusTTL <= 0 {
		c.StatusTTL = 10 * time.Minute
	}
}

// RedisOpt 转为 asynq 的 Redis 连接配置。
func (c Config) RedisOpt() asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Addr:     c.RedisAddr,
		Password: c.RedisPass,
		DB:       c.RedisDB,
	}
}

// Client 任务生产者：把任务投递到 Redis 队列。
//
// 投递成功后同步写入一条 pending 状态，保证客户端紧接着订阅 SSE 时
// 一定能读到初始进度，不会出现「任务已入队但查不到状态」的空窗。
type Client struct {
	cfg    Config
	client *asynq.Client
	status StatusStore
}

// New 构造队列客户端；status 可为 nil（此时只投递、不记录进度）。
func New(cfg Config, status StatusStore) *Client {
	cfg.applyDefaults()
	return &Client{
		cfg:    cfg,
		client: asynq.NewClient(cfg.RedisOpt()),
		status: status,
	}
}

// Close 关闭底层连接。
func (c *Client) Close() error {
	return c.client.Close()
}

// Asynq 返回底层 asynq 客户端，供 asynqmon 等高级用法使用。
func (c *Client) Asynq() *asynq.Client { return c.client }

// EnqueueEmailVerifyCode 投递「发送邮箱验证码」任务，返回任务 ID。
func (c *Client) EnqueueEmailVerifyCode(ctx context.Context, p EmailVerifyCodePayload) (string, error) {
	body, err := marshalPayload(p)
	if err != nil {
		return "", err
	}
	info, err := c.client.EnqueueContext(ctx,
		asynq.NewTask(TypeEmailVerifyCode, body),
		asynq.Queue(c.cfg.QueueName),
		asynq.MaxRetry(c.cfg.MaxRetry),
		asynq.Timeout(c.cfg.Timeout),
		asynq.Retention(c.cfg.Retention),
	)
	if err != nil {
		return "", fmt.Errorf("投递邮箱验证码任务失败: %w", err)
	}
	if c.status != nil {
		st := NewStatus(info.ID, StatusPending, "验证码已排队，正在发送")
		if err := c.status.Update(ctx, st, c.cfg.StatusTTL); err != nil {
			// 进度写入失败不影响任务本身，记录日志后继续
			return info.ID, nil
		}
	}
	return info.ID, nil
}

// EnqueueEmailWelcome 投递「欢迎邮件」任务。
//
// 欢迎邮件属于锦上添花的通知，投递失败只记日志、不阻断注册流程。
func (c *Client) EnqueueEmailWelcome(ctx context.Context, p EmailWelcomePayload) (string, error) {
	body, err := marshalPayload(p)
	if err != nil {
		return "", err
	}
	info, err := c.client.EnqueueContext(ctx,
		asynq.NewTask(TypeEmailWelcome, body),
		asynq.Queue(c.cfg.QueueName),
		asynq.MaxRetry(c.cfg.MaxRetry),
		asynq.Timeout(c.cfg.Timeout),
		asynq.Retention(c.cfg.Retention),
	)
	if err != nil {
		return "", fmt.Errorf("投递欢迎邮件任务失败: %w", err)
	}
	return info.ID, nil
}
