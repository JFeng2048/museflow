// Package modeldto 定义模型目录与系统配置相关接口的 HTTP 结构。
//
// 与 proto 对齐但不照搬：HTTP 层把「创建」与「更新」拆成不同结构，
// 因为两者的必填字段不同——更新渠道时 api_key 留空表示不轮换，
// 更新用户资源时 is_active 可空表示不改启用状态。
package modeldto

// ---------- 平台渠道 ----------

// CreateProviderRequest 新增平台渠道请求。
type CreateProviderRequest struct {
	Code         string            `json:"code" binding:"required,max=50" example:"openai"`
	Name         string            `json:"name" binding:"required,max=100" example:"OpenAI 官方"`
	Protocol     string            `json:"protocol" binding:"required,max=50" example:"openai"`
	BaseURL      string            `json:"base_url" binding:"required,max=500" example:"https://api.openai.com/v1"`
	APIKey       string            `json:"api_key" binding:"max=4096" example:"sk-proj-xxx"`
	Organization string            `json:"organization" binding:"omitempty,max=255" example:"org-xxx"`
	Extra        map[string]string `json:"extra"`
	SortOrder    int32             `json:"sort_order" example:"0"`
}

// UpdateProviderRequest 编辑平台渠道请求。
//
// 不含 Code：渠道编码是被业务代码引用的稳定标识，不提供修改入口。
// APIKey 留空表示不轮换密钥。
type UpdateProviderRequest struct {
	Name         string            `json:"name" binding:"required,max=100" example:"OpenAI 官方"`
	Protocol     string            `json:"protocol" binding:"required,max=50" example:"openai"`
	BaseURL      string            `json:"base_url" binding:"required,max=500" example:"https://api.openai.com/v1"`
	APIKey       string            `json:"api_key" binding:"max=4096" example:"sk-proj-xxx"`
	Organization string            `json:"organization" binding:"omitempty,max=255" example:"org-xxx"`
	Extra        map[string]string `json:"extra"`
	SortOrder    int32             `json:"sort_order" example:"0"`
}

// ---------- 平台模型 ----------

// CreateModelRequest 在指定渠道下新增平台模型请求。
type CreateModelRequest struct {
	ProviderID      int64                `json:"provider_id" binding:"required,min=1" example:"1"`
	Code            string               `json:"code" binding:"required,max=50" example:"gpt-4o"`
	Name            string               `json:"name" binding:"required,max=100" example:"GPT-4o"`
	ModelType       string               `json:"model_type" binding:"required,max=50" example:"chat"`
	APIModel        string               `json:"api_model" binding:"required,max=100" example:"gpt-4o"`
	ContextWindow   int32                `json:"context_window" example:"128000"`
	MaxOutputTokens int32                `json:"max_output_tokens" example:"4096"`
	Capabilities    *CapabilitiesRequest `json:"capabilities"`
	CreditCost      int32                `json:"credit_cost" example:"5"`
	Description     string               `json:"description" binding:"omitempty,max=2000" example:"长文续写与润色"`
	SortOrder       int32                `json:"sort_order" example:"0"`
}

// UpdateModelRequest 编辑平台模型请求（code 与渠道均不可改）。
type UpdateModelRequest struct {
	Name            string               `json:"name" binding:"required,max=100" example:"GPT-4o"`
	ModelType       string               `json:"model_type" binding:"required,max=50" example:"chat"`
	APIModel        string               `json:"api_model" binding:"required,max=100" example:"gpt-4o"`
	ContextWindow   int32                `json:"context_window" example:"128000"`
	MaxOutputTokens int32                `json:"max_output_tokens" example:"4096"`
	Capabilities    *CapabilitiesRequest `json:"capabilities"`
	CreditCost      int32                `json:"credit_cost" example:"5"`
	Description     string               `json:"description" binding:"omitempty,max=2000" example:"长文续写与润色"`
	SortOrder       int32                `json:"sort_order" example:"0"`
}

// ---------- 用户自定义 ----------

// UserProviderRequest 新增 / 编辑用户自定义渠道请求。
type UserProviderRequest struct {
	Name         string            `json:"name" binding:"required,max=100" example:"我的 OpenAI"`
	Protocol     string            `json:"protocol" binding:"required,max=50" example:"openai"`
	BaseURL      string            `json:"base_url" binding:"required,max=500" example:"https://api.openai.com/v1"`
	APIKey       string            `json:"api_key" binding:"max=4096" example:"sk-proj-xxx"`
	Organization string            `json:"organization" binding:"omitempty,max=255"`
	Extra        map[string]string `json:"extra"`
	// IsActive 可空：只在显式传入时改动启用状态。
	// 缺失时 proto3 的默认值 false 会被误读成「停用」，因此这里用指针区分。
	IsActive *bool `json:"is_active"`
}

// UserModelRequest 新增 / 编辑用户自定义模型请求。
type UserModelRequest struct {
	ProviderID      int64                `json:"provider_id" binding:"required,min=1" example:"1"`
	Name            string               `json:"name" binding:"required,max=100" example:"GPT-4o"`
	ModelType       string               `json:"model_type" binding:"required,max=50" example:"chat"`
	APIModel        string               `json:"api_model" binding:"required,max=100" example:"gpt-4o"`
	ContextWindow   int32                `json:"context_window" example:"128000"`
	MaxOutputTokens int32                `json:"max_output_tokens" example:"4096"`
	Capabilities    *CapabilitiesRequest `json:"capabilities"`
	Description     string               `json:"description" binding:"omitempty,max=2000"`
	// IsActive 可空，理由同 UserProviderRequest.IsActive。
	IsActive *bool `json:"is_active"`
}

// ---------- 通用系统配置 ----------

// UpsertSettingRequest 写入系统配置请求。
//
// Value 与 SecretValue 二选一：机密配置只填 SecretValue，
// 后端据此决定落 value 列还是加密后的 secret_value 列。
type UpsertSettingRequest struct {
	Value       string `json:"value" binding:"max=2000" example:"100"`
	SecretValue string `json:"secret_value" binding:"max=4096" example:"sk-proj-xxx"`
	IsSecret    bool   `json:"is_secret" example:"false"`
	ValueType   string `json:"value_type" binding:"omitempty,max=50" example:"number"`
	IsPublic    bool   `json:"is_public" example:"true"`
	Description string `json:"description" binding:"omitempty,max=2000" example:"生成任务默认积分单价"`
}

// ---------- 共用 ----------

// SetActiveRequest 启用 / 停用请求。
type SetActiveRequest struct {
	IsActive bool `json:"is_active" example:"true"`
}

// CapabilitiesRequest 模型能力开关请求。
type CapabilitiesRequest struct {
	Stream   bool `json:"stream" example:"true"`
	ToolCall bool `json:"tool_call" example:"false"`
	JSONMode bool `json:"json_mode" example:"true"`
	Vision   bool `json:"vision" example:"false"`
}
