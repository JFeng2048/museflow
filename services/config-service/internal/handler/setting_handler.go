package handler

import (
	"context"

	"github.com/museflow/pkg/logger"
	modelpb "github.com/museflow/proto/model"

	"github.com/museflow/config-service/internal/service"
)

// ListSettings 查询系统配置。
//
// config_group 空表示全部分组；机密配置只说明「这是机密」，不带明文。
func (h *ModelHandler) ListSettings(ctx context.Context, req *modelpb.ListSettingsRequest) (*modelpb.ListSettingsResponse, error) {
	items, err := h.svc.ListSettings(ctx, req.GetConfigGroup(), req.GetOnlyPublic())
	if err != nil {
		logger.WarnContext(ctx, "查询系统配置失败", "config_group", req.GetConfigGroup(), logger.Err(err))
		return nil, mapError(err)
	}

	// secret_value 一律传空串：读取路径不透出任何机密明文。
	out := make([]*modelpb.SettingInfo, 0, len(items))
	for i := range items {
		out = append(out, toSettingInfo(&items[i], ""))
	}
	return &modelpb.ListSettingsResponse{Items: out}, nil
}

// GetSetting 按 (config_group, key) 查询系统配置。
func (h *ModelHandler) GetSetting(ctx context.Context, req *modelpb.GetSettingRequest) (*modelpb.SettingInfo, error) {
	setting, err := h.svc.GetSetting(ctx, req.GetConfigGroup(), req.GetKey())
	if err != nil {
		logger.WarnContext(ctx, "查询系统配置失败", "config_group", req.GetConfigGroup(), "key", req.GetKey(), logger.Err(err))
		return nil, mapError(err)
	}
	// 读取路径 secret_value 留空，前端据此把已有机密显示为「已配置」。
	return toSettingInfo(setting, ""), nil
}

// UpsertSetting 写入系统配置，存在则更新、不存在则创建。
func (h *ModelHandler) UpsertSetting(ctx context.Context, req *modelpb.UpsertSettingRequest) (*modelpb.SettingInfo, error) {
	result, err := h.svc.UpsertSetting(ctx, service.SettingInput{
		ConfigGroup: req.GetConfigGroup(),
		Key:         req.GetKey(),
		Value:       req.GetValue(),
		SecretValue: req.GetSecretValue(),
		IsSecret:    req.GetIsSecret(),
		ValueType:   req.GetValueType(),
		IsPublic:    req.GetIsPublic(),
		Description: req.GetDescription(),
	})
	if err != nil {
		logger.WarnContext(ctx, "写入系统配置失败", "config_group", req.GetConfigGroup(), "key", req.GetKey(), logger.Err(err))
		return nil, mapError(err)
	}
	logger.InfoContext(ctx, "系统配置已写入", "config_group", result.Setting.ConfigGroup, "key", result.Setting.Key, "is_secret", result.Setting.IsSecret)
	// secret_value 只在写入响应里回显一次，读取路径恒为空。
	return toSettingInfo(result.Setting, result.SecretEcho), nil
}

// DeleteSetting 按 (config_group, key) 删除系统配置。
func (h *ModelHandler) DeleteSetting(ctx context.Context, req *modelpb.DeleteSettingRequest) (*modelpb.DeleteSettingResponse, error) {
	if err := h.svc.DeleteSetting(ctx, req.GetConfigGroup(), req.GetKey()); err != nil {
		logger.WarnContext(ctx, "删除系统配置失败", "config_group", req.GetConfigGroup(), "key", req.GetKey(), logger.Err(err))
		return nil, mapError(err)
	}
	logger.InfoContext(ctx, "系统配置已删除", "config_group", req.GetConfigGroup(), "key", req.GetKey())
	return &modelpb.DeleteSettingResponse{Success: true}, nil
}
