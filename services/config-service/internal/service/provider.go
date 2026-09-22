package service

import (
	"context"

	"github.com/museflow/config-service/internal/model"
	"github.com/museflow/config-service/internal/repository"
)

// ProviderInput 平台渠道写入参数。
//
// Code 仅创建时生效：它是被业务代码引用的稳定标识（如 "openai"），
// 改名会让所有写死该编码的调用点失效，因此不提供修改入口。
// APIKey 在更新语义里是「非空即轮换」：本服务从不回读密钥，
// 无法从响应里判断旧值是什么，留空就只能理解为「不改」。
type ProviderInput struct {
	Code         string
	Name         string
	Protocol     string
	BaseURL      string
	APIKey       string
	Organization string
	Extra        map[string]string
	SortOrder    int32
}

// ListProviders 分页查询平台渠道。
func (s *Service) ListProviders(ctx context.Context, keyword string, onlyActive bool, page, pageSize int) ([]model.ModelProvider, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	return s.providers.List(ctx, repository.ProviderListFilter{
		Keyword:    keyword,
		OnlyActive: onlyActive,
		Offset:     pageOffset(page, pageSize),
		Limit:      pageSize,
	})
}

// GetProvider 按主键查询平台渠道。
func (s *Service) GetProvider(ctx context.Context, id int64) (*model.ModelProvider, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}
	return s.providers.GetByID(ctx, id)
}

// CreateProvider 新增平台渠道。
//
// 新建渠道默认启用：后台填完一串 base_url 加密钥后立刻生效是常见预期，
// 需要灰度时再显式调 SetProviderActive 关掉。
func (s *Service) CreateProvider(ctx context.Context, input ProviderInput) (*model.ModelProvider, error) {
	code, err := validateText("code", input.Code, maxCodeLen, true)
	if err != nil {
		return nil, err
	}
	name, err := validateText("name", input.Name, maxNameLen, true)
	if err != nil {
		return nil, err
	}
	protocol, err := validateProtocol(input.Protocol)
	if err != nil {
		return nil, err
	}
	baseURL, err := validateBaseURL(input.BaseURL)
	if err != nil {
		return nil, err
	}
	if err := validateNonNegative("sort_order", input.SortOrder); err != nil {
		return nil, err
	}
	organization, err := optionalText("organization", input.Organization, maxOrganizationLen)
	if err != nil {
		return nil, err
	}
	extra, err := normalizeExtra(input.Extra)
	if err != nil {
		return nil, err
	}
	sealed, err := s.sealAPIKey(input.APIKey)
	if err != nil {
		return nil, err
	}

	provider := &model.ModelProvider{
		Code:             code,
		Name:             name,
		Protocol:         protocol,
		BaseURL:          baseURL,
		APIKeyCiphertext: sealed.Ciphertext,
		APIKeyHint:       sealed.Hint,
		APIKeyUpdatedAt:  sealed.UpdatedAt,
		Organization:     organization,
		Extra:            model.StringMap(extra),
		IsActive:         true,
		SortOrder:        input.SortOrder,
	}
	if err := s.providers.Create(ctx, provider); err != nil {
		return nil, err
	}
	return provider, nil
}

// UpdateProvider 编辑平台渠道。
//
// organization 与 extra 是覆盖语义：传空即清空。表单提交的本来就是整份配置，
// 「空值」与「不改」无法在字符串字段上区分，选覆盖至少行为可预期。
func (s *Service) UpdateProvider(ctx context.Context, id int64, input ProviderInput) (*model.ModelProvider, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}

	name, err := validateText("name", input.Name, maxNameLen, true)
	if err != nil {
		return nil, err
	}
	protocol, err := validateProtocol(input.Protocol)
	if err != nil {
		return nil, err
	}
	baseURL, err := validateBaseURL(input.BaseURL)
	if err != nil {
		return nil, err
	}
	if err := validateNonNegative("sort_order", input.SortOrder); err != nil {
		return nil, err
	}
	organization, err := optionalText("organization", input.Organization, maxOrganizationLen)
	if err != nil {
		return nil, err
	}
	extra, err := normalizeExtra(input.Extra)
	if err != nil {
		return nil, err
	}
	sealed, err := s.sealAPIKey(input.APIKey)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{
		"name":         name,
		"protocol":     protocol,
		"base_url":     baseURL,
		"organization": organization,
		"extra":        model.StringMap(extra),
		"sort_order":   input.SortOrder,
	}
	if sealed.Ciphertext != "" {
		updates["api_key"] = sealed.Ciphertext
		updates["api_key_hint"] = sealed.Hint
		updates["api_key_updated_at"] = sealed.UpdatedAt
	}

	if err := s.providers.Update(ctx, id, updates); err != nil {
		return nil, err
	}
	return s.providers.GetByID(ctx, id)
}

// SetProviderActive 启用 / 停用平台渠道。
//
// 停用是级联可见的：其下模型在 ListAvailableModels 里会一并消失，
// 因此这是「某家厂商整体下线」的开关，不需要逐个模型操作。
func (s *Service) SetProviderActive(ctx context.Context, id int64, active bool) (*model.ModelProvider, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}
	if err := s.providers.Update(ctx, id, map[string]any{"is_active": active}); err != nil {
		return nil, err
	}
	return s.providers.GetByID(ctx, id)
}

// DeleteProvider 删除平台渠道。渠道下仍有模型时由仓储拒绝。
func (s *Service) DeleteProvider(ctx context.Context, id int64) error {
	if err := validateID(id); err != nil {
		return err
	}
	return s.providers.Delete(ctx, id)
}
