package handler

import (
	"time"

	modelpb "github.com/museflow/proto/model"

	"github.com/museflow/config-service/internal/model"
	"github.com/museflow/config-service/internal/service"
)

// formatTime 把时间统一序列化为 RFC3339（UTC），空时间返回空串。
//
// proto 里时间字段用 string 而非 int64：本服务的调用方是 api-gateway，
// 前端要做「多久前更新」的展示，字符串比秒级时间戳更贴近展示层。
// 取 UTC 是因为库里存的是数据库会话时区，统一成 UTC 才能让多实例部署下
// 不同机器返回同一份文本。
func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// optionalString 把可空字符串还原成字符串，空值取空串。
func optionalString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// optionalInt32 把可空 int32 还原成值，空值取 0。
//
// proto 的 int32 没有「未设置」状态，0 在建模型时被定义为「跟随上游默认」，
// 与 service 层 optionalInt32 的约定正好闭环。
func optionalInt32(v *int32) int32 {
	if v == nil {
		return 0
	}
	return *v
}

// toStringMap 把 jsonb 映射转成 proto 的 map<string,string>。
// nil 映射直接返回 nil，proto 序列化时会省略该字段，响应更紧凑。
func toStringMap(m model.StringMap) map[string]string {
	if len(m) == 0 {
		return nil
	}
	return map[string]string(m)
}

// toModelCapabilities 把 proto 的能力开关转成领域值类型。
//
// nil 取零值（全部关闭）：proto3 的消息字段不带「未设置」状态，
// 后台表单不传该字段时的意图就是「一个都不开」。
func toModelCapabilities(c *modelpb.ModelCapabilities) model.Capabilities {
	if c == nil {
		return model.Capabilities{}
	}
	return model.Capabilities{
		Stream:   c.GetStream(),
		ToolCall: c.GetToolCall(),
		JSONMode: c.GetJsonMode(),
		Vision:   c.GetVision(),
	}
}

// toCapabilities 把能力开关转成 proto 消息。
func toCapabilities(c model.Capabilities) *modelpb.ModelCapabilities {
	return &modelpb.ModelCapabilities{
		Stream:   c.Stream,
		ToolCall: c.ToolCall,
		JsonMode: c.JSONMode,
		Vision:   c.Vision,
	}
}

// toProviderInfo 转换平台渠道。
//
// 只带 api_key_hint 与 api_key_updated_at，不带密文：api_key 是只写字段，
// 这里没有任何变量能承载它，契约上就不可能泄漏。
func toProviderInfo(p *model.ModelProvider) *modelpb.ProviderInfo {
	return &modelpb.ProviderInfo{
		Id:              p.ID,
		Code:            p.Code,
		Name:            p.Name,
		Protocol:        p.Protocol,
		BaseUrl:         p.BaseURL,
		ApiKeyHint:      p.APIKeyHint,
		ApiKeyUpdatedAt: formatTime(p.APIKeyUpdatedAt),
		Organization:    optionalString(p.Organization),
		Extra:           toStringMap(p.Extra),
		IsActive:        p.IsActive,
		SortOrder:       p.SortOrder,
		CreatedAt:       formatTime(&p.CreatedAt),
		UpdatedAt:       formatTime(&p.UpdatedAt),
	}
}

// toModelInfo 转换平台模型。
func toModelInfo(m *model.Model) *modelpb.ModelInfo {
	return &modelpb.ModelInfo{
		Id:              m.ID,
		ProviderId:      m.ProviderID,
		Code:            m.Code,
		Name:            m.Name,
		ModelType:       m.ModelType,
		ApiModel:        m.APIModel,
		ContextWindow:   m.ContextWindow,
		MaxOutputTokens: optionalInt32(m.MaxOutputTokens),
		Capabilities:    toCapabilities(m.Capabilities),
		CreditCost:      m.CreditCost,
		Description:     optionalString(m.Description),
		IsActive:        m.IsActive,
		SortOrder:       m.SortOrder,
		CreatedAt:       formatTime(&m.CreatedAt),
		UpdatedAt:       formatTime(&m.UpdatedAt),
	}
}

// toUserProviderInfo 转换用户自定义渠道。
func toUserProviderInfo(p *model.UserModelProvider) *modelpb.UserProviderInfo {
	return &modelpb.UserProviderInfo{
		Id:              p.ID,
		UserUuid:        p.UserUUID.String(),
		Name:            p.Name,
		Protocol:        p.Protocol,
		BaseUrl:         p.BaseURL,
		ApiKeyHint:      p.APIKeyHint,
		ApiKeyUpdatedAt: formatTime(p.APIKeyUpdatedAt),
		Organization:    optionalString(p.Organization),
		Extra:           toStringMap(p.Extra),
		IsActive:        p.IsActive,
		CreatedAt:       formatTime(&p.CreatedAt),
		UpdatedAt:       formatTime(&p.UpdatedAt),
	}
}

// toUserModelInfo 转换用户自定义模型。
func toUserModelInfo(m *model.UserModel) *modelpb.UserModelInfo {
	return &modelpb.UserModelInfo{
		Id:              m.ID,
		UserUuid:        m.UserUUID.String(),
		UserProviderId:  m.UserProviderID,
		Name:            m.Name,
		ModelType:       m.ModelType,
		ApiModel:        m.APIModel,
		ContextWindow:   m.ContextWindow,
		MaxOutputTokens: optionalInt32(m.MaxOutputTokens),
		Capabilities:    toCapabilities(m.Capabilities),
		Description:     optionalString(m.Description),
		IsActive:        m.IsActive,
		CreatedAt:       formatTime(&m.CreatedAt),
		UpdatedAt:       formatTime(&m.UpdatedAt),
	}
}

// toAvailableModel 转换用户端可用模型。
//
// 与 service 层的约定一致：这里拿不到 base_url 与 api_key，
// 用户端只能用来「选模型」，真正的调用由服务端发出。
func toAvailableModel(m service.AvailableModel) *modelpb.AvailableModel {
	return &modelpb.AvailableModel{
		Source:          m.Source,
		ModelId:         m.ModelID,
		ProviderId:      m.ProviderID,
		Name:            m.Name,
		ModelType:       m.ModelType,
		ApiModel:        m.APIModel,
		ContextWindow:   m.ContextWindow,
		MaxOutputTokens: m.MaxOutputTokens,
		Capabilities:    toCapabilities(m.Capabilities),
		CreditCost:      m.CreditCost,
		Description:     m.Description,
	}
}

// toSettingInfo 转换系统配置。
//
// secretValue 由调用方显式传入：写入路径填一次明文回显，
// 读取路径一律留空，前端据此把已有机密显示为「已配置」而非星号占位。
func toSettingInfo(s *model.SystemSetting, secretValue string) *modelpb.SettingInfo {
	return &modelpb.SettingInfo{
		Id:          s.ID,
		ConfigGroup: s.ConfigGroup,
		Key:         s.Key,
		Value:       string(s.Value),
		SecretValue: secretValue,
		IsSecret:    s.IsSecret,
		ValueType:   s.ValueType,
		IsPublic:    s.IsPublic,
		Description: optionalString(s.Description),
		CreatedAt:   formatTime(&s.CreatedAt),
		UpdatedAt:   formatTime(&s.UpdatedAt),
	}
}
