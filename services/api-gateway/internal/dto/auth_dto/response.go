package authdto

import (
	"github.com/museflow/api-gateway/internal/dto/user_dto"
)

// LoginData 登录成功返回的数据。
// refresh token 不在 body 中返回，而是写入 HttpOnly Cookie。
type LoginData struct {
	AccessToken string           `json:"access_token" example:"eyJhbGciOiJIUzI1NiIs..."`
	TokenType   string           `json:"token_type" example:"Bearer"`
	ExpiresIn   int64            `json:"expires_in" example:"3600"`
	User        userdto.UserInfo `json:"user"`
}

// RefreshData 刷新成功返回的数据。
type RefreshData struct {
	AccessToken string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIs..."`
	TokenType   string `json:"token_type" example:"Bearer"`
	ExpiresIn   int64  `json:"expires_in" example:"3600"`
}

// LoginData 兼容登录与 2FA 二次验证的返回结构。
// 账号开启 2FA 时，access_token 为空，requires_mfa=true，mfa_ticket 用于后续 VerifyMFALogin。
type LoginResponseData struct {
	AccessToken          string           `json:"access_token,omitempty" example:"eyJhbGciOiJIUzI1NiIs..."`
	TokenType            string           `json:"token_type,omitempty" example:"Bearer"`
	ExpiresIn            int64            `json:"expires_in,omitempty" example:"3600"`
	User                 userdto.UserInfo `json:"user"`
	RequiresMFA          bool             `json:"requires_mfa" example:"false"`
	MFATicket            string           `json:"mfa_ticket,omitempty" example:"eyJhbGciOiJIUzI1NiIs..."`
	RecoveryCodes        []string         `json:"recovery_codes,omitempty" example:"[\"a1b2c3d4\",\"e5f6g7h8\"]"`
	RemainingRecoveryCodes int32          `json:"remaining_recovery_codes,omitempty" example:"8"`
}

// MFASetupData 生成 TOTP 密钥返回的数据。
type MFASetupData struct {
	Secret     string `json:"secret" example:"JBSWY3DPEHPK3PXP"`
	OtpauthURL string `json:"otpauth_url" example:"otpauth://totp/MuseFlow:author@museflow.ai?secret=JBSWY3DPEHPK3PXP&issuer=MuseFlow"`
}

// MFAStatusData 2FA 状态返回的数据。
type MFAStatusData struct {
	Enabled                bool  `json:"enabled" example:"true"`
	RemainingRecoveryCodes int32 `json:"remaining_recovery_codes" example:"8"`
}
