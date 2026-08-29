package dto

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

// ==================== 管理后台 ====================

// UpdateUserStatusRequest 修改用户状态请求。
// 状态：1=正常, 2=冻结, 3=已注销, 4=待审核
type UpdateUserStatusRequest struct {
	Status int32 `json:"status" binding:"required,oneof=1 2 3 4" example:"2"`
}

// AssignRoleRequest 分配角色请求。
type AssignRoleRequest struct {
	RoleCode string `json:"role_code" binding:"required,max=50" example:"admin"`
}

// CreateRoleRequest 创建角色请求。
type CreateRoleRequest struct {
	Code        string `json:"code" binding:"required,max=50" example:"auditor"`
	Name        string `json:"name" binding:"required,max=100" example:"审核员"`
	Description string `json:"description" binding:"omitempty,max=500" example:"负责内容审核"`
}

// UpdateRoleRequest 编辑角色请求。
type UpdateRoleRequest struct {
	Name        string `json:"name" binding:"required,max=100" example:"审核员"`
	Description string `json:"description" binding:"omitempty,max=500" example:"负责内容审核"`
}

// SetRolePermissionsRequest 为角色分配权限请求（覆盖式）。
type SetRolePermissionsRequest struct {
	PermissionCodes []string `json:"permission_codes" binding:"required,min=1" example:"novel:read,novel:write"`
}

// Pagination 分页信息。
type Pagination struct {
	Page     int32 `json:"page" example:"1"`
	PageSize int32 `json:"page_size" example:"20"`
	Total    int64 `json:"total" example:"128"`
}
