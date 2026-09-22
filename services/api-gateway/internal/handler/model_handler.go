package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/museflow/api-gateway/internal/client"
	modeldto "github.com/museflow/api-gateway/internal/dto/model_dto"
	"github.com/museflow/api-gateway/internal/middleware"
	"github.com/museflow/pkg/errcode"
	modelpb "github.com/museflow/proto/model"
)

// ModelHandler 模型目录与系统配置处理器。
//
// 权限不在本层判断：管理端路由由 middleware.RequirePermission(userClient,
// "system:admin") 统一校验，用户端路由只要求登录。
type ModelHandler struct {
	models *client.ModelClient
}

// NewModelHandler 构造模型与系统配置处理器。
func NewModelHandler(models *client.ModelClient) *ModelHandler {
	return &ModelHandler{models: models}
}

// requireLogin 取当前登录用户，未登录时已写入响应并返回 false。
//
// 用户级接口（自定义渠道 / 模型 / 可用模型）全部以 token 中的 uuid 为准，
// 不接受请求体或查询参数里的 user_uuid：否则改一个参数就能操作别人的数据。
func requireLogin(c *gin.Context) (string, bool) {
	uid := middleware.CurrentUserUUID(c)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, errcode.ErrorGin(c, errcode.CodeUnauthorized))
		return "", false
	}
	return uid, true
}

// toCapabilities 把 proto 能力开关转成响应结构。
//
// nil 取零值：proto3 消息字段不带「未设置」状态，config-service 侧由
// toCapabilities 保证永不为 nil，这里仍兜一层，避免后续改返回指针时 panic。
func toCapabilities(c *modelpb.ModelCapabilities) modeldto.Capabilities {
	if c == nil {
		return modeldto.Capabilities{}
	}
	return modeldto.Capabilities{
		Stream:   c.GetStream(),
		ToolCall: c.GetToolCall(),
		JSONMode: c.GetJsonMode(),
		Vision:   c.GetVision(),
	}
}

// toCapabilitiesRequest 把请求体里的能力开关转成 proto 消息。
//
// nil 传 nil：config-service 的 toModelCapabilities 会把 nil 解释为「全部关闭」，
// 与「请求体没带该字段」的意图一致。
func toCapabilitiesRequest(c *modeldto.CapabilitiesRequest) *modelpb.ModelCapabilities {
	if c == nil {
		return nil
	}
	return &modelpb.ModelCapabilities{
		Stream:   c.Stream,
		ToolCall: c.ToolCall,
		JsonMode: c.JSONMode,
		Vision:   c.Vision,
	}
}

// toProviderInfo 转换平台渠道响应。
func toProviderInfo(p *modelpb.ProviderInfo) modeldto.ProviderInfo {
	if p == nil {
		return modeldto.ProviderInfo{}
	}
	return modeldto.ProviderInfo{
		ID:              p.GetId(),
		Code:            p.GetCode(),
		Name:            p.GetName(),
		Protocol:        p.GetProtocol(),
		BaseURL:         p.GetBaseUrl(),
		APIKeyHint:      p.GetApiKeyHint(),
		APIKeyUpdatedAt: p.GetApiKeyUpdatedAt(),
		Organization:    p.GetOrganization(),
		Extra:           p.GetExtra(),
		IsActive:        p.GetIsActive(),
		SortOrder:       p.GetSortOrder(),
		CreatedAt:       p.GetCreatedAt(),
		UpdatedAt:       p.GetUpdatedAt(),
	}
}

// toModelInfo 转换平台模型响应。
func toModelInfo(m *modelpb.ModelInfo) modeldto.ModelInfo {
	if m == nil {
		return modeldto.ModelInfo{}
	}
	return modeldto.ModelInfo{
		ID:              m.GetId(),
		ProviderID:      m.GetProviderId(),
		Code:            m.GetCode(),
		Name:            m.GetName(),
		ModelType:       m.GetModelType(),
		APIModel:        m.GetApiModel(),
		ContextWindow:   m.GetContextWindow(),
		MaxOutputTokens: m.GetMaxOutputTokens(),
		Capabilities:    toCapabilities(m.GetCapabilities()),
		CreditCost:      m.GetCreditCost(),
		Description:     m.GetDescription(),
		IsActive:        m.GetIsActive(),
		SortOrder:       m.GetSortOrder(),
		CreatedAt:       m.GetCreatedAt(),
		UpdatedAt:       m.GetUpdatedAt(),
	}
}

