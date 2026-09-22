package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	modeldto "github.com/museflow/api-gateway/internal/dto/model_dto"
	"github.com/museflow/pkg/errcode"
	modelpb "github.com/museflow/proto/model"
)

// ---------- 系统配置（管理端）----------

// ListSettings 系统配置列表
//
//	@Summary		系统配置列表
//	@Description	按分组列出系统配置；机密配置只返回是否已配置，绝不返回明文
//	@Tags			model-模型与系统配置
//	@Produce		json
//	@Security		BearerAuth
//	@Param			config_group	query		string	false	"按分组过滤，空表示全部分组"
//	@Param			only_public		query		bool	false	"是否只返回可下发前端的配置"
//	@Success		200				{object}	errcode.Response{data=modeldto.SettingList}	"查询成功"
//	@Failure		401				{object}	errcode.Response			"未认证"
//	@Failure		403				{object}	errcode.Response			"无权限"
//	@Router			/admin/settings [get]
func (h *ModelHandler) ListSettings(c *gin.Context) {
	resp, err := h.models.Service().ListSettings(c.Request.Context(), &modelpb.ListSettingsRequest{
		ConfigGroup: c.Query("config_group"),
		OnlyPublic:  c.Query("only_public") == "true",
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	// 一律传空串给 toSettingInfo：读取路径不回显机密明文。
	items := make([]modeldto.SettingInfo, 0, len(resp.GetItems()))
	for i := range resp.GetItems() {
		items = append(items, toSettingInfo(resp.GetItems()[i], ""))
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, modeldto.SettingList{Items: items}))
}

// GetSetting 系统配置详情
//
//	@Summary		系统配置详情
//	@Description	按分组与键名读取单条配置；机密配置的 value 为空，只带类型与描述
//	@Tags			model-模型与系统配置
//	@Produce		json
//	@Security		BearerAuth
//	@Param			config_group	path		string	true	"配置分组"
//	@Param			key				path		string	true	"配置键名"
//	@Success		200				{object}	errcode.Response{data=modeldto.SettingInfo}	"查询成功"
//	@Failure		401				{object}	errcode.Response			"未认证"
//	@Failure		403				{object}	errcode.Response			"无权限"
//	@Failure		404				{object}	errcode.Response			"配置不存在"
//	@Router			/admin/settings/{config_group}/{key} [get]
func (h *ModelHandler) GetSetting(c *gin.Context) {
	resp, err := h.models.Service().GetSetting(c.Request.Context(), &modelpb.GetSettingRequest{
		ConfigGroup: c.Param("config_group"),
		Key:         c.Param("key"),
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, toSettingInfo(resp, "")))
}

// UpsertSetting 新增 / 更新系统配置
//
//	@Summary		新增 / 更新系统配置
//	@Description	按 (config_group, key) 存在则更新、不存在则创建；机密值只在本次响应回显一次
//	@Tags			model-模型与系统配置
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			config_group	path		string						true	"配置分组"
//	@Param			key				path		string						true	"配置键名"
//	@Param			body			body		modeldto.UpsertSettingRequest	true	"配置内容"
//	@Success		200				{object}	errcode.Response{data=modeldto.SettingInfo}	"保存成功"
//	@Failure		400				{object}	errcode.Response			"参数错误"
//	@Failure		401				{object}	errcode.Response			"未认证"
//	@Failure		403				{object}	errcode.Response			"无权限"
//	@Router			/admin/settings/{config_group}/{key} [put]
func (h *ModelHandler) UpsertSetting(c *gin.Context) {
	var req modeldto.UpsertSettingRequest
	if !bindModelJSON(c, &req) {
		return
	}

	resp, err := h.models.Service().UpsertSetting(c.Request.Context(), &modelpb.UpsertSettingRequest{
		ConfigGroup: c.Param("config_group"),
		Key:         c.Param("key"),
		Value:       req.Value,
		SecretValue: req.SecretValue,
		IsSecret:    req.IsSecret,
		ValueType:   req.ValueType,
		IsPublic:    req.IsPublic,
		Description: req.Description,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	// 只有写入路径回显机密明文（供前端「已保存成功」提示后立即丢弃）。
	c.JSON(http.StatusOK, errcode.SuccessGin(c, toSettingInfo(resp, resp.GetSecretValue())))
}

// DeleteSetting 删除系统配置
//
//	@Summary		删除系统配置
//	@Description	按分组与键名删除配置
//	@Tags			model-模型与系统配置
//	@Produce		json
//	@Security		BearerAuth
//	@Param			config_group	path		string	true	"配置分组"
//	@Param			key				path		string	true	"配置键名"
//	@Success		200				{object}	errcode.Response{data=modeldto.DeleteResult}	"删除成功"
//	@Failure		401				{object}	errcode.Response			"未认证"
//	@Failure		403				{object}	errcode.Response			"无权限"
//	@Failure		404				{object}	errcode.Response			"配置不存在"
//	@Router			/admin/settings/{config_group}/{key} [delete]
func (h *ModelHandler) DeleteSetting(c *gin.Context) {
	if _, err := h.models.Service().DeleteSetting(c.Request.Context(), &modelpb.DeleteSettingRequest{
		ConfigGroup: c.Param("config_group"),
		Key:         c.Param("key"),
	}); err != nil {
		writeGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, modeldto.DeleteResult{Success: true}))
}

// ---------- 系统配置（用户端只读）----------

// ListPublicSettings 可公开下发的系统配置
//
//	@Summary		公开系统配置
//	@Description	列出标记为公开的系统配置（如积分单价、功能开关），机密配置不会出现在结果中
//	@Tags			model-模型与系统配置
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	errcode.Response{data=modeldto.SettingList}	"查询成功"
//	@Failure		401	{object}	errcode.Response			"未登录"
//	@Router			/user/settings [get]
func (h *ModelHandler) ListPublicSettings(c *gin.Context) {
	if _, ok := requireLogin(c); !ok {
		return
	}

	resp, err := h.models.Service().ListSettings(c.Request.Context(), &modelpb.ListSettingsRequest{
		OnlyPublic: true,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	items := make([]modeldto.SettingInfo, 0, len(resp.GetItems()))
	for i := range resp.GetItems() {
		items = append(items, toSettingInfo(resp.GetItems()[i], ""))
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, modeldto.SettingList{Items: items}))
}
