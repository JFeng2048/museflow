package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/museflow/api-gateway/internal/client"
	"github.com/museflow/api-gateway/internal/config"
	authdto "github.com/museflow/api-gateway/internal/dto/auth_dto"
	commondto "github.com/museflow/api-gateway/internal/dto/common_dto"
	"github.com/museflow/pkg/errcode"
	"github.com/museflow/pkg/logger"
	userpb "github.com/museflow/proto/user"
)

// 任务终态：收到后服务端主动关闭 SSE 连接。
const (
	taskStatusSuccess = "success"
	taskStatusFailed  = "failed"
)

// SSE 连接参数。
const (
	// sseRetryInterval 通过 retry 字段告知浏览器断线后的重连间隔。
	sseRetryInterval = 3 * time.Second
	// sseHeartbeatInterval 心跳间隔，防止中间层（Nginx / 负载均衡）因空闲断开连接。
	sseHeartbeatInterval = 15 * time.Second
)

// CommonHandler 公开通用接口处理器（/common 路由组）。
//
// 这些接口无需 access token：发送邮箱验证码、刷新访问令牌、订阅异步任务进度。
type CommonHandler struct {
	users *client.UserClient
	cfg   *config.Config
}

// NewCommonHandler 构造公开通用处理器。
func NewCommonHandler(users *client.UserClient, cfg *config.Config) *CommonHandler {
	return &CommonHandler{users: users, cfg: cfg}
}

