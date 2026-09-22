package modeldto

// Capabilities 模型能力开关。
type Capabilities struct {
	Stream   bool `json:"stream" example:"true"`
	ToolCall bool `json:"tool_call" example:"false"`
	JSONMode bool `json:"json_mode" example:"true"`
	Vision   bool `json:"vision" example:"false"`
}

// ProviderInfo 平台渠道。
//
// api_key 是只写字段：响应只带 api_key_hint（末 4 位）与轮换时间，
// 没有任何字段能还原出完整密钥。
type ProviderInfo struct {
	ID              int64             `json:"id" example:"1"`
	Code            string            `json:"code" example:"openai"`
	Name            string            `json:"name" example:"OpenAI 官方"`
	Protocol        string            `json:"protocol" example:"openai"`
	BaseURL         string            `json:"base_url" example:"https://api.openai.com/v1"`
	APIKeyHint      string            `json:"api_key_hint" example:"c7Xk"`
	APIKeyUpdatedAt string            `json:"api_key_updated_at" example:"2026-09-22T08:00:00Z"`
	Organization    string            `json:"organization,omitempty" example:"org-xxx"`
	Extra           map[string]string `json:"extra,omitempty"`
	IsActive        bool              `json:"is_active" example:"true"`
	SortOrder       int32             `json:"sort_order" example:"0"`
	CreatedAt       string            `json:"created_at" example:"2026-09-22T08:00:00Z"`
	UpdatedAt       string            `json:"updated_at" example:"2026-09-22T08:00:00Z"`
}

// ModelInfo 平台模型。
type ModelInfo struct {
	ID              int64        `json:"id" example:"1"`
	ProviderID      int64        `json:"provider_id" example:"1"`
	Code            string       `json:"code" example:"gpt-4o"`
	Name            string       `json:"name" example:"GPT-4o"`
	ModelType       string       `json:"model_type" example:"chat"`
	APIModel        string       `json:"api_model" example:"gpt-4o"`
	ContextWindow   int32        `json:"context_window" example:"128000"`
	MaxOutputTokens int32        `json:"max_output_tokens" example:"4096"`
	Capabilities    Capabilities `json:"capabilities"`
	CreditCost      int32        `json:"credit_cost" example:"5"`
	Description     string       `json:"description,omitempty"`
	IsActive        bool         `json:"is_active" example:"true"`
	SortOrder       int32        `json:"sort_order" example:"0"`
	CreatedAt       string       `json:"created_at" example:"2026-09-22T08:00:00Z"`
	UpdatedAt       string       `json:"updated_at" example:"2026-09-22T08:00:00Z"`
}

// UserProviderInfo 用户自定义渠道。
type UserProviderInfo struct {
	ID              int64             `json:"id" example:"1"`
	UserUUID        string            `json:"user_uuid" example:"3f7c1e2a-5b6d-4e8f-9a0b-1c2d3e4f5a6b"`
	Name            string            `json:"name" example:"我的 OpenAI"`
	Protocol        string            `json:"protocol" example:"openai"`
	BaseURL         string            `json:"base_url" example:"https://api.openai.com/v1"`
	APIKeyHint      string            `json:"api_key_hint" example:"c7Xk"`
	APIKeyUpdatedAt string            `json:"api_key_updated_at" example:"2026-09-22T08:00:00Z"`
	Organization    string            `json:"organization,omitempty"`
	Extra           map[string]string `json:"extra,omitempty"`
	IsActive        bool              `json:"is_active" example:"true"`
	CreatedAt       string            `json:"created_at" example:"2026-09-22T08:00:00Z"`
	UpdatedAt       string            `json:"updated_at" example:"2026-09-22T08:00:00Z"`
}

// UserModelInfo 用户自定义模型。
type UserModelInfo struct {
	ID              int64        `json:"id" example:"1"`
	UserUUID        string       `json:"user_uuid" example:"3f7c1e2a-5b6d-4e8f-9a0b-1c2d3e4f5a6b"`
	UserProviderID  int64        `json:"user_provider_id" example:"1"`
	Name            string       `json:"name" example:"GPT-4o"`
	ModelType       string       `json:"model_type" example:"chat"`
	APIModel        string       `json:"api_model" example:"gpt-4o"`
	ContextWindow   int32        `json:"context_window" example:"128000"`
	MaxOutputTokens int32        `json:"max_output_tokens" example:"4096"`
	Capabilities    Capabilities `json:"capabilities"`
	Description     string       `json:"description,omitempty"`
	IsActive        bool         `json:"is_active" example:"true"`
	CreatedAt       string       `json:"created_at" example:"2026-09-22T08:00:00Z"`
	UpdatedAt       string       `json:"updated_at" example:"2026-09-22T08:00:00Z"`
}

