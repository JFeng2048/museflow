package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	modeldto "github.com/museflow/api-gateway/internal/dto/model_dto"
	"github.com/museflow/pkg/errcode"
	modelpb "github.com/museflow/proto/model"
)

// ---------- 平台渠道（管理端）----------

// ListProviders 平台渠道列表
//
//	@Summary		平台渠道列表
//	@Description	分页查询平台模型渠道，支持关键字（编码 / 名称）与状态筛选
//	@Tags			model-模型与系统配置
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int		false	"页码，默认 1"
//	@Param			page_size	query		int		false	"每页条数，默认 20，最大 100"
//	@Param			keyword		query		string	false	"按编码或名称模糊搜索"
//	@Param			only_active	query		bool	false	"是否只返回启用中的渠道"
//	@Success		200			{object}	errcode.Response{data=modeldto.ProviderList}	"查询成功"
//	@Failure		401			{object}	errcode.Response			"未认证"
//	@Failure		403			{object}	errcode.Response			"无权限"
//	@Router			/admin/model-providers [get]
func (h *ModelHandler) ListProviders(c *gin.Context) {
	page, pageSize, _ := parsePage(c)

	resp, err := h.models.Service().ListProviders(c.Request.Context(), &modelpb.ListProvidersRequest{
		Keyword:    c.Query("keyword"),
		OnlyActive: c.Query("only_active") == "true",
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	items := make([]modeldto.ProviderInfo, 0, len(resp.GetItems()))
	for _, p := range resp.GetItems() {
		items = append(items, toProviderInfo(p))
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, modeldto.ProviderList{
		Items:    items,
		Total:    resp.GetTotal(),
		Page:     page,
		PageSize: pageSize,
	}))
}

// CreateProvider 新增平台渠道
//
//	@Summary		新增平台渠道
//	@Description	新增平台模型渠道，api_key 加密落库后只回显末 4 位
//	@Tags			model-模型与系统配置
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		modeldto.CreateProviderRequest	true	"渠道信息"
//	@Success		200		{object}	errcode.Response{data=modeldto.ProviderInfo}	"创建成功"
//	@Failure		400		{object}	errcode.Response			"参数错误"
//	@Failure		401		{object}	errcode.Response			"未认证"
//	@Failure		403		{object}	errcode.Response			"无权限"
//	@Failure		409		{object}	errcode.Response			"渠道编码已存在"
//	@Router			/admin/model-providers [post]
func (h *ModelHandler) CreateProvider(c *gin.Context) {
	var req modeldto.CreateProviderRequest
	if !bindModelJSON(c, &req) {
		return
	}

	resp, err := h.models.Service().CreateProvider(c.Request.Context(), &modelpb.CreateProviderRequest{
		Code:         req.Code,
		Name:         req.Name,
		Protocol:     req.Protocol,
		BaseUrl:      req.BaseURL,
		ApiKey:       req.APIKey,
		Organization: req.Organization,
		Extra:        req.Extra,
		SortOrder:    req.SortOrder,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, toProviderInfo(resp)))
}

// UpdateProvider 编辑平台渠道
//
//	@Summary		编辑平台渠道
//	@Description	编辑平台渠道；api_key 留空表示不轮换密钥，渠道编码不可修改
//	@Tags			model-模型与系统配置
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int64								true	"渠道 ID"
//	@Param			body	body		modeldto.UpdateProviderRequest	true	"渠道信息"
//	@Success		200		{object}	errcode.Response{data=modeldto.ProviderInfo}	"更新成功"
//	@Failure		400		{object}	errcode.Response			"参数错误"
//	@Failure		401		{object}	errcode.Response			"未认证"
//	@Failure		403		{object}	errcode.Response			"无权限"
//	@Failure		404		{object}	errcode.Response			"渠道不存在"
//	@Router			/admin/model-providers/{id} [put]
func (h *ModelHandler) UpdateProvider(c *gin.Context) {
	id, ok := parseModelID(c, "id")
	if !ok {
		return
	}

	var req modeldto.UpdateProviderRequest
	if !bindModelJSON(c, &req) {
		return
	}

	resp, err := h.models.Service().UpdateProvider(c.Request.Context(), &modelpb.UpdateProviderRequest{
		Id:           id,
		Name:         req.Name,
		Protocol:     req.Protocol,
		BaseUrl:      req.BaseURL,
		ApiKey:       req.APIKey,
		Organization: req.Organization,
		Extra:        req.Extra,
		SortOrder:    req.SortOrder,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, toProviderInfo(resp)))
}

// SetProviderActive 启用 / 停用平台渠道
//
//	@Summary		启用 / 停用平台渠道
//	@Description	停用后该渠道下的模型对用户端全部不可见
//	@Tags			model-模型与系统配置
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int64							true	"渠道 ID"
//	@Param			body	body		modeldto.SetActiveRequest	true	"目标状态"
//	@Success		200		{object}	errcode.Response{data=modeldto.ProviderInfo}	"操作成功"
//	@Failure		400		{object}	errcode.Response			"参数错误"
//	@Failure		401		{object}	errcode.Response			"未认证"
//	@Failure		403		{object}	errcode.Response			"无权限"
//	@Failure		404		{object}	errcode.Response			"渠道不存在"
//	@Router			/admin/model-providers/{id}/active [put]
func (h *ModelHandler) SetProviderActive(c *gin.Context) {
	id, ok := parseModelID(c, "id")
	if !ok {
		return
	}

	var req modeldto.SetActiveRequest
	if !bindModelJSON(c, &req) {
		return
	}

	resp, err := h.models.Service().SetProviderActive(c.Request.Context(), &modelpb.SetProviderActiveRequest{
		Id:       id,
		IsActive: req.IsActive,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, toProviderInfo(resp)))
}

// DeleteProvider 删除平台渠道
//
//	@Summary		删除平台渠道
//	@Description	删除平台渠道；渠道下仍有模型时返回 400（需先清理模型）
//	@Tags			model-模型与系统配置
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int64	true	"渠道 ID"
//	@Success		200	{object}	errcode.Response{data=modeldto.DeleteResult}	"删除成功"
//	@Failure		401	{object}	errcode.Response			"未认证"
//	@Failure		403	{object}	errcode.Response			"无权限"
//	@Failure		404	{object}	errcode.Response			"渠道不存在"
//	@Router			/admin/model-providers/{id} [delete]
func (h *ModelHandler) DeleteProvider(c *gin.Context) {
	id, ok := parseModelID(c, "id")
	if !ok {
		return
	}

	if _, err := h.models.Service().DeleteProvider(c.Request.Context(), &modelpb.DeleteProviderRequest{Id: id}); err != nil {
		writeGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, modeldto.DeleteResult{Success: true}))
}