// toUserProviderInfo 转换用户自定义渠道响应。
func toUserProviderInfo(p *modelpb.UserProviderInfo) modeldto.UserProviderInfo {
	if p == nil {
		return modeldto.UserProviderInfo{}
	}
	return modeldto.UserProviderInfo{
		ID:              p.GetId(),
		UserUUID:        p.GetUserUuid(),
		Name:            p.GetName(),
		Protocol:        p.GetProtocol(),
		BaseURL:         p.GetBaseUrl(),
		APIKeyHint:      p.GetApiKeyHint(),
		APIKeyUpdatedAt: p.GetApiKeyUpdatedAt(),
		Organization:    p.GetOrganization(),
		Extra:           p.GetExtra(),
		IsActive:        p.GetIsActive(),
		CreatedAt:       p.GetCreatedAt(),
		UpdatedAt:       p.GetUpdatedAt(),
	}
}

// toUserModelInfo 转换用户自定义模型响应。
func toUserModelInfo(m *modelpb.UserModelInfo) modeldto.UserModelInfo {
	if m == nil {
		return modeldto.UserModelInfo{}
	}
	return modeldto.UserModelInfo{
		ID:              m.GetId(),
		UserUUID:        m.GetUserUuid(),
		UserProviderID:  m.GetUserProviderId(),
		Name:            m.GetName(),
		ModelType:       m.GetModelType(),
		APIModel:        m.GetApiModel(),
		ContextWindow:   m.GetContextWindow(),
		MaxOutputTokens: m.GetMaxOutputTokens(),
		Capabilities:    toCapabilities(m.GetCapabilities()),
		Description:     m.GetDescription(),
		IsActive:        m.GetIsActive(),
		CreatedAt:       m.GetCreatedAt(),
		UpdatedAt:       m.GetUpdatedAt(),
	}
}

// toAvailableModel 转换用户端可用模型。
func toAvailableModel(m *modelpb.AvailableModel) modeldto.AvailableModel {
	if m == nil {
		return modeldto.AvailableModel{}
	}
	return modeldto.AvailableModel{
		Source:          m.GetSource(),
		ModelID:         m.GetModelId(),
		ProviderID:      m.GetProviderId(),
		Name:            m.GetName(),
		ModelType:       m.GetModelType(),
		APIModel:        m.GetApiModel(),
		ContextWindow:   m.GetContextWindow(),
		MaxOutputTokens: m.GetMaxOutputTokens(),
		Capabilities:    toCapabilities(m.GetCapabilities()),
		CreditCost:      m.GetCreditCost(),
		Description:     m.GetDescription(),
	}
}

// toSettingInfo 转换系统配置响应。
//
// secretEcho 只在写入响应中传一次明文，读取路径一律传空串——
// 这样可以保证任何查询接口都不可能把机密配置的明文带出去。
func toSettingInfo(s *modelpb.SettingInfo, secretEcho string) modeldto.SettingInfo {
	if s == nil {
		return modeldto.SettingInfo{}
	}
	return modeldto.SettingInfo{
		ID:          s.GetId(),
		ConfigGroup: s.GetConfigGroup(),
		Key:         s.GetKey(),
		Value:       s.GetValue(),
		SecretValue: secretEcho,
		IsSecret:    s.GetIsSecret(),
		ValueType:   s.GetValueType(),
		IsPublic:    s.GetIsPublic(),
		Description: s.GetDescription(),
		CreatedAt:   s.GetCreatedAt(),
		UpdatedAt:   s.GetUpdatedAt(),
	}
}

// bindModelJSON 绑定并校验请求体，失败时已写入响应并返回 false。
//
// 模型域四个文件有十余处请求体绑定，收敛成一个助手避免重复「绑定失败 → 400 →
// return」这段样板；错误文案沿用 admin_handler 的
// errcode.Fail(CodeParamInvalid, err.Error())，把校验细节回给前端。
func bindModelJSON(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		c.JSON(http.StatusBadRequest, errcode.Fail(errcode.CodeParamInvalid, err.Error()))
		return false
	}
	return true
}

// parseModelID 解析路径中的 int64 主键，失败时已写入响应并返回 false。
//
// 与 admin_handler 的 parseID（int32）分开：模型目录的主键是 int8/bigint，
// 共用一套 int32 解析会在 id 超过 21 亿时静默截断，因此这里按 int64 自己解析。
func parseModelID(c *gin.Context, param string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param(param)), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, errcode.ErrorGin(c, errcode.CodeParamInvalid))
		return 0, false
	}
	return id, true
}
