package service

import (
	"context"

	"github.com/museflow/config-service/internal/model"
	"github.com/museflow/config-service/internal/repository"
)

// ModelInput 平台模型写入参数。
//
// Code 仅创建时生效，理由同 ProviderInput.Code：它是业务代码引用的稳定标识。
// MaxOutputTokens 为 0 表示「跟随上游默认」，与数据库列可空对应——
// proto3 的 int32 无法区分「没填」与「填了 0」，而 0 个输出 token 本身没有意义，
// 因此统一按未设置处理。
type ModelInput struct {
	ProviderID      int64
	Code            string
	Name            string
	ModelType       string
	APIModel        string
	ContextWindow   int32
	MaxOutputTokens int32
	Capabilities    model.Capabilities
	CreditCost      int32
	Description     string
	SortOrder       int32
}

// ModelListFilter 平台模型列表查询条件。
type ModelListFilter struct {
	ProviderID int64  // 0 表示不限渠道
	ModelType  string // 空表示不限类型
	Keyword    string
	OnlyActive bool
	Page       int
	PageSize   int
}

// ListModels 分页查询平台模型。
func (s *Service) ListModels(ctx context.Context, filter ModelListFilter) ([]model.Model, int64, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	return s.models.List(ctx, repository.ModelListFilter{
		ProviderID: filter.ProviderID,
		ModelType:  filter.ModelType,
		Keyword:    filter.Keyword,
		OnlyActive: filter.OnlyActive,
		Offset:     pageOffset(page, pageSize),
		Limit:      pageSize,
	})
}

// GetModel 按主键查询平台模型。
func (s *Service) GetModel(ctx context.Context, id int64) (*model.Model, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}
	return s.models.GetByID(ctx, id)
}

// CreateModel 在指定平台渠道下新增模型。
//
// 先校验渠道存在再校验自身字段：渠道不存在时提示「请先创建渠道」比提示
// 「字段校验失败」更接近用户的下一步动作。
func (s *Service) CreateModel(ctx context.Context, input ModelInput) (*model.Model, error) {
	if err := validateID(input.ProviderID); err != nil {
		return nil, invalidArgumentf("provider_id 必须大于 0")
	}
	if _, err := s.providers.GetByID(ctx, input.ProviderID); err != nil {
		return nil, err
	}

	// code 用 maxCodeLen 而不是 maxAPIModelLen：model.code 是 varchar(50)，
	// 用错上限会让 51~100 字符的编码通过校验、到 Postgres 才报 22001，
	// 上层只能看到一个 500，而用户想改的只是一个字段。
	code, err := validateText("code", input.Code, maxCodeLen, true)
	if err != nil {
		return nil, err
	}
	name, err := validateText("name", input.Name, maxNameLen, true)
	if err != nil {
		return nil, err
	}
	modelType, err := validateModelType(input.ModelType)
	if err != nil {
		return nil, err
	}
	apiModel, err := validateText("api_model", input.APIModel, maxAPIModelLen, true)
	if err != nil {
		return nil, err
	}
	if err := validateNonNegative("context_window", input.ContextWindow); err != nil {
		return nil, err
	}
	if err := validateNonNegative("max_output_tokens", input.MaxOutputTokens); err != nil {
		return nil, err
	}
	if err := validateNonNegative("credit_cost", input.CreditCost); err != nil {
		return nil, err
	}
	if err := validateNonNegative("sort_order", input.SortOrder); err != nil {
		return nil, err
	}
	description, err := optionalText("description", input.Description, maxDescriptionLen)
	if err != nil {
		return nil, err
	}

	m := &model.Model{
		ProviderID:      input.ProviderID,
		Code:            code,
		Name:            name,
		ModelType:       modelType,
		APIModel:        apiModel,
		ContextWindow:   input.ContextWindow,
		MaxOutputTokens: optionalInt32(input.MaxOutputTokens),
		Capabilities:    input.Capabilities,
		CreditCost:      input.CreditCost,
		Description:     description,
		IsActive:        true,
		SortOrder:       input.SortOrder,
	}
	if err := s.models.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// UpdateModel 编辑平台模型。
func (s *Service) UpdateModel(ctx context.Context, id int64, input ModelInput) (*model.Model, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}

	name, err := validateText("name", input.Name, maxNameLen, true)
	if err != nil {
		return nil, err
	}
	modelType, err := validateModelType(input.ModelType)
	if err != nil {
		return nil, err
	}
	apiModel, err := validateText("api_model", input.APIModel, maxAPIModelLen, true)
	if err != nil {
		return nil, err
	}
	if err := validateNonNegative("context_window", input.ContextWindow); err != nil {
		return nil, err
	}
	if err := validateNonNegative("max_output_tokens", input.MaxOutputTokens); err != nil {
		return nil, err
	}
	if err := validateNonNegative("credit_cost", input.CreditCost); err != nil {
		return nil, err
	}
	if err := validateNonNegative("sort_order", input.SortOrder); err != nil {
		return nil, err
	}
	description, err := optionalText("description", input.Description, maxDescriptionLen)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{
		"name":              name,
		"model_type":        modelType,
		"api_model":         apiModel,
		"context_window":    input.ContextWindow,
		"max_output_tokens": optionalInt32(input.MaxOutputTokens),
		"capabilities":      input.Capabilities,
		"credit_cost":       input.CreditCost,
		"description":       description,
		"sort_order":        input.SortOrder,
	}
	if err := s.models.Update(ctx, id, updates); err != nil {
		return nil, err
	}
	return s.models.GetByID(ctx, id)
}

// SetModelActive 上架 / 下架平台模型。
//
// 下架只影响这一个模型在用户端的可见性，不影响同渠道的其他模型。
func (s *Service) SetModelActive(ctx context.Context, id int64, active bool) (*model.Model, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}
	if err := s.models.Update(ctx, id, map[string]any{"is_active": active}); err != nil {
		return nil, err
	}
	return s.models.GetByID(ctx, id)
}

// DeleteModel 删除平台模型。
func (s *Service) DeleteModel(ctx context.Context, id int64) error {
	if err := validateID(id); err != nil {
		return err
	}
	return s.models.Delete(ctx, id)
}

// optionalInt32 把 0 转成 NULL，其余返回指针。
func optionalInt32(v int32) *int32 {
	if v == 0 {
		return nil
	}
	return &v
}
