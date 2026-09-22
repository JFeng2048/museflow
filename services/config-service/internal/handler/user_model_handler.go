package handler

import (
	"context"

	"github.com/museflow/pkg/logger"
	modelpb "github.com/museflow/proto/model"

	"github.com/museflow/config-service/internal/service"
)

// ListUserModels 分页查询当前用户的自定义模型。
func (h *ModelHandler) ListUserModels(ctx context.Context, req *modelpb.ListUserModelsRequest) (*modelpb.ListUserModelsResponse, error) {
	userUUID, err := parseUUID(req.GetUserUuid())
	if err != nil {
		return nil, mapError(err)
	}

	items, total, err := h.svc.ListUserModels(ctx, userUUID, req.GetUserProviderId(), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		logger.WarnContext(ctx, "查询自定义模型失败", logger.UserUUID(req.GetUserUuid()), logger.Err(err))
		return nil, mapError(err)
	}

	out := make([]*modelpb.UserModelInfo, 0, len(items))
	for i := range items {
		out = append(out, toUserModelInfo(&items[i]))
	}
	return &modelpb.ListUserModelsResponse{Items: out, Total: total}, nil
}

// CreateUserModel 在用户自己的渠道下新增自定义模型。
//
// user_provider_id 必须属于该用户：归属校验在 service 层完成，
// 协议层只负责把标识原样传下去。
func (h *ModelHandler) CreateUserModel(ctx context.Context, req *modelpb.CreateUserModelRequest) (*modelpb.UserModelInfo, error) {
	userUUID, err := parseUUID(req.GetUserUuid())
	if err != nil {
		return nil, mapError(err)
	}

	m, err := h.svc.CreateUserModel(ctx, userUUID, toCreateUserModelInput(req))
	if err != nil {
		logger.WarnContext(ctx, "创建自定义模型失败", logger.UserUUID(req.GetUserUuid()), "user_provider_id", req.GetUserProviderId(), logger.Err(err))
		return nil, mapError(err)
	}
	logger.InfoContext(ctx, "自定义模型已创建", logger.UserUUID(req.GetUserUuid()), "model_id", m.ID)
	return toUserModelInfo(m), nil
}

// UpdateUserModel 编辑自定义模型。
//
// api_model 可改（用户自己填的调用标识），is_active 是 optional。
func (h *ModelHandler) UpdateUserModel(ctx context.Context, req *modelpb.UpdateUserModelRequest) (*modelpb.UserModelInfo, error) {
	userUUID, err := parseUUID(req.GetUserUuid())
	if err != nil {
		return nil, mapError(err)
	}

	input := toUpdateUserModelInput(req)
	input.IsActive = req.IsActive
	m, err := h.svc.UpdateUserModel(ctx, userUUID, req.GetId(), input)
	if err != nil {
		logger.WarnContext(ctx, "更新自定义模型失败", logger.UserUUID(req.GetUserUuid()), "model_id", req.GetId(), logger.Err(err))
		return nil, mapError(err)
	}
	logger.InfoContext(ctx, "自定义模型已更新", logger.UserUUID(req.GetUserUuid()), "model_id", m.ID)
	return toUserModelInfo(m), nil
}

// DeleteUserModel 删除自定义模型。
func (h *ModelHandler) DeleteUserModel(ctx context.Context, req *modelpb.DeleteUserModelRequest) (*modelpb.DeleteUserModelResponse, error) {
	userUUID, err := parseUUID(req.GetUserUuid())
	if err != nil {
		return nil, mapError(err)
	}

	if err := h.svc.DeleteUserModel(ctx, userUUID, req.GetId()); err != nil {
		logger.WarnContext(ctx, "删除自定义模型失败", logger.UserUUID(req.GetUserUuid()), "model_id", req.GetId(), logger.Err(err))
		return nil, mapError(err)
	}
	logger.InfoContext(ctx, "自定义模型已删除", logger.UserUUID(req.GetUserUuid()), "model_id", req.GetId())
	return &modelpb.DeleteUserModelResponse{Success: true}, nil
}

// toCreateUserModelInput 把创建请求转成 service 层入参。
func toCreateUserModelInput(req *modelpb.CreateUserModelRequest) service.UserModelInput {
	return service.UserModelInput{
		ProviderID:      req.GetUserProviderId(),
		Name:            req.GetName(),
		ModelType:       req.GetModelType(),
		APIModel:        req.GetApiModel(),
		ContextWindow:   req.GetContextWindow(),
		MaxOutputTokens: req.GetMaxOutputTokens(),
		Capabilities:    toModelCapabilities(req.GetCapabilities()),
		Description:     req.GetDescription(),
	}
}

// toUpdateUserModelInput 把更新请求转成 service 层入参。
//
// 不带 ProviderID：自定义模型不提供「换渠道」的入口，service 层会沿用旧值。
func toUpdateUserModelInput(req *modelpb.UpdateUserModelRequest) service.UserModelInput {
	return service.UserModelInput{
		Name:            req.GetName(),
		ModelType:       req.GetModelType(),
		APIModel:        req.GetApiModel(),
		ContextWindow:   req.GetContextWindow(),
		MaxOutputTokens: req.GetMaxOutputTokens(),
		Capabilities:    toModelCapabilities(req.GetCapabilities()),
		Description:     req.GetDescription(),
	}
}
