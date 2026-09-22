package handler

import (
	"context"

	"github.com/museflow/pkg/logger"
	modelpb "github.com/museflow/proto/model"
)

// ListAvailableModels 返回用户端可用的模型清单（平台模型 + 用户自定义模型）。
//
// 平台模型在前、自定义模型在后，由 service 层保证顺序；
// 清单里不含任何密钥与 base_url，用户端只能据此「选模型」。
func (h *ModelHandler) ListAvailableModels(ctx context.Context, req *modelpb.ListAvailableModelsRequest) (*modelpb.ListAvailableModelsResponse, error) {
	userUUID, err := parseUUID(req.GetUserUuid())
	if err != nil {
		return nil, mapError(err)
	}

	items, err := h.svc.ListAvailableModels(ctx, userUUID, req.GetModelType())
	if err != nil {
		logger.WarnContext(ctx, "查询可用模型失败", logger.UserUUID(req.GetUserUuid()), "model_type", req.GetModelType(), logger.Err(err))
		return nil, mapError(err)
	}

	out := make([]*modelpb.AvailableModel, 0, len(items))
	for i := range items {
		out = append(out, toAvailableModel(items[i]))
	}
	return &modelpb.ListAvailableModelsResponse{Items: out}, nil
}
