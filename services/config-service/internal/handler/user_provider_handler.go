package handler

import (
	"context"

	"github.com/google/uuid"
	"github.com/museflow/pkg/logger"
	modelpb "github.com/museflow/proto/model"

	"github.com/museflow/config-service/internal/service"
)

// ListUserProviders 列出当前用户的自定义渠道。
func (h *ModelHandler) ListUserProviders(ctx context.Context, req *modelpb.ListUserProvidersRequest) (*modelpb.ListUserProvidersResponse, error) {
	userUUID, err := parseUUID(req.GetUserUuid())
	if err != nil {
		return nil, mapError(err)
	}

	items, err := h.svc.ListUserProviders(ctx, userUUID)
	if err != nil {
		logger.WarnContext(ctx, "查询自定义渠道失败", logger.UserUUID(req.GetUserUuid()), logger.Err(err))
		return nil, mapError(err)
	}

	out := make([]*modelpb.UserProviderInfo, 0, len(items))
	for i := range items {
		out = append(out, toUserProviderInfo(&items[i]))
	}
	// total 如实回填：service 层不分页，网关拿它渲染总数即可。
	return &modelpb.ListUserProvidersResponse{Items: out, Total: int64(len(out))}, nil
}

// CreateUserProvider 新增自定义渠道。
func (h *ModelHandler) CreateUserProvider(ctx context.Context, req *modelpb.CreateUserProviderRequest) (*modelpb.UserProviderInfo, error) {
	userUUID, err := parseUUID(req.GetUserUuid())
	if err != nil {
		return nil, mapError(err)
	}

	provider, err := h.svc.CreateUserProvider(ctx, userUUID, toCreateUserProviderInput(req))
	if err != nil {
		logger.WarnContext(ctx, "创建自定义渠道失败", logger.UserUUID(req.GetUserUuid()), logger.Err(err))
		return nil, mapError(err)
	}
	logger.InfoContext(ctx, "自定义渠道已创建", logger.UserUUID(req.GetUserUuid()), "provider_id", provider.ID)
	return toUserProviderInfo(provider), nil
}

// UpdateUserProvider 编辑自定义渠道。
//
// is_active 是 optional：只有显式传入才会改动启用状态，
// 否则 proto3 的默认值 false 会被误读成「停用」。
func (h *ModelHandler) UpdateUserProvider(ctx context.Context, req *modelpb.UpdateUserProviderRequest) (*modelpb.UserProviderInfo, error) {
	userUUID, err := parseUUID(req.GetUserUuid())
	if err != nil {
		return nil, mapError(err)
	}

	input := toUpdateUserProviderInput(req)
	input.IsActive = req.IsActive
	provider, err := h.svc.UpdateUserProvider(ctx, userUUID, req.GetId(), input)
	if err != nil {
		logger.WarnContext(ctx, "更新自定义渠道失败", logger.UserUUID(req.GetUserUuid()), "provider_id", req.GetId(), logger.Err(err))
		return nil, mapError(err)
	}
	logger.InfoContext(ctx, "自定义渠道已更新", logger.UserUUID(req.GetUserUuid()), "provider_id", provider.ID)
	return toUserProviderInfo(provider), nil
}

// DeleteUserProvider 删除自定义渠道，其下仍有模型时返回 FailedPrecondition。
func (h *ModelHandler) DeleteUserProvider(ctx context.Context, req *modelpb.DeleteUserProviderRequest) (*modelpb.DeleteUserProviderResponse, error) {
	userUUID, err := parseUUID(req.GetUserUuid())
	if err != nil {
		return nil, mapError(err)
	}

	if err := h.svc.DeleteUserProvider(ctx, userUUID, req.GetId()); err != nil {
		logger.WarnContext(ctx, "删除自定义渠道失败", logger.UserUUID(req.GetUserUuid()), "provider_id", req.GetId(), logger.Err(err))
		return nil, mapError(err)
	}
	logger.InfoContext(ctx, "自定义渠道已删除", logger.UserUUID(req.GetUserUuid()), "provider_id", req.GetId())
	return &modelpb.DeleteUserProviderResponse{Success: true}, nil
}

// toCreateUserProviderInput 把创建请求转成 service 层入参。
//
// IsActive 不在这里赋值：新建渠道固定启用，由 service 层决定。
func toCreateUserProviderInput(req *modelpb.CreateUserProviderRequest) service.UserProviderInput {
	return service.UserProviderInput{
		Name:         req.GetName(),
		Protocol:     req.GetProtocol(),
		BaseURL:      req.GetBaseUrl(),
		APIKey:       req.GetApiKey(),
		Organization: req.GetOrganization(),
		Extra:        req.GetExtra(),
	}
}

// toUpdateUserProviderInput 把更新请求转成 service 层入参。
//
// api_key 留空表示不轮换：协议层原样透传空串，由 service 层判断是否写 key 列。
func toUpdateUserProviderInput(req *modelpb.UpdateUserProviderRequest) service.UserProviderInput {
	return service.UserProviderInput{
		Name:         req.GetName(),
		Protocol:     req.GetProtocol(),
		BaseURL:      req.GetBaseUrl(),
		APIKey:       req.GetApiKey(),
		Organization: req.GetOrganization(),
		Extra:        req.GetExtra(),
	}
}

// parseUUID 解析用户标识，非法时报 InvalidArgument。
//
// 用户级表全部按 user_uuid 隔离，这里放行一个空 UUID 会让数据落到
// uuid.Nil 名下，事后谁也查不回来。
func parseUUID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, ErrInvalidUUID
	}
	return id, nil
}
