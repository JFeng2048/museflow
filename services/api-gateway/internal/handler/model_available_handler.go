package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	modeldto "github.com/museflow/api-gateway/internal/dto/model_dto"
	"github.com/museflow/pkg/errcode"
	modelpb "github.com/museflow/proto/model"
)

// ListAvailableModels 当前用户可用的模型清单
//
//	@Summary		可用模型清单
//	@Description	合并「平台已上架模型」与「我的自定义模型」，供前端模型选择器使用；不含 base_url 与任何密钥
//	@Tags			model-模型与系统配置
//	@Produce		json
//	@Security		BearerAuth
//	@Param			model_type	query		string	false	"按类型过滤：chat/embedding/rerank/vision/image/audio/video"
//	@Success		200			{object}	errcode.Response{data=modeldto.AvailableModelList}	"查询成功"
//	@Failure		401			{object}	errcode.Response			"未登录"
//	@Router			/user/models/available [get]
func (h *ModelHandler) ListAvailableModels(c *gin.Context) {
	uid, ok := requireLogin(c)
	if !ok {
		return
	}

	resp, err := h.models.Service().ListAvailableModels(c.Request.Context(), &modelpb.ListAvailableModelsRequest{
		UserUuid:  uid,
		ModelType: c.Query("model_type"),
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	items := make([]modeldto.AvailableModel, 0, len(resp.GetItems()))
	for _, m := range resp.GetItems() {
		items = append(items, toAvailableModel(m))
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, modeldto.AvailableModelList{Items: items}))
}
