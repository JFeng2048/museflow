package userdto

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

// SessionInfo 登录会话信息。
type SessionInfo struct {
	TokenID         string `json:"token_id" example:"9f8e7d6c-5b4a-3928-1706-0f1e2d3c4b5a"`
	DeviceID        string `json:"device_id" example:"a1b2c3d4e5f6"`
	DeviceName      string `json:"device_name" example:"Chrome on Windows"`
	LoginTime       int64  `json:"login_time" example:"1756280000"`
	LastRefreshTime int64  `json:"last_refresh_time" example:"1756283600"`
}

// SessionList 会话列表。
type SessionList struct {
	Sessions []SessionInfo `json:"sessions"`
}

// PermissionListData 当前用户权限列表。
type PermissionListData struct {
	Permissions []string `json:"permissions" example:"user:read,novel:write"`
}

// PermissionCheck 权限校验结果。
type PermissionCheck struct {
	Allowed bool `json:"allowed" example:"true"`
}

// OAuthBinding 已绑定的第三方账号。
type OAuthBinding struct {
	Provider         string `json:"provider" example:"github"`
	ProviderUserID   string `json:"provider_user_id" example:"gh-123456"`
	ProviderEmail    string `json:"provider_email,omitempty" example:"dev@github.com"`
	ProviderNickname string `json:"provider_nickname,omitempty" example:"octocat"`
	ProviderAvatar   string `json:"provider_avatar,omitempty"`
	LastLoginAt      int64  `json:"last_login_at,omitempty" example:"1756280000"`
	CreatedAt        int64  `json:"created_at" example:"1756270000"`
}

// OAuthBindingList 第三方账号绑定列表。
type OAuthBindingList struct {
	Bindings []OAuthBinding `json:"bindings"`
}
