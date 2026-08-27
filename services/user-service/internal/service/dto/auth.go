// Package dto 定义 user-service 业务层内部使用的数据对象（Service 内部 DTO），
// 例如 Device（设备上下文）与 TokenPair（双令牌签发结果）。
package dto

// Device 登录设备上下文。
type Device struct {
	DeviceID   string
	UserAgent  string
	IP         string
	DeviceName string
}

// TokenPair 登录成功后签发的双令牌。
type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	DeviceID         string
	ExpiresIn        int64
	RefreshExpiresIn int64
}
