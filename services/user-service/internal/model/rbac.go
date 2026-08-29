// Package model 定义 user-service 的数据模型，与 database/user_svc.sql 中的表结构对应。
//
// 注意：数据库 schema 由 database/user_svc.sql 维护（含序列、触发器、schema 命名空间），
// 服务端不执行 AutoMigrate，避免与 SQL 脚本产生冲突。
package model

import (
	"time"

	"github.com/google/uuid"
)

// Role 角色定义表，对应 user_svc.role。
type Role struct {
	ID          int16     `gorm:"column:id;primaryKey"`
	Code        string    `gorm:"column:code;size:50;not null;uniqueIndex"`
	Name        string    `gorm:"column:name;size:100;not null"`
	Description string    `gorm:"column:description"`
	IsSystem    bool      `gorm:"column:is_system;not null;default:false"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

// TableName 指定带 schema 的表名。
func (Role) TableName() string { return "user_svc.roles" }

// Permission 权限定义表，对应 user_svc.permission。
type Permission struct {
	ID          int16     `gorm:"column:id;primaryKey"`
	Code        string    `gorm:"column:code;size:100;not null;uniqueIndex"`
	Name        string    `gorm:"column:name;size:100;not null"`
	Resource    string    `gorm:"column:resource;size:50;not null"`
	Action      string    `gorm:"column:action;size:50;not null"`
	Description string    `gorm:"column:description"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

// TableName 指定带 schema 的表名。
func (Permission) TableName() string { return "user_svc.permissions" }

// RolePermission 角色-权限关联表，对应 user_svc.role_permission。
type RolePermission struct {
	RoleID       int16     `gorm:"column:role_id;primaryKey"`
	PermissionID int16     `gorm:"column:permission_id;primaryKey"`
	CreatedAt    time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

// TableName 指定带 schema 的表名。
func (RolePermission) TableName() string { return "user_svc.role_permissions" }

// UserRole 用户-角色关联表，对应 user_svc.user_roles。
type UserRole struct {
	UserUUID uuid.UUID `gorm:"column:user_uuid;type:uuid;primaryKey"`
	RoleID   int16     `gorm:"column:role_id;primaryKey"`
	GrantedBy uuid.UUID `gorm:"column:granted_by;type:uuid"`
	GrantedAt time.Time `gorm:"column:granted_at;not null;autoCreateTime"`
}

// TableName 指定带 schema 的表名。
func (UserRole) TableName() string { return "user_svc.user_roles" }
