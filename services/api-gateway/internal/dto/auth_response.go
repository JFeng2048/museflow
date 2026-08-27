package dto

// UserInfo 用户信息。
type UserInfo struct {
	UUID          string `json:"uuid" example:"3f7c1e2a-5b6d-4e8f-9a0b-1c2d3e4f5a6b"`
	Email         string `json:"email" example:"author@museflow.ai"`
	Phone         string `json:"phone,omitempty" example:"13800138000"`
	Nickname      string `json:"nickname" example:"墨流作者"`
	AvatarURL     string `json:"avatar_url,omitempty"`
	Bio           string `json:"bio,omitempty"`
	Status        int32  `json:"status" example:"1"`
	EmailVerified bool   `json:"email_verified" example:"false"`
	PhoneVerified bool   `json:"phone_verified" example:"false"`
	MFAEnabled    bool   `json:"mfa_enabled" example:"false"`
	LastLoginAt   int64  `json:"last_login_at,omitempty" example:"1756280000"`
	CreatedAt     int64  `json:"created_at" example:"1756270000"`
}

// LoginData 登录成功返回的数据。
// refresh token 不在 body 中返回，而是写入 HttpOnly Cookie。
type LoginData struct {
	AccessToken string   `json:"access_token" example:"eyJhbGciOiJIUzI1NiIs..."`
	TokenType   string   `json:"token_type" example:"Bearer"`
	ExpiresIn   int64    `json:"expires_in" example:"3600"`
	User        UserInfo `json:"user"`
}

// RefreshData 刷新成功返回的数据。
type RefreshData struct {
	AccessToken string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIs..."`
	TokenType   string `json:"token_type" example:"Bearer"`
	ExpiresIn   int64  `json:"expires_in" example:"3600"`
}
