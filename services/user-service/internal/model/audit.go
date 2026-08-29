// Package model 定义 user-service 的数据模型，与 database/user_svc.sql 中的表结构对应。
package model

import (
	"time"

	"github.com/google/uuid"
)

// AuditLog 用户操作审计日志表，对应 user_svc.audit_log。
//
// 记录注册、登录、登出、改密、分配角色等关键操作，
// 供管理后台按用户 / 时间 / 操作类型筛选查询。
type AuditLog struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	UserUUID   uuid.UUID `gorm:"column:user_uuid;type:uuid;index"`
	Action     string    `gorm:"column:action;size:100;not null;index"`
	Resource   string    `gorm:"column:resource;size:50;not null;index"`
	ResourceID string    `gorm:"column:resource_id;size:100"`
	IP         *string   `gorm:"column:ip;type:inet"`
	UserAgent  string    `gorm:"column:user_agent"`
	Detail     string    `gorm:"column:detail;type:jsonb"`
	CreatedAt  time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

// TableName 指定带 schema 的表名。
func (AuditLog) TableName() string { return "user_svc.audit_logs" }

// 审计操作类型常量，与 audit_log.action 字段对应。
const (
	AuditActionRegister    = "register"
	AuditActionLogin       = "login"
	AuditActionLoginFail   = "login_fail"
	AuditActionLogout      = "logout"
	AuditActionUpdateUser  = "update_user"
	AuditActionChangePwd   = "change_password"
	AuditActionAssignRole  = "assign_role"
	AuditActionUpdateStat  = "update_status"
	AuditActionCreateRole  = "create_role"
	AuditActionUpdateRole  = "update_role"
	AuditActionDeleteRole  = "delete_role"
	AuditActionSetRolePerm = "set_role_permissions"
)

// 审计资源类型常量，与 audit_log.resource 字段对应。
const (
	AuditResourceUser  = "user"
	AuditResourceRole  = "role"
	AuditResourcePerm  = "permission"
	AuditResourceAuth  = "auth"
)
