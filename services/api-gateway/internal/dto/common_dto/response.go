// Package commondto 定义 /common 路由组（公开通用接口）的 HTTP 数据结构。
package commondto

// SendVerifyCodeData 发送邮箱验证码响应。
type SendVerifyCodeData struct {
	// TaskID 异步任务 ID，用于订阅发送进度（GET /common/tasks/{task_id}/stream）。
	TaskID string `json:"task_id" example:"8f2c1e5a-3b7d-4a1e-9c0f-2d6b8a4e1f30"`
	// ExpiresIn 验证码有效期（秒）。
	ExpiresIn int64 `json:"expires_in" example:"600"`
}

// TaskEventData SSE 推送的任务进度事件。
//
// 事件名即 Status 取值：pending / sending / retrying / success / failed。
// 收到 success 或 failed 后服务端会关闭连接，前端无需再重连该任务。
type TaskEventData struct {
	TaskID string `json:"task_id" example:"8f2c1e5a-3b7d-4a1e-9c0f-2d6b8a4e1f30"`
	// Status 任务状态：pending（已入队）/ sending（发送中）/ retrying（重试中）/ success（成功）/ failed（失败）
	Status string `json:"status" example:"success"`
	// Message 面向用户的友好提示，可直接展示
	Message string `json:"message" example:"验证码已发送，请查收邮件"`
	// UpdatedAt 状态更新时间（Unix 秒）
	UpdatedAt int64 `json:"updated_at" example:"1756521600"`
}
