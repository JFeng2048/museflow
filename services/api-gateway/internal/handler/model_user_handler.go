package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	modeldto "github.com/museflow/api-gateway/internal/dto/model_dto"
	"github.com/museflow/pkg/errcode"
	modelpb "github.com/museflow/proto/model"
)

// ---------- 用户自定义渠道 ----------

// ListUserProviders 我的自定义渠道列表
//
//	@Summary		我的自定义渠道列表
//	@Description	列出当前用户添加的大模型渠道（自带 base_url 与 api_key），api_key 只回显末 4 位
//	@Tags			model-模型与系统配置
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int	false	"页码，默认 1"
//	@Param			page_size	query		int	false	"每页条数，默认 20，最大 100"
//	@Success		200			{object}	errcode.Response{data=modeldto.UserProviderList}	"查询成功"
//	@Failure		401			{object}	errcode.Response			"未登录"
//	@Router			/user/model-providers [get]
func (h *ModelHandler) ListUserProviders(c *gin.Context) {
	uid, ok := requireLogin(c)
	if !ok {
		return
	}
	page, pageSize, _ := parsePage(c)

	resp, err := h.models.Service().ListUserProviders(c.Request.Context(), &modelpb.ListUserProvidersRequest{
		UserUuid: uid,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	items := make([]modeldto.UserProviderInfo, 0, len(resp.GetItems()))
	for _, p := range resp.GetItems() {
		items = append(items, toUserProviderInfo(p))
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, modeldto.UserProviderList{
		Items:    items,
		Total:    resp.GetTotal(),
		Page:     page,
		PageSize: pageSize,
	}))
}

// CreateUserProvider 添加自定义渠道
//
//	@Summary		添加自定义渠道
//	@Description	添加用户自有的大模型渠道；api_key 加密落库后只回显末 4 位
//	@Tags			model-模型与系统配置
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		modeldto.UserProviderRequest	true	"渠道信息"
//	@Success		200		{object}	errcode.Response{data=modeldto.UserProviderInfo}	"添加成功"
//	@Failure		400		{object}	errcode.Response			"参数错误"
//	@Failure		401		{object}	errcode.Response			"未登录"
//	@Router			/user/model-providers [post]
func (h *ModelHandler) CreateUserProvider(c *gin.Context) {
	uid, ok := requireLogin(c)
	if !ok {
		return
	}

	var req modeldto.UserProviderRequest
	if !bindModelJSON(c, &req) {
		return
	}

	resp, err := h.models.Service().CreateUserProvider(c.Request.Context(), &modelpb.CreateUserProviderRequest{
		UserUuid:     uid,
		Name:         req.Name,
		Protocol:     req.Protocol,
		BaseUrl:      req.BaseURL,
		ApiKey:       req.APIKey,
		Organization: req.Organization,
		Extra:        req.Extra,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, toUserProviderInfo(resp)))
}

// UpdateUserProvider 编辑自定义渠道
//
//	@Summary		编辑自定义渠道
//	@Description	编辑我的自定义渠道；api_key 留空表示不轮换密钥
//	@Tags			model-模型与系统配置
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int64							true	"渠道 ID"
//	@Param			body	body		modeldto.UserProviderRequest	true	"渠道信息"
//	@Success		200		{object}	errcode.Response{data=modeldto.UserProviderInfo}	"更新成功"
//	@Failure		400		{object}	errcode.Response			"参数错误"
//	@Failure		401		{object}	errcode.Response			"未登录"
//	@Failure		404		{object}	errcode.Response			"渠道不存在"
//	@Router			/user/model-providers/{id} [put]
func (h *ModelHandler) UpdateUserProvider(c *gin.Context) {
	uid, ok := requireLogin(c)
	if !ok {
		return
	}

	id, ok := parseModelID(c, "id")
	if !ok {
		return
	}

	var req modeldto.UserProviderRequest
	if !bindModelJSON(c, &req) {
		return
	}

	// user_uuid 由 token 注入，同时作为归属校验依据：改不到别人的渠道。
	resp, err := h.models.Service().UpdateUserProvider(c.Request.Context(), &modelpb.UpdateUserProviderRequest{
		Id:           id,
		UserUuid:     uid,
		Name:         req.Name,
		Protocol:     req.Protocol,
		BaseUrl:      req.BaseURL,
		ApiKey:       req.APIKey,
		Organization: req.Organization,
		Extra:        req.Extra,
		IsActive:     req.IsActive,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, toUserProviderInfo(resp)))
}

// DeleteUserProvider 删除自定义渠道
//
//	@Summary		删除自定义渠道
//	@Description	删除我的自定义渠道；渠道下仍有模型时返回 400（需先清理模型）
//	@Tags			model-模型与系统配置
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int64	true	"渠道 ID"
//	@Success		200	{object}	errcode.Response{data=modeldto.DeleteResult}	"删除成功"
//	@Failure		400	{object}	errcode.Response			"渠道下仍有模型"
//	@Failure		401	{object}	errcode.Response			"未登录"
//	@Failure		404	{object}	errcode.Response			"渠道不存在"
//	@Router			/user/model-providers/{id} [delete]
func (h *ModelHandler) DeleteUserProvider(c *gin.Context) {
	uid, ok := requireLogin(c)
	if !ok {
		return
	}

	id, ok := parseModelID(c, "id")
	if !ok {
		return
	}

	if _, err := h.models.Service().DeleteUserProvider(c.Request.Context(), &modelpb.DeleteUserProviderRequest{
		Id:       id,
		UserUuid: uid,
	}); err != nil {
		writeGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, modeldto.DeleteResult{Success: true}))
}