// SendVerifyCode 发送邮箱验证码
//
//	@Summary		发送邮箱验证码
//	@Description	按场景发送邮箱验证码：register（注册校验）/ login（验证码登录）/ reset_password（密码重置）/ change_email（修改邮箱）；避免账号枚举，邮箱不存在时也返回成功
//	@Description	邮件通过异步队列发送，接口只负责生成验证码并入队，立即返回 task_id（HTTP 202）；
//	@Description	客户端可用 task_id 订阅 GET /common/tasks/{task_id}/stream 获取实时发送进度。
//	@Tags			common-公共
//	@Accept			json
//	@Produce		json
//	@Param			body	body		authdto.SendVerifyCodeRequest	true	"邮箱与场景"
//	@Success		202		{object}	errcode.Response{data=commondto.SendVerifyCodeData}	"已受理，异步发送中"
//	@Failure		400		{object}	errcode.Response	"参数校验失败"
//	@Failure		429		{object}	errcode.Response	"发送过于频繁"
//	@Router			/common/email/send-code [post]
func (h *CommonHandler) SendVerifyCode(c *gin.Context) {
	var req authdto.SendVerifyCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errcode.Fail(errcode.CodeParamInvalid, err.Error()))
		return
	}

	resp, err := h.users.Service().SendVerifyCode(c.Request.Context(), &userpb.SendVerifyCodeRequest{
		Email: req.Email,
		Scene: req.Scene,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	// 202 表示「已受理但尚未完成」，进度通过 SSE 端点推送
	data := commondto.SendVerifyCodeData{
		TaskID:    resp.GetTaskId(),
		ExpiresIn: resp.GetExpiresIn(),
	}
	c.JSON(http.StatusAccepted, errcode.AcceptedGin(c, data))
}

// StreamTask 订阅异步任务进度（SSE）
//
//	@Summary		订阅异步任务进度（SSE）
//	@Description	以 Server-Sent Events 持续推送任务状态，供前端展示「发送中 / 已发送 / 发送失败」。
//	@Description	事件名即任务状态：pending（已入队）/ sending（发送中）/ retrying（重试中）/ success / failed；
//	@Description	收到 success 或 failed 后服务端关闭连接。建议用 EventSource 接入。
//	@Tags			common-公共
//	@Produce		text/event-stream
//	@Param			task_id	path		string	true	"任务 ID，来自 /common/email/send-code 的返回"
//	@Success		200		{string}	string	"事件流：event: <status> / data: {\"task_id\":..,\"status\":..,\"message\":..}"
//	@Failure		400		{object}	errcode.Response	"缺少任务 ID"
//	@Failure		404		{object}	errcode.Response	"任务不存在或已过期"
//	@Router			/common/tasks/{task_id}/stream [get]
func (h *CommonHandler) StreamTask(c *gin.Context) {
	taskID := strings.TrimSpace(c.Param("task_id"))
	if taskID == "" {
		c.JSON(http.StatusBadRequest, errcode.ErrorGin(c, errcode.CodeParamInvalid))
		return
	}

	ctx := c.Request.Context()
	stream, err := h.users.Service().WatchTask(ctx, &userpb.WatchTaskRequest{TaskId: taskID})
	if err != nil {
		// 流建立失败仍走 JSON 错误响应（此时响应头尚未切换为 SSE）
		writeGRPCError(c, err)
		return
	}

	// 先取第一个事件再切换为 SSE。
	// gRPC 流式调用是惰性建连的：任务不存在（状态已过期）等错误只在首次 Recv 时暴露，
	// 此时响应头尚未写死，还能回一个正常的 JSON 错误；否则客户端只会看到一条空的事件流。
	first, err := stream.Recv()
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	// 切换为 SSE：禁用一切缓冲，保证事件即时到达浏览器
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache, no-transform")
	c.Writer.Header().Set("Connection", "keep-alive")
	// 关闭 Nginx 的响应缓冲，否则事件会被攒在一起下发
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	// 告知浏览器断线后的重连间隔
	fmt.Fprintf(c.Writer, "retry: %d\n\n", sseRetryInterval.Milliseconds())
	c.Writer.Flush()

	// 补发首个事件（任务的当前快照）
	firstStatus := first.GetStatus()
	if err := writeSSEEvent(c, firstStatus, commondto.TaskEventData{
		TaskID:    first.GetTaskId(),
		Status:    firstStatus,
		Message:   first.GetMessage(),
		UpdatedAt: first.GetUpdatedAt(),
	}); err != nil {
		logger.WarnContext(ctx, "写入 SSE 事件失败", "task_id", taskID, logger.Err(err))
		return
	}
	if firstStatus == taskStatusSuccess || firstStatus == taskStatusFailed {
		return
	}

	// gRPC Recv 是阻塞调用，放到独立协程，主线程才能同时处理心跳与客户端断开
	type event struct {
		msg *userpb.TaskEvent
		err error
	}
	events := make(chan event, 1)
	go func() {
		defer close(events)
		for {
			msg, err := stream.Recv()
			select {
			case events <- event{msg: msg, err: err}:
			case <-ctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(sseHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// 客户端断开连接
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			if ev.err != nil {
				if ev.err == io.EOF {
					return
				}
				logger.WarnContext(ctx, "接收任务进度失败", "task_id", taskID, logger.Err(ev.err))
				return
			}
			st := ev.msg.GetStatus()
			if err := writeSSEEvent(c, st, commondto.TaskEventData{
				TaskID:    ev.msg.GetTaskId(),
				Status:    st,
				Message:   ev.msg.GetMessage(),
				UpdatedAt: ev.msg.GetUpdatedAt(),
			}); err != nil {
				logger.WarnContext(ctx, "写入 SSE 事件失败", "task_id", taskID, logger.Err(err))
				return
			}
			// 终态事件推送完毕即关闭，避免客户端空等
			if st == taskStatusSuccess || st == taskStatusFailed {
				return
			}
		case <-ticker.C:
			// 注释行（冒号开头）不触发前端事件，仅用于保活
			if _, err := io.WriteString(c.Writer, ": keep-alive\n\n"); err != nil {
				return
			}
			c.Writer.Flush()
		}
	}
}

// writeSSEEvent 按 SSE 规范写入一条事件并立即刷新。
func writeSSEEvent(c *gin.Context, event string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, payload); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}

// Refresh 刷新访问令牌
//
//	@Summary		刷新访问令牌
//	@Description	从 Cookie 读取 refresh token 换取新的 access token，刷新后旧 refresh 轮转
//	@Tags			common-公共
//	@Produce		json
//	@Success		200	{object}	errcode.Response{data=authdto.RefreshData}	"刷新成功"
//	@Failure		401	{object}	errcode.Response				"刷新令牌无效或已过期"
//	@Failure		403	{object}	errcode.Response				"设备校验失败"
//	@Router			/common/refresh [post]
func (h *CommonHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie(refreshCookieName)
	if err != nil || refreshToken == "" {
		c.JSON(http.StatusUnauthorized, errcode.ErrorGin(c, errcode.CodeMissingRefresh))
		return
	}
	deviceID, err := c.Cookie(deviceCookieName)
	if err != nil || deviceID == "" {
		c.JSON(http.StatusUnauthorized, errcode.ErrorGin(c, errcode.CodeMissingDevice))
		return
	}

	resp, err := h.users.Service().Refresh(c.Request.Context(), &userpb.RefreshRequest{
		RefreshToken: refreshToken,
		Device: &userpb.DeviceContext{
			DeviceId:  deviceID,
			UserAgent: c.Request.UserAgent(),
			Ip:        c.ClientIP(),
		},
	})
	if err != nil {
		// 刷新失败说明会话已不可用，清除 Cookie 避免客户端反复重试
		if status.Code(err) == codes.Unauthenticated {
			clearCookies(c, h.cfg)
		}
		writeGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, errcode.SuccessGin(c, authdto.RefreshData{
		AccessToken: resp.GetAccessToken(),
		TokenType:   "Bearer",
		ExpiresIn:   resp.GetExpiresIn(),
	}))
}
