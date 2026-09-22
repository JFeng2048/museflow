package handler

import (
	"context"

	"github.com/museflow/pkg/logger"
	modelpb "github.com/museflow/proto/model"

	"github.com/museflow/config-service/internal/service"
)

// ListProviders 分页查询平台渠道。
func (h *ModelHandler) ListProviders(ctx context.Context, req *modelpb.ListProvidersRequest) (*modelpb.ListProvidersResponse, error) {
	items, total, err := h.svc.ListProviders(ctx, req.GetKeyword(), req.GetOnlyActive(), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		logger.WarnContext(ctx, "查询平台渠道失败", "keyword", req.GetKeyword(), logger.Err(err))
		return nil, mapError(err)
	}

	out := make([]*modelpb.ProviderInfo, 0, len(items))
	for i := range items {
		out = append(out, toProviderInfo(&items[i]))
	}
	return &modelpb.ListProvidersResponse{Items: out, Total: total}, nil
}

// CreateProvider 新增平台渠道。
func (h *ModelHandler) CreateProvider(ctx context.Context, req *modelpb.CreateProviderRequest) (*modelpb.ProviderInfo, error) {
	provider, err := h.svc.CreateProvider(ctx, toCreateProviderInput(req))
	if err != nil {
		logger.WarnContext(ctx, "创建平台渠道失败", "code", req.GetCode(), logger.Err(err))
		return nil, mapError(err)
	}
	logger.InfoContext(ctx, "平台渠道已创建", "provider_id", provider.ID, "code", provider.Code)
	return toProviderInfo(provider), nil
}

// UpdateProvider 编辑平台渠道（api_key 留空表示不轮换）。
func (h *ModelHandler) UpdateProvider(ctx context.Context, req *modelpb.UpdateProviderRequest) (*modelpb.ProviderInfo, error) {
	provider, err := h.svc.UpdateProvider(ctx, req.GetId(), toUpdateProviderInput(req))
	if err != nil {
		logger.WarnContext(ctx, "更新平台渠道失败", "provider_id", req.GetId(), logger.Err(err))
		return nil, mapError(err)
	}
	logger.InfoContext(ctx, "平台渠道已更新", "provider_id", provider.ID, "code", provider.Code)
	return toProviderInfo(provider), nil
}

// SetProviderActive 启用 / 停用平台渠道（级联影响其下模型的可见性）。
func (h *ModelHandler) SetProviderActive(ctx context.Context, req *modelpb.SetProviderActiveRequest) (*modelpb.ProviderInfo, error) {
	provider, err := h.svc.SetProviderActive(ctx, req.GetId(), req.GetIsActive())
	if err != nil {
		logger.WarnContext(ctx, "切换平台渠道状态失败", "provider_id", req.GetId(), logger.Err(err))
		return nil, mapError(err)
	}
	logger.InfoContext(ctx, "平台渠道状态已切换", "provider_id", provider.ID, "is_active", provider.IsActive)
	return toProviderInfo(provider), nil
}

// DeleteProvider 删除平台渠道，其下仍有模型时返回 FailedPrecondition。
func (h *ModelHandler) DeleteProvider(ctx context.Context, req *modelpb.DeleteProviderRequest) (*modelpb.DeleteProviderResponse, error) {
	if err := h.svc.DeleteProvider(ctx, req.GetId()); err != nil {
		logger.WarnContext(ctx, "删除平台渠道失败", "provider_id", req.GetId(), logger.Err(err))
		return nil, mapError(err)
	}
	logger.InfoContext(ctx, "平台渠道已删除", "provider_id", req.GetId())
	return &modelpb.DeleteProviderResponse{Success: true}, nil
}

// toCreateProviderInput 把创建请求转成 service 层入参。
func toCreateProviderInput(req *modelpb.CreateProviderRequest) service.ProviderInput {
	return service.ProviderInput{
		Code:         req.GetCode(),
		Name:         req.GetName(),
		Protocol:     req.GetProtocol(),
		BaseURL:      req.GetBaseUrl(),
		APIKey:       req.GetApiKey(),
		Organization: req.GetOrganization(),
		Extra:        req.GetExtra(),
		SortOrder:    req.GetSortOrder(),
	}
}

// toUpdateProviderInput 把更新请求转成 service 层入参。
//
// 不带 code：它是被业务代码引用的稳定标识，service 层只在创建时读取。
// 协议层也刻意不暴露该字段，后台表单便无从提交。
func toUpdateProviderInput(req *modelpb.UpdateProviderRequest) service.ProviderInput {
	return service.ProviderInput{
		Name:         req.GetName(),
		Protocol:     req.GetProtocol(),
		BaseURL:      req.GetBaseUrl(),
		APIKey:       req.GetApiKey(),
		Organization: req.GetOrganization(),
		Extra:        req.GetExtra(),
		SortOrder:    req.GetSortOrder(),
	}
}
