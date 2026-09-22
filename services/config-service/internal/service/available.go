package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/museflow/config-service/internal/model"
)

// 用户端可用模型的来源标识，对应 proto 的 AvailableModel.source。
//
// 前端按它区分「平台模型」与「我的模型」：平台模型要展示积分消耗、
// 自定义模型则提示「调用费用由你直接付给上游」。
const (
	sourcePlatform = "platform"
	sourceCustom   = "custom"
)

// AvailableModel 用户端可见的单条模型。
//
// 刻意不带任何密钥与 base_url 字段：用户端拿到这份清单只用于「选哪个模型」，
// 真正的 HTTP 请求由服务端按 provider 配置发出。一旦把 base_url 下发到浏览器，
// 就等于把平台的 api_key 暴露给了每一个登录用户。
type AvailableModel struct {
	Source     string
	ModelID    int64
	ProviderID int64

	Name            string
	ModelType       string
	APIModel        string
	ContextWindow   int32
	MaxOutputTokens int32
	Capabilities    model.Capabilities
	CreditCost      int32
	Description     string
}

// ListAvailableModels 合并平台模型与用户自定义模型。
//
// 平台模型在前、自定义模型在后：用户第一眼看到的应该是平台默认提供的那批，
// 自己添加的属于补充。两边都不分页——可用模型量级在几十条以内，
// 分页会让前端的选择器多出一次无意义的请求。
func (s *Service) ListAvailableModels(ctx context.Context, userUUID uuid.UUID, modelType string) ([]AvailableModel, error) {
	if err := validateUserUUID(userUUID); err != nil {
		return nil, err
	}

	// model_type 空表示不限类型；非空时先校验再查库，
	// 否则拼错的类型会被当成「没有可用模型」，调用方无从分辨是配错了还是真没有。
	filterType := strings.TrimSpace(modelType)
	if filterType != "" {
		validated, err := validateModelType(filterType)
		if err != nil {
			return nil, err
		}
		filterType = validated
	}

	platformModels, err := s.models.ListPlatformActive(ctx, filterType)
	if err != nil {
		return nil, err
	}
	customModels, err := s.userModels.ListActiveByUserAndType(ctx, userUUID, filterType)
	if err != nil {
		return nil, err
	}

	out := make([]AvailableModel, 0, len(platformModels)+len(customModels))
	for _, m := range platformModels {
		out = append(out, AvailableModel{
			Source:          sourcePlatform,
			ModelID:         m.ID,
			ProviderID:      m.ProviderID,
			Name:            m.Name,
			ModelType:       m.ModelType,
			APIModel:        m.APIModel,
			ContextWindow:   m.ContextWindow,
			MaxOutputTokens: derefInt32(m.MaxOutputTokens),
			Capabilities:    m.Capabilities,
			CreditCost:      m.CreditCost,
			Description:     derefString(m.Description),
		})
	}
	for _, m := range customModels {
		out = append(out, AvailableModel{
			Source:          sourceCustom,
			ModelID:         m.ID,
			ProviderID:      m.UserProviderID,
			Name:            m.Name,
			ModelType:       m.ModelType,
			APIModel:        m.APIModel,
			ContextWindow:   m.ContextWindow,
			MaxOutputTokens: derefInt32(m.MaxOutputTokens),
			Capabilities:    m.Capabilities,
			// 自定义模型恒为 0：用户直接向上游付费，平台不重复计积分。
			CreditCost:  0,
			Description: derefString(m.Description),
		})
	}
	return out, nil
}

// derefInt32 把可空 int32 还原成值，空值取 0。
//
// 下游是 proto 的 int32，没有「未设置」状态；0 在这里表示「跟随上游默认」，
// 与建模型时 optionalInt32 的约定正好闭环。
func derefInt32(v *int32) int32 {
	if v == nil {
		return 0
	}
	return *v
}

// derefString 把可空字符串还原成值，空值取空串。
func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
