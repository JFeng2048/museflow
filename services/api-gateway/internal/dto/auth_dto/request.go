// Package authdto 定义认证相关接口（注册 / 登录 / 刷新 / 登出 / 密码重置）的 HTTP 请求结构。
package authdto

// RegisterRequest 注册请求。
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email" example:"author@museflow.ai"`
	Password string `json:"password" binding:"required,min=8,max=72" example:"P@ssw0rd123"`
	Nickname string `json:"nickname" binding:"omitempty,max=100" example:"墨流作者"`
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
