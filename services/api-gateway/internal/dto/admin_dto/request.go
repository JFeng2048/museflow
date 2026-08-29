// Package admindto 定义管理后台接口（用户 / 角色 / 权限 / 审计日志）的 HTTP 请求结构。
package admindto

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
