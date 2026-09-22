package handler

import (
	"context"

	"github.com/museflow/pkg/logger"
	modelpb "github.com/museflow/proto/model"

	"github.com/museflow/config-service/internal/service"
)

// ListModels 分页查询平台模型。
func (h *ModelHandler) ListModels(ctx context.Context, req *modelpb.ListModelsRequest) (*modelpb.ListModelsResponse, error) {
	items, total, err := h.svc.ListModels(ctx, service.ModelListFilter{
		ProviderID: req.GetProviderId(),
		ModelType:  req.GetModelType(),
		Keyword:    req.GetKeyword(),
		OnlyActive: req.GetOnlyActive(),
		Page:       int(req.GetPage()),
		PageSize:   int(req.GetPageSize()),
	})
	if err != nil {
		logger.WarnContext(ctx, "查询平台模型失败", "provider_id", req.GetProviderId(), logger.Err(err))
		return nil, mapError(err)
	}

	out := make([]*modelpb.ModelInfo, 0, len(items))
	for i := range items {
		out = append(out, toModelInfo(&items[i]))
	}
	return &modelpb.ListModelsResponse{Items: out, Total: total}, nil
}

// CreateModel 在指定平台渠道下新增模型。
func (h *ModelHandler) CreateModel(ctx context.Context, req *modelpb.CreateModelRequest) (*modelpb.ModelInfo, error) {
	m, err := h.svc.CreateModel(ctx, toCreateModelInput(req))
	if err != nil {
		logger.WarnContext(ctx, "创建平台模型失败", "provider_id", req.GetProviderId(), "code", req.GetCode(), logger.Err(err))
		return nil, mapError(err)
	}
	logger.InfoContext(ctx, "平台模型已创建", "model_id", m.ID, "code", m.Code)
	return toModelInfo(m), nil
}

// UpdateModel 编辑平台模型（code 不可改）。
func (h *ModelHandler) UpdateModel(ctx context.Context, req *modelpb.UpdateModelRequest) (*modelpb.ModelInfo, error) {
	m, err := h.svc.UpdateModel(ctx, req.GetId(), toUpdateModelInput(req))
	if err != nil {
		logger.WarnContext(ctx, "更新平台模型失败", "model_id", req.GetId(), logger.Err(err))
		return nil, mapError(err)
	}
	logger.InfoContext(ctx, "平台模型已更新", "model_id", m.ID, "code", m.Code)
	return toModelInfo(m), nil
}

// SetModelActive 上架 / 下架平台模型。
//
// 只影响该模型在用户端的可见性，同渠道其他模型不受影响。
func (h *ModelHandler) SetModelActive(ctx context.Context, req *modelpb.SetModelActiveRequest) (*modelpb.ModelInfo, error) {
	m, err := h.svc.SetModelActive(ctx, req.GetId(), req.GetIsActive())
	if err != nil {
		logger.WarnContext(ctx, "切换平台模型状态失败", "model_id", req.GetId(), logger.Err(err))
		return nil, mapError(err)
	}
	logger.InfoContext(ctx, "平台模型状态已切换", "model_id", m.ID, "is_active", m.IsActive)
	return toModelInfo(m), nil
}

// DeleteModel 删除平台模型。
func (h *ModelHandler) DeleteModel(ctx context.Context, req *modelpb.DeleteModelRequest) (*modelpb.DeleteModelResponse, error) {
	if err := h.svc.DeleteModel(ctx, req.GetId()); err != nil {
		logger.WarnContext(ctx, "删除平台模型失败", "model_id", req.GetId(), logger.Err(err))
		return nil, mapError(err)
	}
	logger.InfoContext(ctx, "平台模型已删除", "model_id", req.GetId())
	return &modelpb.DeleteModelResponse{Success: true}, nil
}

// toCreateModelInput 把创建请求转成 service 层入参。
func toCreateModelInput(req *modelpb.CreateModelRequest) service.ModelInput {
	return service.ModelInput{
		ProviderID:      req.GetProviderId(),
		Code:            req.GetCode(),
		Name:            req.GetName(),
		ModelType:       req.GetModelType(),
		APIModel:        req.GetApiModel(),
		ContextWindow:   req.GetContextWindow(),
		MaxOutputTokens: req.GetMaxOutputTokens(),
		Capabilities:    toModelCapabilities(req.GetCapabilities()),
		CreditCost:      req.GetCreditCost(),
		Description:     req.GetDescription(),
		SortOrder:       req.GetSortOrder(),
	}
}

// toUpdateModelInput 把更新请求转成 service 层入参。
//
// 不带 ProviderID 与 Code：模型不提供「换个渠道」的入口（那是另一条调用链路），
// code 是被业务代码引用的稳定标识，两者都不该在编辑时变更。
func toUpdateModelInput(req *modelpb.UpdateModelRequest) service.ModelInput {
	return service.ModelInput{
		Name:            req.GetName(),
		ModelType:       req.GetModelType(),
		APIModel:        req.GetApiModel(),
		ContextWindow:   req.GetContextWindow(),
		MaxOutputTokens: req.GetMaxOutputTokens(),
		Capabilities:    toModelCapabilities(req.GetCapabilities()),
		CreditCost:      req.GetCreditCost(),
		Description:     req.GetDescription(),
		SortOrder:       req.GetSortOrder(),
	}
}
