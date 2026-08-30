package queue

import (
	"encoding/json"
	"fmt"
)

// 任务类型常量。命名采用「领域:动作」形式，便于在 asynqmon 中按前缀过滤。
const (
	// TypeEmailVerifyCode 发送邮箱验证码（注册 / 登录 / 重置密码 / 修改邮箱）。
	TypeEmailVerifyCode = "email:verify_code"
	// TypeEmailWelcome 注册成功后的欢迎邮件。
	TypeEmailWelcome = "email:welcome"
)

// EmailVerifyCodePayload 邮箱验证码任务载荷。
//
// 注意：验证码在生成后即写入 Redis 并在此处传递给 Worker，
// Worker 只负责渲染与投递，不再访问业务存储。
type EmailVerifyCodePayload struct {
	To   string `json:"to"`
	Code string `json:"code"`
	// Scene 业务场景：register / login / reset_password / change_email。
	Scene string `json:"scene"`
	// Purpose 面向用户的用途描述，如「注册 MuseFlow 账号」。
	Purpose string `json:"purpose"`
	// TTL 验证码有效期（秒），用于邮件正文提示。
	TTL int64 `json:"ttl_seconds"`
}

// EmailWelcomePayload 欢迎邮件任务载荷。
type EmailWelcomePayload struct {
	To       string `json:"to"`
	Nickname string `json:"nickname"`
}

// marshalPayload 序列化任务载荷。
func marshalPayload(v any) ([]byte, error) {
	return json.Marshal(v)
}

// UnmarshalVerifyCodePayload 解析邮箱验证码任务载荷。
func UnmarshalVerifyCodePayload(data []byte, v *EmailVerifyCodePayload) error {
	return unmarshalPayload(data, v)
}

// UnmarshalWelcomePayload 解析欢迎邮件任务载荷。
func UnmarshalWelcomePayload(data []byte, v *EmailWelcomePayload) error {
	return unmarshalPayload(data, v)
}

// unmarshalPayload 反序列化任务载荷。
func unmarshalPayload(data []byte, v any) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("任务载荷解析失败: %w", err)
	}
	return nil
}