// ---------- 平台模型（管理端）----------

// ListModels 平台模型列表
//
//	@Summary		平台模型列表
//	@Description	分页查询平台模型，支持按渠道、类型、关键字筛选
//	@Tags			model-模型与系统配置
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int		false	"页码，默认 1"
//	@Param			page_size	query		int		false	"每页条数，默认 20，最大 100"
//	@Param			provider_id	query		int		false	"按渠道过滤，0 表示不限"
//	@Param			model_type	query		string	false	"按类型过滤：chat/embedding/rerank/vision/image/audio/video"
//	@Param			keyword		query		string	false	"按编码、名称或调用标识模糊搜索"
//	@Param			only_active	query		bool	false	"是否只返回上架中的模型"
//	@Success		200			{object}	errcode.Response{data=modeldto.ModelList}	"查询成功"
//	@Failure		401			{object}	errcode.Response			"未认证"
//	@Failure		403			{object}	errcode.Response			"无权限"
//	@Router			/admin/models [get]
func (h *ModelHandler) ListModels(c *gin.Context) {
	page, pageSize, _ := parsePage(c)

	resp, err := h.models.Service().ListModels(c.Request.Context(), &modelpb.ListModelsRequest{
		ProviderId: int64(parseIntDefault(c.Query("provider_id"), 0)),
		ModelType:  c.Query("model_type"),
		Keyword:    c.Query("keyword"),
		OnlyActive: c.Query("only_active") == "true",
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	items := make([]modeldto.ModelInfo, 0, len(resp.GetItems()))
	for _, m := range resp.GetItems() {
		items = append(items, toModelInfo(m))
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, modeldto.ModelList{
		Items:    items,
		Total:    resp.GetTotal(),
		Page:     page,
		PageSize: pageSize,
	}))
}

// CreateModel 新增平台模型
//
//	@Summary		新增平台模型
//	@Description	在指定平台渠道下新增模型
//	@Tags			model-模型与系统配置
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		modeldto.CreateModelRequest	true	"模型信息"
//	@Success		200		{object}	errcode.Response{data=modeldto.ModelInfo}	"创建成功"
//	@Failure		400		{object}	errcode.Response			"参数错误"
//	@Failure		401		{object}	errcode.Response			"未认证"
//	@Failure		403		{object}	errcode.Response			"无权限"
//	@Failure		409		{object}	errcode.Response			"模型编码已存在"
//	@Router			/admin/models [post]
func (h *ModelHandler) CreateModel(c *gin.Context) {
	var req modeldto.CreateModelRequest
	if !bindModelJSON(c, &req) {
		return
	}

	resp, err := h.models.Service().CreateModel(c.Request.Context(), &modelpb.CreateModelRequest{
		ProviderId:      req.ProviderID,
		Code:            req.Code,
		Name:            req.Name,
		ModelType:       req.ModelType,
		ApiModel:        req.APIModel,
		ContextWindow:   req.ContextWindow,
		MaxOutputTokens: req.MaxOutputTokens,
		Capabilities:    toCapabilitiesRequest(req.Capabilities),
		CreditCost:      req.CreditCost,
		Description:     req.Description,
		SortOrder:       req.SortOrder,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, toModelInfo(resp)))
}

// UpdateModel 编辑平台模型
//
//	@Summary		编辑平台模型
//	@Description	编辑平台模型；模型编码与所属渠道均不可修改
//	@Tags			model-模型与系统配置
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int64							true	"模型 ID"
//	@Param			body	body		modeldto.UpdateModelRequest	true	"模型信息"
//	@Success		200		{object}	errcode.Response{data=modeldto.ModelInfo}	"更新成功"
//	@Failure		400		{object}	errcode.Response			"参数错误"
//	@Failure		401		{object}	errcode.Response			"未认证"
//	@Failure		403		{object}	errcode.Response			"无权限"
//	@Failure		404		{object}	errcode.Response			"模型不存在"
//	@Router			/admin/models/{id} [put]
func (h *ModelHandler) UpdateModel(c *gin.Context) {
	id, ok := parseModelID(c, "id")
	if !ok {
		return
	}

	var req modeldto.UpdateModelRequest
	if !bindModelJSON(c, &req) {
		return
	}

	resp, err := h.models.Service().UpdateModel(c.Request.Context(), &modelpb.UpdateModelRequest{
		Id:              id,
		Name:            req.Name,
		ModelType:       req.ModelType,
		ApiModel:        req.APIModel,
		ContextWindow:   req.ContextWindow,
		MaxOutputTokens: req.MaxOutputTokens,
		Capabilities:    toCapabilitiesRequest(req.Capabilities),
		CreditCost:      req.CreditCost,
		Description:     req.Description,
		SortOrder:       req.SortOrder,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, toModelInfo(resp)))
}

// SetModelActive 上架 / 下架平台模型
//
//	@Summary		上架 / 下架平台模型
//	@Description	只影响该模型在用户端的可见性，同渠道其他模型不受影响
//	@Tags			model-模型与系统配置
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int64							true	"模型 ID"
//	@Param			body	body		modeldto.SetActiveRequest	true	"目标状态"
//	@Success		200		{object}	errcode.Response{data=modeldto.ModelInfo}	"操作成功"
//	@Failure		400		{object}	errcode.Response			"参数错误"
//	@Failure		401		{object}	errcode.Response			"未认证"
//	@Failure		403		{object}	errcode.Response			"无权限"
//	@Failure		404		{object}	errcode.Response			"模型不存在"
//	@Router			/admin/models/{id}/active [put]
func (h *ModelHandler) SetModelActive(c *gin.Context) {
	id, ok := parseModelID(c, "id")
	if !ok {
		return
	}

	var req modeldto.SetActiveRequest
	if !bindModelJSON(c, &req) {
		return
	}

	resp, err := h.models.Service().SetModelActive(c.Request.Context(), &modelpb.SetModelActiveRequest{
		Id:       id,
		IsActive: req.IsActive,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, toModelInfo(resp)))
}

// DeleteModel 删除平台模型
//
//	@Summary		删除平台模型
//	@Description	删除平台模型，用户端立即可见性随之消失
//	@Tags			model-模型与系统配置
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int64	true	"模型 ID"
//	@Success		200	{object}	errcode.Response{data=modeldto.DeleteResult}	"删除成功"
//	@Failure		401	{object}	errcode.Response			"未认证"
//	@Failure		403	{object}	errcode.Response			"无权限"
//	@Failure		404	{object}	errcode.Response			"模型不存在"
//	@Router			/admin/models/{id} [delete]
func (h *ModelHandler) DeleteModel(c *gin.Context) {
	id, ok := parseModelID(c, "id")
	if !ok {
		return
	}

	if _, err := h.models.Service().DeleteModel(c.Request.Context(), &modelpb.DeleteModelRequest{Id: id}); err != nil {
		writeGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, modeldto.DeleteResult{Success: true}))
}
