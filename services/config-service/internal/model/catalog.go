package model

import (
	"time"

	"github.com/google/uuid"
)

// Model 平台模型明细表，对应 config_svc.model。
//
// 挂在某个平台渠道之下，是用户端「刷出来」的可用模型主体：
// model_type 区分对话大模型 / 向量化 / 重排序等，api_model 是真正传给上游的标识。
type Model struct {
	ID         int64 `gorm:"column:id;primaryKey;autoIncrement"`
	ProviderID int64 `gorm:"column:provider_id;not null;index"`
	// Code 模型唯一编码，供业务代码引用（如 muse-pro），与上游无关。
	Code string `gorm:"column:code;size:100;not null;uniqueIndex"`
	Name string `gorm:"column:name;size:100;not null"`
	// ModelType 模型类型，取值见 ModelType* 常量。
	ModelType string `gorm:"column:model_type;size:50;not null;index"`
	// APIModel 实际调用时传给上游的模型标识，如 gpt-4o。
	APIModel        string       `gorm:"column:api_model;size:100;not null"`
	ContextWindow   int32        `gorm:"column:context_window;not null;default:0"`
	MaxOutputTokens *int32       `gorm:"column:max_output_tokens"`
	Capabilities    Capabilities `gorm:"column:capabilities;type:jsonb"`
	// CreditCost 单次调用消耗的平台积分，0 表示免费。
	CreditCost  int32   `gorm:"column:credit_cost;not null;default:0"`
	Description *string `gorm:"column:description"`
	IsActive    bool    `gorm:"column:is_active;not null;default:true"`
	SortOrder   int32   `gorm:"column:sort_order;not null;default:0"`

	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

// TableName 指定带 schema 的表名。
func (Model) TableName() string { return "config_svc.model" }

// UserModel 用户自选模型表，对应 config_svc.user_model。
//
// 只能挂在用户自己的渠道下；没有 credit_cost 字段，即约定自定义模型
// 不消耗平台积分（用户直接向上游付费）。
type UserModel struct {
	ID       int64     `gorm:"column:id;primaryKey;autoIncrement"`
	UserUUID uuid.UUID `gorm:"column:user_uuid;type:uuid;not null;index"`
	// UserProviderID 关联 config_svc.user_model_provider.id（逻辑关联，无物理外键）。
	UserProviderID  int64        `gorm:"column:user_provider_id;not null;index"`
	Name            string       `gorm:"column:name;size:100;not null"`
	ModelType       string       `gorm:"column:model_type;size:50;not null;index"`
	APIModel        string       `gorm:"column:api_model;size:100;not null"`
	ContextWindow   int32        `gorm:"column:context_window;not null;default:0"`
	MaxOutputTokens *int32       `gorm:"column:max_output_tokens"`
	Capabilities    Capabilities `gorm:"column:capabilities;type:jsonb"`
	Description     *string      `gorm:"column:description"`
	IsActive        bool         `gorm:"column:is_active;not null;default:true"`

	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

// TableName 指定带 schema 的表名。
func (UserModel) TableName() string { return "config_svc.user_model" }
