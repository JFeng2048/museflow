package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/museflow/config-service/internal/model"
)

// UserProviderInput 用户自定义渠道写入参数。
//
// IsActive 只在更新时生效，且是「填了才改」：proto 里它是 optional，
// 为的就是避免一次忘带该字段的调用把所有渠道静默停用。创建时渠道固定启用。
type UserProviderInput struct {
	Name         string
	Protocol     string
	BaseURL      string
	APIKey       string
	Organization string
	Extra        map[string]string
	IsActive     *bool
}

// ListUserProviders 列出某用户的全部自定义渠道。
//
// 不分页：自定义渠道由用户逐个手工添加，量级在个位到几十之间，
// 一次取回还能省掉前端的「加载更多」。
func (s *Service) ListUserProviders(ctx context.Context, userUUID uuid.UUID) ([]model.UserModelProvider, error) {
	if err := validateUserUUID(userUUID); err != nil {
		return nil, err
	}
	return s.userProviders.ListByUser(ctx, userUUID)
}

// CreateUserProvider 新增用户自定义渠道。
//
// 与平台渠道的差别只有两点：没有 code（不被业务代码引用）、没有 sort_order
// （用户自己的顺序用 id 的自然增长就够了）。
func (s *Service) CreateUserProvider(ctx context.Context, userUUID uuid.UUID, input UserProviderInput) (*model.UserModelProvider, error) {
	if err := validateUserUUID(userUUID); err != nil {
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

	provider := &model.UserModelProvider{
		UserUUID:         userUUID,
		Name:             name,
		Protocol:         protocol,
		BaseURL:          baseURL,
		APIKeyCiphertext: sealed.Ciphertext,
		APIKeyHint:       sealed.Hint,
		APIKeyUpdatedAt:  sealed.UpdatedAt,
		Organization:     organization,
		Extra:            model.StringMap(extra),
		IsActive:         true,
	}
	if err := s.userProviders.Create(ctx, provider); err != nil {
		return nil, err
	}
	return provider, nil
}

// UpdateUserProvider 编辑用户自定义渠道。
//
// 归属校验落在仓储层：所有写操作都带 user_uuid 条件，即便上层漏传也碰不到别人的行。
// api_key 语义与平台渠道一致——留空表示不修改（本服务不回读密钥，无法比对旧值）。
func (s *Service) UpdateUserProvider(ctx context.Context, userUUID uuid.UUID, id int64, input UserProviderInput) (*model.UserModelProvider, error) {
	if err := validateUserUUID(userUUID); err != nil {
		return nil, err
	}
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
	}
	if input.IsActive != nil {
		updates["is_active"] = *input.IsActive
	}
	if sealed.Ciphertext != "" {
		updates["api_key"] = sealed.Ciphertext
		updates["api_key_hint"] = sealed.Hint
		updates["api_key_updated_at"] = sealed.UpdatedAt
	}

	if err := s.userProviders.Update(ctx, id, userUUID, updates); err != nil {
		return nil, err
	}
	return s.userProviders.GetOwnedByID(ctx, id, userUUID)
}

// DeleteUserProvider 删除用户自定义渠道。渠道下仍有模型时由仓储拒绝。
func (s *Service) DeleteUserProvider(ctx context.Context, userUUID uuid.UUID, id int64) error {
	if err := validateUserUUID(userUUID); err != nil {
		return err
	}
	if err := validateID(id); err != nil {
		return err
	}
	return s.userProviders.Delete(ctx, id, userUUID)
}
