// Package model 定义 config-service 的数据模型，与 database/config_svc.sql 中的表结构对应。
//
// 注意：数据库 schema 由 database/config_svc.sql 维护（含序列、触发器、schema 命名空间），
// 服务端不执行 AutoMigrate，避免与 SQL 脚本产生冲突。
package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrInvalidJSONValue 写入 jsonb 列的值不是合法 JSON。
var ErrInvalidJSONValue = errors.New("配置值不是合法 JSON")

// 模型类型常量，对应 model.model_type / user_model.model_type。
//
// 用户端与后台共用同一套英文码：前端下拉框、筛选条件都按这些码渲染，
// 不要在这里写中文展示名（展示名由前端维护，避免多语言时改库）。
const (
	ModelTypeChat      = "chat"      // 对话大模型
	ModelTypeEmbedding = "embedding" // 向量化
	ModelTypeRerank    = "rerank"    // 重排序
	ModelTypeVision    = "vision"    // 视觉理解
	ModelTypeImage     = "image"     // 图像生成
	ModelTypeAudio     = "audio"     // 语音
	ModelTypeVideo     = "video"     // 视频
)

// 通讯协议常量，对应 model_provider.protocol / user_model_provider.protocol。
const (
	ProtocolOpenAI    = "openai"    // OpenAI 兼容协议（含国内多数兼容厂商）
	ProtocolAnthropic = "anthropic" // Anthropic Messages
	ProtocolGemini    = "gemini"    // Google Gemini
	ProtocolCustom    = "custom"    // 其他，细节由 extra 描述
)

// Capabilities 模型能力开关，对应 jsonb 列：
//
//	{"stream":true,"tool_call":false,"json_mode":false,"vision":false}
//
// 实现 driver.Valuer / sql.Scanner，直接以 jsonb 读写；
// 四个键总是完整序列化，读取端不必判断缺键。
type Capabilities struct {
	Stream   bool `json:"stream"`
	ToolCall bool `json:"tool_call"`
	JSONMode bool `json:"json_mode"`
	Vision   bool `json:"vision"`
}

// Value 写入 jsonb 列。
func (c Capabilities) Value() (driver.Value, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("序列化模型能力失败: %w", err)
	}
	return string(b), nil
}

// Scan 从 jsonb 列读取。NULL 归零值；非法 JSON 直接报错，便于暴露脏数据。
func (c *Capabilities) Scan(value any) error {
	raw, err := jsonBytes(value)
	if err != nil {
		return err
	}
	if len(raw) == 0 {
		*c = Capabilities{}
		return nil
	}
	if err := json.Unmarshal(raw, c); err != nil {
		return fmt.Errorf("解析模型能力失败: %w", err)
	}
	return nil
}

// StringMap 以 jsonb 存储的字符串映射，用于 model_provider.extra 等协议扩展参数。
type StringMap map[string]string

// Value 写入 jsonb 列。空映射落 NULL，读取端得到 nil，两侧语义一致。
func (m StringMap) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(map[string]string(m))
	if err != nil {
		return nil, fmt.Errorf("序列化扩展参数失败: %w", err)
	}
	return string(b), nil
}

// Scan 从 jsonb 列读取。NULL / JSON null 归零值（nil）。
func (m *StringMap) Scan(value any) error {
	raw, err := jsonBytes(value)
	if err != nil {
		return err
	}
	if len(raw) == 0 || string(raw) == "null" {
		*m = nil
		return nil
	}
	out := make(StringMap)
	if err := json.Unmarshal(raw, &out); err != nil {
		return fmt.Errorf("解析扩展参数失败: %w", err)
	}
	*m = out
	return nil
}

// JSONText 以 jsonb 列存储的原始 JSON 文本，用于 system_setting.value。
//
// 约定：调用方传入的已经是合法 JSON 文本（由 service 层按 value_type 归一化），
// 这里只做兜底校验。
type JSONText string

