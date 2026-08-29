// Package userdto 定义用户自助接口（资料 / 密码 / 会话 / 权限 / 第三方绑定）的 HTTP 结构。
package userdto

// UpdateProfileRequest 更新个人信息请求（留空表示不修改该字段）。
type UpdateProfileRequest struct {
	Nickname  string `json:"nickname" binding:"omitempty,max=100" example:"墨流作者"`
	AvatarURL string `json:"avatar_url" binding:"omitempty,url" example:"https://cdn.museflow.ai/avatar/1.png"`
	Bio       string `json:"bio" binding:"omitempty,max=500" example:"专注于玄幻小说创作"`
}

// ChangePasswordRequest 修改密码请求。
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required" example:"OldP@ss123"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72" example:"NewP@ss456"`
}

// RevokeSessionRequest 强制下线指定会话请求。
type RevokeSessionRequest struct {
	TokenID string `json:"token_id" binding:"required" example:"9f8e7d6c-5b4a-3928-1706-0f1e2d3c4b5a"`
}
