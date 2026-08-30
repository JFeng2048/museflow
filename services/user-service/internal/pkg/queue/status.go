// Package queue 基于 asynq 封装异步任务能力。
//
// 分层说明：
//   - tasks.go  任务类型与载荷定义（生产者与消费者共享的契约）
//   - status.go 任务状态模型与存储抽象（用于向客户端回传进度）
//   - client.go 生产者：把任务投递到 Redis 队列
//
// 为什么需要异步化：SMTP 属于慢速外部依赖（单次往返常在数百毫秒到数秒），
// 放在 gRPC 请求链路内会直接拖慢接口响应并占用连接；改为投递到队列后，
// 请求线程只负责生成验证码与投递任务，由独立 Worker 并发消费。
package queue

import (
	"context"
	"time"
)

// 任务状态取值。客户端（SSE）据此判断任务是否结束。
const (
	// StatusPending 已入队，等待 Worker 领取。
	StatusPending = "pending"
	// StatusSending 正在发送。
	StatusSending = "sending"
	// StatusRetrying 发送失败，等待重试。
	StatusRetrying = "retrying"
	// StatusSuccess 发送成功（终态）。
	StatusSuccess = "success"
	// StatusFailed 重试耗尽仍失败（终态）。
	StatusFailed = "failed"
)

// TerminalStatus 返回状态是否为终态（终态之后不再有后续事件）。
func TerminalStatus(status string) bool {
	return status == StatusSuccess || status == StatusFailed
}

// TaskStatus 任务状态快照。
//
// 同一份结构既用于 Redis 键值存储（供后到的订阅者补齐当前进度），
// 也用于 Pub/Sub 广播（供在线订阅者实时接收）。
type TaskStatus struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
	// Message 面向用户的友好提示，可直接展示在前端。
	Message string `json:"message"`
	// UpdatedAt 状态更新时间（Unix 秒）。
	UpdatedAt int64 `json:"updated_at"`
}

// NewStatus 构造状态快照，时间缺省为当前时间。
func NewStatus(taskID, status, message string) TaskStatus {
	return TaskStatus{
		TaskID:    taskID,
		Status:    status,
		Message:   message,
		UpdatedAt: time.Now().Unix(),
	}
}

// StatusStore 任务状态存储抽象，由 repository 层基于 Redis 实现。
//
// 仅暴露生产端需要的写入与读取能力；订阅端（Watch）由 repository.TaskStore 提供。
type StatusStore interface {
	// Update 写入状态并广播给订阅者，ttl 为状态保留时长。
	Update(ctx context.Context, st TaskStatus, ttl time.Duration) error
	// Get 读取当前状态，不存在时返回 (nil, nil)。
	Get(ctx context.Context, taskID string) (*TaskStatus, error)
}
