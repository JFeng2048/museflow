// Package authdto 定义认证相关接口（注册 / 登录 / 刷新 / 登出）的 HTTP 请求结构。
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
