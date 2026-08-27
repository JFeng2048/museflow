// Package token 实现 JWT 的签发、解析与设备指纹计算。
package token

import (
	"github.com/golang-jwt/jwt/v5"
)

// 令牌类型常量，写入 JWT 的 type 声明，防止 refresh token 被当作 access token 使用。
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// AccessClaims access token 载荷。
// Subject 为用户 uuid，ID(jti) 用于登出时加入黑名单。
type AccessClaims struct {
	Type string `json:"type"`
	jwt.RegisteredClaims
}

// RefreshClaims refresh token 载荷。
// 携带设备信息用于防止令牌被窃取后跨设备使用。
type RefreshClaims struct {
	Type              string `json:"type"`
	TokenID           string `json:"tokenId"`
	DeviceID          string `json:"deviceId"`
	DeviceFingerprint string `json:"deviceFingerprint"`
	jwt.RegisteredClaims
}
