// Package model 定义 user-service 的数据模型，与 database/user_svc.sql 中的表结构对应。
//
// 注意：数据库 schema 由 database/user_svc.sql 维护（含序列、触发器、schema 命名空间），
// 服务端不执行 AutoMigrate，避免与 SQL 脚本产生冲突。
package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// 用户状态常量，对应 users.status 字段。
const (
	StatusNormal    int16 = 1 // 正常
	StatusFrozen    int16 = 2 // 冻结
	StatusDeleted   int16 = 3 // 已注销
	StatusPendinAdt int16 = 4 // 待审核
)

// User 用户主表，对应 user_svc.user。
//
// 字段说明：
//   - ID 为自增主键，仅内部使用，不对外暴露；
//   - UUID 为对外唯一标识，同时作为 JWT 的 subject；
//   - PasswordHash 可为空（第三方登录用户），故使用指针类型。
type User struct {
	ID   int64     `gorm:"column:id;primaryKey;autoIncrement"`
	UUID uuid.UUID `gorm:"column:uuid;type:uuid;default:gen_random_uuid();uniqueIndex"`

	Email        string  `gorm:"column:email;size:255;not null;uniqueIndex"`
	Phone        *string `gorm:"column:phone;size:20"`
	PasswordHash *string `gorm:"column:password_hash;size:255"`

	Nickname  string  `gorm:"column:nickname;size:100;not null"`
	AvatarURL *string `gorm:"column:avatar_url;size:500"`
	Bio       *string `gorm:"column:bio"`

	// MFA 相关字段：本期不实现校验逻辑，仅保留映射。
	MFAEnabled       bool           `gorm:"column:mfa_enabled;not null;default:false"`
	MFASecret        *string        `gorm:"column:mfa_secret;size:100"`
	MFARecoveryCodes pq.StringArray `gorm:"column:mfa_recovery_codes;type:text[]"`

	Status        int16 `gorm:"column:status;not null;default:1"`
	EmailVerified bool  `gorm:"column:email_verified;not null;default:false"`
	PhoneVerified bool  `gorm:"column:phone_verified;not null;default:false"`

	LastLoginAt       *time.Time `gorm:"column:last_login_at"`
	LastLoginIP       *string    `gorm:"column:last_login_ip;type:inet"`
	LastLoginPlatform *string    `gorm:"column:last_login_platform;size:50"`

	// 登录失败计数与锁定时间：本期不参与登录流程，仅保留字段映射。
	LoginFailCount int32      `gorm:"column:login_fail_count;not null;default:0"`
	LockedUntil    *time.Time `gorm:"column:locked_until"`

	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

// TableName 指定带 schema 的表名；schema 由 database/user_svc.sql 固定创建为 user_svc。
func (User) TableName() string {
	return "user_svc.user"
}

// IsActive 判断账号是否处于正常状态。
func (u *User) IsActive() bool {
	return u.Status == StatusNormal
}