// Value 写入 jsonb 列。
//
// 空串落 NULL：机密配置的值必须为空（值走 secret_value 列），
// 直接写 `""` 会触发 system_setting 的 CHECK 约束
// （value 与 secret_value 恰好一个非空）。
func (v JSONText) Value() (driver.Value, error) {
	text := strings.TrimSpace(string(v))
	if text == "" {
		return nil, nil
	}
	if !json.Valid([]byte(text)) {
		return nil, ErrInvalidJSONValue
	}
	return text, nil
}

// Scan 从 jsonb 列读取。NULL 得到空串。
func (v *JSONText) Scan(value any) error {
	raw, err := jsonBytes(value)
	if err != nil {
		return err
	}
	if len(raw) == 0 || string(raw) == "null" {
		*v = ""
		return nil
	}
	*v = JSONText(raw)
	return nil
}

// jsonBytes 把数据库驱动返回的 jsonb 值统一成 []byte。
func jsonBytes(value any) ([]byte, error) {
	switch v := value.(type) {
	case nil:
		return nil, nil
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	default:
		return nil, fmt.Errorf("无法识别的 jsonb 值类型 %T", value)
	}
}

// ModelProvider 平台模型渠道表，对应 config_svc.model_provider。
//
// 一行 = 一个上游渠道：厂商 + base_url + api_key，由管理端维护，
// 是「平台提供的」那一半；模型明细挂在其下（config_svc.model）。
type ModelProvider struct {
	ID   int64  `gorm:"column:id;primaryKey;autoIncrement"`
	Code string `gorm:"column:code;size:50;not null;uniqueIndex"`
	Name string `gorm:"column:name;size:100;not null"`
	// Protocol 通讯协议，取值见 Protocol* 常量。
	Protocol string `gorm:"column:protocol;size:50;not null"`
	BaseURL  string `gorm:"column:base_url;size:500;not null"`

	// APIKeyCiphertext AES-256-GCM 密文（v1:nonce:ciphertext），只写不读。
	APIKeyCiphertext string `gorm:"column:api_key;not null"`
	// APIKeyHint 密钥末 4 位，唯一可对外的脱敏信息。
	APIKeyHint string `gorm:"column:api_key_hint;size:8"`
	// APIKeyUpdatedAt 密钥最后一次轮换时间。与 updated_at 区分：
	// 后者任何字段变更都会动，前者只在轮换密钥时刷新。
	APIKeyUpdatedAt *time.Time `gorm:"column:api_key_updated_at"`

	Organization *string   `gorm:"column:organization;size:255"`
	Extra        StringMap `gorm:"column:extra;type:jsonb"`
	IsActive     bool      `gorm:"column:is_active;not null;default:true"`
	SortOrder    int32     `gorm:"column:sort_order;not null;default:0"`

	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

// TableName 指定带 schema 的表名；schema 由 database/config_svc.sql 固定创建为 config_svc。
func (ModelProvider) TableName() string { return "config_svc.model_provider" }

// UserModelProvider 用户自定义模型渠道表，对应 config_svc.user_model_provider。
//
// 与平台渠道的区别：没有 code（不被业务代码引用）、没有 sort_order（用户自有顺序），
// 归属由 user_uuid 逻辑关联 user_svc.user.uuid。
type UserModelProvider struct {
	ID       int64     `gorm:"column:id;primaryKey;autoIncrement"`
	UserUUID uuid.UUID `gorm:"column:user_uuid;type:uuid;not null;index"`

	Name     string `gorm:"column:name;size:100;not null"`
	Protocol string `gorm:"column:protocol;size:50;not null"`
	BaseURL  string `gorm:"column:base_url;size:500;not null"`

	APIKeyCiphertext string     `gorm:"column:api_key;not null"`
	APIKeyHint       string     `gorm:"column:api_key_hint;size:8"`
	APIKeyUpdatedAt  *time.Time `gorm:"column:api_key_updated_at"`

	Organization *string   `gorm:"column:organization;size:255"`
	Extra        StringMap `gorm:"column:extra;type:jsonb"`
	IsActive     bool      `gorm:"column:is_active;not null;default:true"`

	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

// TableName 指定带 schema 的表名。
func (UserModelProvider) TableName() string { return "config_svc.user_model_provider" }