// ---------- 用户自选模型 ----------

// ListUserModels 我的自定义模型列表
//
//	@Summary		我的自定义模型列表
//	@Description	列出当前用户在自定义渠道下登记的模型
//	@Tags			model-模型与系统配置
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page				query		int		false	"页码，默认 1"
//	@Param			page_size			query		int		false	"每页条数，默认 20，最大 100"
//	@Param			user_provider_id	query		int		false	"按自定义渠道过滤，0 表示不限"
//	@Success		200					{object}	errcode.Response{data=modeldto.UserModelList}	"查询成功"
//	@Failure		401					{object}	errcode.Response			"未登录"
//	@Router			/user/models [get]
func (h *ModelHandler) ListUserModels(c *gin.Context) {
	uid, ok := requireLogin(c)
	if !ok {
		return
	}
	page, pageSize, _ := parsePage(c)

	resp, err := h.models.Service().ListUserModels(c.Request.Context(), &modelpb.ListUserModelsRequest{
		UserUuid:       uid,
		UserProviderId: int64(parseIntDefault(c.Query("user_provider_id"), 0)),
		Page:           page,
		PageSize:       pageSize,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	items := make([]modeldto.UserModelInfo, 0, len(resp.GetItems()))
	for _, m := range resp.GetItems() {
		items = append(items, toUserModelInfo(m))
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, modeldto.UserModelList{
		Items:    items,
		Total:    resp.GetTotal(),
		Page:     page,
		PageSize: pageSize,
	}))
}

// CreateUserModel 添加自定义模型
//
//	@Summary		添加自定义模型
//	@Description	在我的某个自定义渠道下登记一个模型；不计平台积分
//	@Tags			model-模型与系统配置
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		modeldto.UserModelRequest	true	"模型信息"
//	@Success		200		{object}	errcode.Response{data=modeldto.UserModelInfo}	"添加成功"
//	@Failure		400		{object}	errcode.Response			"参数错误"
//	@Failure		401		{object}	errcode.Response			"未登录"
//	@Failure		404		{object}	errcode.Response			"渠道不存在"
//	@Router			/user/models [post]
func (h *ModelHandler) CreateUserModel(c *gin.Context) {
	uid, ok := requireLogin(c)
	if !ok {
		return
	}

	var req modeldto.UserModelRequest
	if !bindModelJSON(c, &req) {
		return
	}

	resp, err := h.models.Service().CreateUserModel(c.Request.Context(), &modelpb.CreateUserModelRequest{
		UserUuid:        uid,
		UserProviderId:  req.ProviderID,
		Name:            req.Name,
		ModelType:       req.ModelType,
		ApiModel:        req.APIModel,
		ContextWindow:   req.ContextWindow,
		MaxOutputTokens: req.MaxOutputTokens,
		Capabilities:    toCapabilitiesRequest(req.Capabilities),
		Description:     req.Description,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, toUserModelInfo(resp)))
}

// UpdateUserModel 编辑自定义模型
//
//	@Summary		编辑自定义模型
//	@Description	编辑我的自定义模型；所属渠道不可修改
//	@Tags			model-模型与系统配置
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int64						true	"模型 ID"
//	@Param			body	body		modeldto.UserModelRequest	true	"模型信息"
//	@Success		200		{object}	errcode.Response{data=modeldto.UserModelInfo}	"更新成功"
//	@Failure		400		{object}	errcode.Response			"参数错误"
//	@Failure		401		{object}	errcode.Response			"未登录"
//	@Failure		404		{object}	errcode.Response			"模型不存在"
//	@Router			/user/models/{id} [put]
func (h *ModelHandler) UpdateUserModel(c *gin.Context) {
	uid, ok := requireLogin(c)
	if !ok {
		return
	}

	id, ok := parseModelID(c, "id")
	if !ok {
		return
	}

	var req modeldto.UserModelRequest
	if !bindModelJSON(c, &req) {
		return
	}

	resp, err := h.models.Service().UpdateUserModel(c.Request.Context(), &modelpb.UpdateUserModelRequest{
		Id:              id,
		UserUuid:        uid,
		Name:            req.Name,
		ModelType:       req.ModelType,
		ApiModel:        req.APIModel,
		ContextWindow:   req.ContextWindow,
		MaxOutputTokens: req.MaxOutputTokens,
		Capabilities:    toCapabilitiesRequest(req.Capabilities),
		Description:     req.Description,
		IsActive:        req.IsActive,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, toUserModelInfo(resp)))
}

// DeleteUserModel 删除自定义模型
//
//	@Summary		删除自定义模型
//	@Description	删除我的自定义模型
//	@Tags			model-模型与系统配置
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int64	true	"模型 ID"
//	@Success		200	{object}	errcode.Response{data=modeldto.DeleteResult}	"删除成功"
//	@Failure		401	{object}	errcode.Response			"未登录"
//	@Failure		404	{object}	errcode.Response			"模型不存在"
//	@Router			/user/models/{id} [delete]
func (h *ModelHandler) DeleteUserModel(c *gin.Context) {
	uid, ok := requireLogin(c)
	if !ok {
		return
	}

	id, ok := parseModelID(c, "id")
	if !ok {
		return
	}

	if _, err := h.models.Service().DeleteUserModel(c.Request.Context(), &modelpb.DeleteUserModelRequest{
		Id:       id,
		UserUuid: uid,
	}); err != nil {
		writeGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, modeldto.DeleteResult{Success: true}))
}