// AvailableModel 用户端可用模型。
//
// 刻意不含 base_url 与任何密钥：用户端只能据此「选模型」，
// 真正的上游调用由服务端发出。
type AvailableModel struct {
	Source          string       `json:"source" example:"platform"`
	ModelID         int64        `json:"model_id" example:"1"`
	ProviderID      int64        `json:"provider_id" example:"1"`
	Name            string       `json:"name" example:"GPT-4o"`
	ModelType       string       `json:"model_type" example:"chat"`
	APIModel        string       `json:"api_model" example:"gpt-4o"`
	ContextWindow   int32        `json:"context_window" example:"128000"`
	MaxOutputTokens int32        `json:"max_output_tokens" example:"4096"`
	Capabilities    Capabilities `json:"capabilities"`
	CreditCost      int32        `json:"credit_cost" example:"5"`
	Description     string       `json:"description,omitempty"`
}

// SettingInfo 系统配置。
//
// SecretValue 只在写入响应中出现一次，列表与详情查询恒为空，
// 前端据此把已有机密显示为「已配置」。
type SettingInfo struct {
	ID          int64  `json:"id" example:"1"`
	ConfigGroup string `json:"config_group" example:"generation"`
	Key         string `json:"key" example:"credit_price"`
	Value       string `json:"value,omitempty" example:"100"`
	SecretValue string `json:"secret_value,omitempty" example:"sk-proj-xxx"`
	IsSecret    bool   `json:"is_secret" example:"false"`
	ValueType   string `json:"value_type" example:"number"`
	IsPublic    bool   `json:"is_public" example:"true"`
	Description string `json:"description,omitempty" example:"生成任务默认积分单价"`
	CreatedAt   string `json:"created_at" example:"2026-09-22T08:00:00Z"`
	UpdatedAt   string `json:"updated_at" example:"2026-09-22T08:00:00Z"`
}

// ProviderList 平台渠道分页结果。
type ProviderList struct {
	Items    []ProviderInfo `json:"items"`
	Total    int64          `json:"total" example:"1"`
	Page     int32          `json:"page" example:"1"`
	PageSize int32          `json:"page_size" example:"20"`
}

// ModelList 平台模型分页结果。
type ModelList struct {
	Items    []ModelInfo `json:"items"`
	Total    int64       `json:"total" example:"2"`
	Page     int32       `json:"page" example:"1"`
	PageSize int32       `json:"page_size" example:"20"`
}

// UserProviderList 用户自定义渠道列表。
type UserProviderList struct {
	Items    []UserProviderInfo `json:"items"`
	Total    int64              `json:"total" example:"1"`
	Page     int32              `json:"page" example:"1"`
	PageSize int32              `json:"page_size" example:"20"`
}

// UserModelList 用户自定义模型分页结果。
type UserModelList struct {
	Items    []UserModelInfo `json:"items"`
	Total    int64           `json:"total" example:"1"`
	Page     int32           `json:"page" example:"1"`
	PageSize int32           `json:"page_size" example:"20"`
}

// SettingList 系统配置列表。
type SettingList struct {
	Items []SettingInfo `json:"items"`
}

// RemoteModelInfo 上游模型目录条目。
//
// 字段刻意与已登记模型对齐：勾选后 id 直接作为 api_model，
// object 可辅助判断该模型是对话模型还是向量模型。
type RemoteModelInfo struct {
	ID        string `json:"id" example:"gpt-4o"`
	Object    string `json:"object" example:"model"`
	OwnedBy   string `json:"owned_by" example:"openai"`
	CreatedAt string `json:"created_at" example:"2024-06-01T00:00:00Z"`
}

// RemoteModelList 上游模型目录结果。
type RemoteModelList struct {
	Items []RemoteModelInfo `json:"items"`
	Total int32             `json:"total" example:"42"`
}

// AvailableModelList 用户端可用模型列表。
type AvailableModelList struct {
	Items []AvailableModel `json:"items"`
}

// DeleteResult 删除结果。
type DeleteResult struct {
	Success bool `json:"success" example:"true"`
}
