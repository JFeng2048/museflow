package model

import "time"

// 配置值类型常量，对应 system_setting.value_type，供后台表单渲染用。
const (
	ValueTypeString = "string"
	ValueTypeNumber = "number"
	ValueTypeBool   = "bool"
	ValueTypeJSON   = "json"
)

// SystemSetting 通用系统配置表，对应 config_svc.system_setting。
//
// key-value + jsonb，承接「其他系统可变更的配置」：
//   - 非机密值放 Value（jsonb）；
//   - 机密值（令牌 / 密码 / 第三方 secret）放 SecretValue（AES-256-GCM 密文）。
//
// 数据库侧有两条 CHECK 约束保证两者恰好一个非空，且 is_secret 与
// secret_value 是否非空一致，因此写入前必须由 service 层兜好这一层关系。
type SystemSetting struct {
	ID int64 `gorm:"column:id;primaryKey;autoIncrement"`
	// ConfigGroup 配置分组：model / general / notify / quota 等，同组内 key 唯一。
	ConfigGroup string `gorm:"column:config_group;size:50;not null"`
	Key         string `gorm:"column:key;size:100;not null"`

	// Value 非机密配置值，任意 JSON；is_secret=true 时为 NULL。
	Value JSONText `gorm:"column:value;type:jsonb"`
	// SecretValue 机密配置值，AES-256-GCM 密文；is_secret=false 时为 NULL。
	SecretValue *string `gorm:"column:secret_value"`
	IsSecret    bool    `gorm:"column:is_secret;not null;default:false"`
	ValueType   string  `gorm:"column:value_type;size:20;not null;default:json"`
	// IsPublic 是否可下发前端：true 才能随公开配置接口返回。
	IsPublic    bool    `gorm:"column:is_public;not null;default:false"`
	Description *string `gorm:"column:description"`

	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

// TableName 指定带 schema 的表名。
func (SystemSetting) TableName() string { return "config_svc.system_setting" }
