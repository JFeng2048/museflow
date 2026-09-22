package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	modeldto "github.com/museflow/api-gateway/internal/dto/model_dto"
	"github.com/museflow/pkg/errcode"
	modelpb "github.com/museflow/proto/model"
)

// ---------- 上游模型目录探测 ----------

// FetchAdminRemoteModels 拉取平台渠道的模型目录
//
//	@Summary		拉取平台渠道模型目录
//	@Description	按 provider_id 读取已保存的平台渠道配置，请求上游 /models 接口并返回可登记的模型清单；api_key 由服务端解密使用，请求与响应都不携带密钥明文
//	@Tags			model-模型与系统配置
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		modeldto.FetchRemoteModelsRequest	true	"渠道定位，provider_id 必填"
//	@Success		200		{object}	errcode.Response{data=modeldto.RemoteModelList}	"查询成功"
//	@Failure		400		{object}	errcode.Response			"参数错误或上游调用失败"
//	@Failure		401		{object}	errcode.Response			"未认证"
//	@Failure		403		{object}	errcode.Response			"无权限"
//	@Failure		404		{object}	errcode.Response			"渠道不存在"
//	@Router			/admin/model-providers/remote-models [post]
func (h *ModelHandler) FetchAdminRemoteModels(c *gin.Context) {
	var req modeldto.FetchRemoteModelsRequest
	if !bindModelJSON(c, &req) {
		return
	}

	resp, err := h.models.Service().FetchProviderModels(c.Request.Context(), &modelpb.FetchProviderModelsRequest{
		ProviderId: req.ProviderID,
		Protocol:   req.Protocol,
		BaseUrl:    req.BaseURL,
		ApiKey:     req.APIKey,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, toRemoteModelList(resp)))
}

// FetchUserRemoteModels 拉取我的自定义渠道的模型目录
//
//	@Summary		拉取我的渠道模型目录
//	@Description	按 user_provider_id 读取当前用户自己的渠道配置并请求上游 /models 接口；user_uuid 取自登录态，取不到别人的渠道
//	@Tags			model-模型与系统配置
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		modeldto.FetchRemoteModelsRequest	true	"渠道定位，provider_id 填自定义渠道 ID"
//	@Success		200		{object}	errcode.Response{data=modeldto.RemoteModelList}	"查询成功"
//	@Failure		400		{object}	errcode.Response			"参数错误或上游调用失败"
//	@Failure		401		{object}	errcode.Response			"未登录"
//	@Failure		404		{object}	errcode.Response			"渠道不存在"
//	@Router			/user/model-providers/remote-models [post]
func (h *ModelHandler) FetchUserRemoteModels(c *gin.Context) {
	uid, ok := requireLogin(c)
	if !ok {
		return
	}

	var req modeldto.FetchRemoteModelsRequest
	if !bindModelJSON(c, &req) {
		return
	}

	// user_uuid 由 token 注入并作为归属校验依据，不接受请求体里的覆盖。
	resp, err := h.models.Service().FetchProviderModels(c.Request.Context(), &modelpb.FetchProviderModelsRequest{
		UserProviderId: req.ProviderID,
		UserUuid:       uid,
		Protocol:       req.Protocol,
		BaseUrl:        req.BaseURL,
		ApiKey:         req.APIKey,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, toRemoteModelList(resp)))
}

// toRemoteModelList 转换上游模型目录响应。
func toRemoteModelList(resp *modelpb.FetchProviderModelsResponse) modeldto.RemoteModelList {
	items := make([]modeldto.RemoteModelInfo, 0, len(resp.GetItems()))
	for _, m := range resp.GetItems() {
		items = append(items, modeldto.RemoteModelInfo{
			ID:        m.GetId(),
			Object:    m.GetObject(),
			OwnedBy:   m.GetOwnedBy(),
			CreatedAt: m.GetCreatedAt(),
		})
	}
	return modeldto.RemoteModelList{Items: items, Total: resp.GetTotal()}
}
