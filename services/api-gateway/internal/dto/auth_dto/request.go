// Package authdto 定义认证相关接口（注册 / 登录 / 刷新 / 登出 / 密码重置）的 HTTP 请求结构。
package authdto

// RegisterRequest 注册请求。
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email" example:"author@museflow.ai"`
	Password string `json:"password" binding:"required,min=8,max=72" example:"P@ssw0rd123"`
	Nickname string `json:"nickname" binding:"omitempty,max=100" example:"墨流作者"`
	// Code 注册邮箱验证码，调 /auth/email/send-code（scene=register）获取
	Code string `json:"code" binding:"required" example:"824913"`
}

// LoginRequest 登录请求。
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"author@museflow.ai"`
	Password string `json:"password" binding:"required" example:"P@ssw0rd123"`
	// DeviceName 设备名称，用于设备列表展示，可选
	DeviceName string `json:"device_name" binding:"omitempty,max=100" example:"Chrome on Windows"`
}

// SendResetCodeRequest 发送密码重置验证码请求。
type SendResetCodeRequest struct {
	Email string `json:"email" binding:"required,email" example:"author@museflow.ai"`
}

// ResetPasswordRequest 重置密码请求。
type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email" example:"author@museflow.ai"`
	Code        string `json:"code" binding:"required" example:"824913"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72" example:"NewP@ss456"`
}

// VerifyMFALoginRequest 登录二次验证请求。
type VerifyMFALoginRequest struct {
	MfaTicket string `json:"mfa_ticket" binding:"required" example:"eyJhbGciOiJIUzI1NiIs..."`
	Code      string `json:"code" binding:"required" example:"824913"`
	// DeviceName 设备名称，用于设备列表展示，可选
	DeviceName string `json:"device_name" binding:"omitempty,max=100" example:"Chrome on Windows"`
}

// SetupMFARequest 生成 TOTP 密钥请求（需登录）。UUID 由网关从 token 注入。
type SetupMFARequest struct{}

// VerifyMFARequest 验证验证码并启用 2FA 请求（需登录）。UUID 由网关从 token 注入。
type VerifyMFARequest struct {
	Code string `json:"code" binding:"required" example:"824913"`
}

// DisableMFARequest 验证验证码后关闭 2FA 请求（需登录）。UUID 由网关从 token 注入。
type DisableMFARequest struct {
	Code string `json:"code" binding:"required" example:"824913"`
}

// RegenerateRecoveryCodesRequest 重新生成恢复码请求（需登录）。UUID 由网关从 token 注入。
type RegenerateRecoveryCodesRequest struct {
	Code string `json:"code" binding:"required" example:"824913"`
}

// SendVerifyCodeRequest 发送邮箱验证码请求。
// Scene 取值：register（注册校验）/ verify（补验证邮箱）/ login（验证码登录）。
type SendVerifyCodeRequest struct {
	Email string `json:"email" binding:"required,email" example:"author@museflow.ai"`
	Scene string `json:"scene" binding:"required,oneof=register verify login" example:"register"`
}

// VerifyEmailRequest 校验邮箱验证码并标记已验证请求。
type VerifyEmailRequest struct {
	Email string `json:"email" binding:"required,email" example:"author@museflow.ai"`
	Code  string `json:"code" binding:"required" example:"824913"`
}

// LoginWithCodeRequest 邮箱验证码登录（免密）请求。
type LoginWithCodeRequest struct {
	Email string `json:"email" binding:"required,email" example:"author@museflow.ai"`
	Code  string `json:"code" binding:"required" example:"824913"`
	// DeviceName 设备名称，用于设备列表展示，可选
	DeviceName string `json:"device_name" binding:"omitempty,max=100" example:"Chrome on Windows"`
}
