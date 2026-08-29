// Package model 定义 user-service 的数据模型，与 database/user_svc.sql 中的表结构对应。
package model

import (
	"time"

	"github.com/google/uuid"
)

// OAuth 第三方登录关联表，对应 user_svc.oauth。
//
// 一个用户可绑定多个第三方平台；provider + provider_user_id 唯一标识一个第三方账号。
type OAuth struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement"`
	UserUUID        uuid.UUID `gorm:"column:user_uuid;type:uuid;not null;index"`
	Provider        string    `gorm:"column:provider;size:50;not null;index"`
	SSOProvider     *string   `gorm:"column:sso_provider;size:50"`
	ProviderUserID  string    `gorm:"column:provider_user_id;size:255;not null"`
	ProviderEmail   *string   `gorm:"column:provider_email;size:255"`
	ProviderNickname *string  `gorm:"column:provider_nickname;size:100"`
	ProviderAvatar  *string   `gorm:"column:provider_avatar;size:500"`
	AccessToken     *string   `gorm:"column:access_token"`
	RefreshToken    *string   `gorm:"column:refresh_token"`
	ExpiresAt       *time.Time `gorm:"column:expires_at"`
	Extra           string    `gorm:"column:extra;type:jsonb"`
	IsActive        bool      `gorm:"column:is_active;not null;default:true"`
	LastLoginAt     *time.Time `gorm:"column:last_login_at"`
	CreatedAt       time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt       time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

// TableName 指定带 schema 的表名。
func (OAuth) TableName() string { return "user_svc.oauths" }

// 第三方平台常量，与 oauth.provider 字段对应。
const (
	ProviderGitHub = "github"
	ProviderGoogle = "google"
	ProviderWeChat = "wechat"
	ProviderQQ     = "qq"
	ProviderApple  = "apple"
)
