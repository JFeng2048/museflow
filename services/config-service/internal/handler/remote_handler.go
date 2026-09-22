package handler

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/museflow/pkg/logger"
	modelpb "github.com/museflow/proto/model"

	"github.com/museflow/config-service/internal/service"
)

// FetchProviderModels 拉取某个渠道的模型目录。
//
// 三种定位方式（平台渠道 / 用户渠道 / 临时凭证）的取舍在 service 层，
// 这里只负责把请求字段原样搬过去：没有字段需要在这里做默认值决策。
func (h *ModelHandler) FetchProviderModels(ctx context.Context, req *modelpb.FetchProviderModelsRequest) (*modelpb.FetchProviderModelsResponse, error) {
	userUUID, err := parseOptionalUUID(req.GetUserUuid())
	if err != nil {
		return nil, mapError(err)
	}

	items, err := h.svc.FetchProviderModels(ctx, service.FetchRemoteInput{
		ProviderID:     req.GetProviderId(),
		UserProviderID: req.GetUserProviderId(),
		UserUUID:       userUUID,
		BaseURL:        req.GetBaseUrl(),
		APIKey:         req.GetApiKey(),
		Protocol:       req.GetProtocol(),
	})
	if err != nil {
		// 只记失败原因与定位标识，不记 body：api_key 明文会随请求字段一起进来。
		logger.WarnContext(ctx, "拉取上游模型列表失败",
			"provider_id", req.GetProviderId(),
			"user_provider_id", req.GetUserProviderId(),
			logger.UserUUID(req.GetUserUuid()),
			logger.Err(err))
		return nil, mapError(err)
	}

	out := make([]*modelpb.RemoteModelInfo, 0, len(items))
	for i := range items {
		out = append(out, toRemoteModelInfo(items[i]))
	}
	return &modelpb.FetchProviderModelsResponse{Items: out, Total: int32(len(out))}, nil
}

// parseOptionalUUID 解析可空的用户标识，空串返回 uuid.Nil 而不报错。
//
// 与 parseUUID 的分工：用户级接口的 user_uuid 必填，缺了就是没带登录态；
// 而本接口也接受不带 user_uuid 的平台渠道探测，此时传空是正常用法。
// 传了但解析不出来仍按参数非法处理——半截标识比完全没传更难排查。
func parseOptionalUUID(raw string) (uuid.UUID, error) {
	if strings.TrimSpace(raw) == "" {
		return uuid.Nil, nil
	}
	return parseUUID(raw)
}
