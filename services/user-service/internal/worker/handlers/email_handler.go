// Package handlers 存放 asynq 任务处理器。
//
// 每个处理器只做一件事：解析载荷 -> 执行 -> 回写进度。
// 业务判断（如验证码生成、账号校验）留在 service 层，Worker 保持无状态，
// 这样可以水平扩容多个实例来提高发信吞吐。
package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	"github.com/museflow/pkg/logger"
	"github.com/museflow/user-service/internal/pkg/email"
	"github.com/museflow/user-service/internal/pkg/queue"
)

// 面向用户的进度提示。
const (
	msgSending  = "正在发送邮件，请稍候"
	msgSuccess  = "验证码已发送，请查收邮件（如未收到请检查垃圾邮件）"
	msgFailed   = "邮件发送失败，请稍后重试"
	msgRetrying = "邮件发送遇到一点问题，正在重试"
)

// asynq 从 context 读取任务元数据的入口。
//
// 抽成包级变量是为了让单元测试能注入固定的任务 ID / 重试次数：
// asynq 的 context 元数据键未导出，外部无法自行构造带任务信息的 ctx。
var (
	taskIDFromContext = asynq.GetTaskID
	retryStateFromCtx = func(ctx context.Context) (retry, maxRetry int) {
		retry, _ = asynq.GetRetryCount(ctx)
		maxRetry, _ = asynq.GetMaxRetry(ctx)
		return retry, maxRetry
	}
)

// EmailHandler 邮件类异步任务处理器。
type EmailHandler struct {
	sender    email.Sender
	renderer  *email.Renderer
	status    queue.StatusStore
	statusTTL time.Duration
}

// EmailHandlerConfig 邮件任务处理器配置。
type EmailHandlerConfig struct {
	// StatusStore 任务状态存储，为 nil 时不回写进度（仅发信）。
	StatusStore queue.StatusStore
	// StatusTTL 进度保留时长。
	StatusTTL time.Duration
}

// NewEmailHandler 构造邮件任务处理器。
func NewEmailHandler(sender email.Sender, renderer *email.Renderer, cfg EmailHandlerConfig) *EmailHandler {
	ttl := cfg.StatusTTL
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &EmailHandler{
		sender:    sender,
		renderer:  renderer,
		status:    cfg.StatusStore,
		statusTTL: ttl,
	}
}

// ProcessTask 按任务类型分发（实现 asynq.Handler 接口）。
func (h *EmailHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	switch t.Type() {
	case queue.TypeEmailVerifyCode:
		return h.handleVerifyCode(ctx, t)
	case queue.TypeEmailWelcome:
		return h.handleWelcome(ctx, t)
	default:
		return fmt.Errorf("未知任务类型: %s", t.Type())
	}
}

// handleVerifyCode 发送邮箱验证码。
func (h *EmailHandler) handleVerifyCode(ctx context.Context, t *asynq.Task) error {
	taskID, _ := taskIDFromContext(ctx)

	var p queue.EmailVerifyCodePayload
	if err := queue.UnmarshalVerifyCodePayload(t.Payload(), &p); err != nil {
		// 载荷损坏无法恢复，标记为失败且不重试（避免反复占用队列）
		h.mark(ctx, taskID, queue.StatusFailed, msgFailed)
		return skipRetry(err)
	}

	h.mark(ctx, taskID, queue.StatusSending, msgSending)

	html, text, err := h.renderer.Render(email.TemplateForScene(p.Scene), email.VerifyCodeData{
		Purpose: p.Purpose,
		Code:    email.FormatCode(p.Code),
		Minutes: minutesOf(p.TTL),
	})
	if err != nil {
		h.reportFailure(ctx, taskID, err)
		return err
	}

	msg := email.Message{
		To:      p.To,
		Subject: email.VerifyCodeSubject(p.Scene),
		HTML:    html,
		Text:    text,
	}
	if err := h.sender.Send(ctx, msg); err != nil {
		h.reportFailure(ctx, taskID, err)
		return err
	}

	h.mark(ctx, taskID, queue.StatusSuccess, msgSuccess)
	logger.InfoContext(ctx, "邮箱验证码已发送", "task_id", taskID, "to", p.To, "scene", p.Scene)
	return nil
}

// handleWelcome 发送注册欢迎邮件。
func (h *EmailHandler) handleWelcome(ctx context.Context, t *asynq.Task) error {
	taskID, _ := taskIDFromContext(ctx)

	var p queue.EmailWelcomePayload
	if err := queue.UnmarshalWelcomePayload(t.Payload(), &p); err != nil {
		return skipRetry(err)
	}

	html, text, err := h.renderer.Render(email.TemplateWelcome, email.WelcomeData{
		Nickname: p.Nickname,
		Email:    p.To,
	})
	if err != nil {
		logger.ErrorContext(ctx, "渲染欢迎邮件失败", logger.Err(err))
		return skipRetry(err)
	}

	if err := h.sender.Send(ctx, email.Message{
		To:      p.To,
		Subject: "欢迎加入 MuseFlow",
		HTML:    html,
		Text:    text,
	}); err != nil {
		logger.WarnContext(ctx, "欢迎邮件发送失败", "to", p.To, logger.Err(err))
		return err
	}

	// 欢迎邮件不参与进度订阅，无需回写状态
	logger.InfoContext(ctx, "欢迎邮件已发送", "task_id", taskID, "to", p.To)
	return nil
}

// reportFailure 记录失败并在重试耗尽时标记终态。
//
// 重试期间状态为 retrying，客户端可继续等待；重试耗尽后才写入 failed，
// 避免前端在中途就提示「发送失败」。
func (h *EmailHandler) reportFailure(ctx context.Context, taskID string, err error) {
	retry, maxRetry := retryStateFromCtx(ctx)

	logger.WarnContext(ctx, "邮件发送失败", "task_id", taskID, "retry", retry, "max_retry", maxRetry, logger.Err(err))
	if retry >= maxRetry {
		h.mark(ctx, taskID, queue.StatusFailed, msgFailed)
		return
	}
	h.mark(ctx, taskID, queue.StatusRetrying, msgRetrying)
}

// mark 回写任务进度，写入失败只记日志（不能让进度写失败影响发信主流程）。
func (h *EmailHandler) mark(ctx context.Context, taskID, status, message string) {
	if h.status == nil || taskID == "" {
		return
	}
	if err := h.status.Update(ctx, queue.NewStatus(taskID, status, message), h.statusTTL); err != nil {
		logger.WarnContext(ctx, "写入任务进度失败", "task_id", taskID, "status", status, logger.Err(err))
	}
}

// minutesOf 把秒换算为展示用的分钟数，不足 1 分钟按 1 分钟计。
func minutesOf(seconds int64) int {
	m := int(seconds / 60)
	if m < 1 {
		return 1
	}
	return m
}

// skipRetry 包装错误并告知 asynq 不再重试（用于载荷损坏等不可恢复错误）。
func skipRetry(err error) error {
	return fmt.Errorf("%w: %w", err, asynq.SkipRetry)
}
