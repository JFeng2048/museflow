package admindto

import (
	"github.com/museflow/api-gateway/internal/dto/user_dto"
)

// AdminUserItem 管理后台用户列表项（用户 + 角色）。
type AdminUserItem struct {
	User      userdto.UserInfo `json:"user"`
	Roles     []string         `json:"roles" example:"admin,user"`
	UpdatedAt int64            `json:"updated_at" example:"1756280000"`
}

// UserList 管理后台用户列表（带分页）。
type UserList struct {
	Users    []AdminUserItem `json:"users"`
	Total    int64           `json:"total" example:"128"`
	Page     int32           `json:"page" example:"1"`
	PageSize int32           `json:"page_size" example:"20"`
}

// UserDetail 管理后台用户详情（含最终权限）。
type UserDetail struct {
	User        AdminUserItem `json:"user"`
	Permissions []string      `json:"permissions" example:"user:read,novel:write"`
}

// RoleInfo 角色信息。
type RoleInfo struct {
	ID          int32  `json:"id" example:"1"`
	Code        string `json:"code" example:"super_admin"`
	Name        string `json:"name" example:"超级管理员"`
	Description string `json:"description" example:"拥有全部系统权限"`
	IsSystem    bool   `json:"is_system" example:"true"`
	CreatedAt   int64  `json:"created_at" example:"1756270000"`
}

// RoleList 角色列表。
type RoleList struct {
	Roles []RoleInfo `json:"roles"`
}

// PermissionInfo 权限信息。
type PermissionInfo struct {
	ID          int32  `json:"id" example:"1"`
	Code        string `json:"code" example:"user:read"`
	Name        string `json:"name" example:"查看用户"`
	Resource    string `json:"resource" example:"user"`
	Action      string `json:"action" example:"read"`
	Description string `json:"description" example:"查看用户列表与详情"`
}

// PermissionList 权限列表。
type PermissionList struct {
	Permissions []PermissionInfo `json:"permissions"`
}

// AuditLogItem 审计日志条目。
type AuditLogItem struct {
	ID         int64  `json:"id" example:"1"`
	UserUUID   string `json:"user_uuid" example:"3f7c1e2a-5b6d-4e8f-9a0b-1c2d3e4f5a6b"`
	Action     string `json:"action" example:"login"`
	Resource   string `json:"resource" example:"auth"`
	ResourceID string `json:"resource_id" example:"3f7c1e2a-5b6d-4e8f-9a0b-1c2d3e4f5a6b"`
	IP         string `json:"ip,omitempty" example:"203.0.113.10"`
	UserAgent  string `json:"user_agent,omitempty"`
	Detail     string `json:"detail,omitempty" example:"{\"reason\":\"bad_password\"}"`
	CreatedAt  int64  `json:"created_at" example:"1756280000"`
}

// AuditLogList 审计日志列表（带分页）。
type AuditLogList struct {
	Logs     []AuditLogItem `json:"logs"`
	Total    int64          `json:"total" example:"1024"`
	Page     int32          `json:"page" example:"1"`
	PageSize int32          `json:"page_size" example:"20"`
}
