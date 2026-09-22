package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/museflow/config-service/internal/model"
	"github.com/museflow/config-service/internal/repository"
)

// UserModelInput 用户自定义模型写入参数。
//
// ProviderID 仅创建时生效：自定义模型不提供「换个渠道」的入口——换渠道意味着
// 换了 base_url 与密钥，本质上已经是另一条调用链路，删掉重建比留一个
// 半迁移状态的记录更干净。
// IsActive 与 UserProviderInput.IsActive 同样只在更新时生效。
type UserModelInput struct {
	ProviderID      int64
	Name            string
	ModelType       string
	APIModel        string
	ContextWindow   int32
	MaxOutputTokens int32
	Capabilities    model.Capabilities
	Description     string
	IsActive        *bool
}

// ListUserModels 分页查询某用户的自定义模型。
//
// providerID 传 0 表示不限渠道；负数按参数非法处理，避免被
// `provider_id > 0` 的条件静默忽略后返回全量。
func (s *Service) ListUserModels(ctx context.Context, userUUID uuid.UUID, providerID int64, page, pageSize int) ([]model.UserModel, int64, error) {
	if err := validateUserUUID(userUUID); err != nil {
		return nil, 0, err
	}
	if providerID < 0 {
		return nil, 0, invalidArgumentf("user_provider_id 不能为负数")
	}

	page, pageSize = normalizePage(page, pageSize)
	return s.userModels.List(ctx, repository.UserModelListFilter{
		UserUUID:   userUUID,
		ProviderID: providerID,
		Offset:     pageOffset(page, pageSize),
		Limit:      pageSize,
	})
}

// CreateUserModel 在用户自己的渠道下新增自定义模型。
//
// 归属校验放在字段校验之前：渠道不是该用户的时候，后面每一项校验都是白做，
// 而仓储层给出的错误（自定义模型不存在）与真实原因也对不上。
func (s *Service) CreateUserModel(ctx context.Context, userUUID uuid.UUID, input UserModelInput) (*model.UserModel, error) {
	if err := validateUserUUID(userUUID); err != nil {
		return nil, err
	}
	if err := validateID(input.ProviderID); err != nil {
		return nil, invalidArgumentf("user_provider_id 必须大于 0")
	}
	if _, err := s.userProviders.GetOwnedByID(ctx, input.ProviderID, userUUID); err != nil {
		return nil, err
	}

	m := &model.UserModel{
		UserUUID:        userUUID,
		UserProviderID:  input.ProviderID,
		Name:            "",
		ModelType:       "",
		APIModel:        "",
		ContextWindow:   input.ContextWindow,
		MaxOutputTokens: optionalInt32(input.MaxOutputTokens),
		Capabilities:    input.Capabilities,
		Description:     nil,
		IsActive:        true,
	}
	if err := fillUserModelFields(m, input); err != nil {
		return nil, err
	}
	if err := s.userModels.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// UpdateUserModel 编辑用户自定义模型。
//
// api_model 可以改：自定义模型的调用标识本来就是用户自己填的，
// 与平台模型的 code（被业务代码引用）不是一回事。
func (s *Service) UpdateUserModel(ctx context.Context, userUUID uuid.UUID, id int64, input UserModelInput) (*model.UserModel, error) {
	if err := validateUserUUID(userUUID); err != nil {
		return nil, err
	}
	if err := validateID(id); err != nil {
		return nil, err
	}

	// 先取出真实行：IsActive 未显式传入时要保留原值，
	// 而「整字段覆盖」的更新方式拿不到旧值，只能回读一次。
	existing, err := s.userModels.GetOwnedByID(ctx, id, userUUID)
	if err != nil {
		return nil, err
	}

	m := &model.UserModel{
		UserUUID:        userUUID,
		UserProviderID:  existing.UserProviderID,
		Name:            existing.Name,
		ModelType:       existing.ModelType,
		APIModel:        existing.APIModel,
		ContextWindow:   existing.ContextWindow,
		MaxOutputTokens: existing.MaxOutputTokens,
		Capabilities:    existing.Capabilities,
		Description:     existing.Description,
		IsActive:        existing.IsActive,
	}
	if err := fillUserModelFields(m, input); err != nil {
		return nil, err
	}
	if input.IsActive != nil {
		m.IsActive = *input.IsActive
	}

	updates := map[string]any{
		"name":              m.Name,
		"model_type":        m.ModelType,
		"api_model":         m.APIModel,
		"context_window":    m.ContextWindow,
		"max_output_tokens": m.MaxOutputTokens,
		"capabilities":      m.Capabilities,
		"description":       m.Description,
		"is_active":         m.IsActive,
	}
	if err := s.userModels.Update(ctx, id, userUUID, updates); err != nil {
		return nil, err
	}
	return s.userModels.GetOwnedByID(ctx, id, userUUID)
}

// DeleteUserModel 删除用户自定义模型。
func (s *Service) DeleteUserModel(ctx context.Context, userUUID uuid.UUID, id int64) error {
	if err := validateUserUUID(userUUID); err != nil {
		return err
	}
	if err := validateID(id); err != nil {
		return err
	}
	return s.userModels.Delete(ctx, id, userUUID)
}

// fillUserModelFields 校验并填充用户自定义模型的可变字段。
//
// 创建与更新复用同一套校验规则，避免两边字段越改越不一致；
// 区别只在调用方给的初始值是零值（创建）还是旧行回读值（更新）。
func fillUserModelFields(m *model.UserModel, input UserModelInput) error {
	name, err := validateText("name", input.Name, maxNameLen, true)
	if err != nil {
		return err
	}
	modelType, err := validateModelType(input.ModelType)
	if err != nil {
		return err
	}
	apiModel, err := validateText("api_model", input.APIModel, maxAPIModelLen, true)
	if err != nil {
		return err
	}
	if err := validateNonNegative("context_window", input.ContextWindow); err != nil {
		return err
	}
	if err := validateNonNegative("max_output_tokens", input.MaxOutputTokens); err != nil {
		return err
	}
	description, err := optionalText("description", input.Description, maxDescriptionLen)
	if err != nil {
		return err
	}

	m.Name = name
	m.ModelType = modelType
	m.APIModel = apiModel
	m.Description = description
	return nil
}
