package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/museflow/api-gateway/internal/client"
	"github.com/museflow/api-gateway/internal/config"
	authdto "github.com/museflow/api-gateway/internal/dto/auth_dto"
	// userdto 仅用于 Swagger 注解中的类型引用（用户信息由 user_dto 定义）
	_ "github.com/museflow/api-gateway/internal/dto/user_dto"
	"github.com/museflow/pkg/errcode"
	userpb "github.com/museflow/proto/user"
)

// CommonHandler 公开通用接口处理器（/common 路由组）。
//
// 这些接口无需 access token：发送邮箱验证码、刷新访问令牌。
type CommonHandler struct {
	users *client.UserClient
	cfg   *config.Config
}

// NewCommonHandler 构造公开通用处理器。
func NewCommonHandler(users *client.UserClient, cfg *config.Config) *CommonHandler {
	return &CommonHandler{users: users, cfg: cfg}
}

// SendVerifyCode 发送邮箱验证码
//
//	@Summary		发送邮箱验证码
//	@Description	按场景发送邮箱验证码：register（注册校验）/ login（验证码登录）/ reset_password（密码重置）/ change_email（修改邮箱）；避免账号枚举，邮箱不存在时也返回成功
//	@Tags			common-公共
//	@Accept			json
//	@Produce		json
//	@Param			body	body		authdto.SendVerifyCodeRequest	true	"邮箱与场景"
//	@Success		200		{object}	errcode.Response	"发送成功"
//	@Failure		400		{object}	errcode.Response	"参数校验失败"
//	@Failure		429		{object}	errcode.Response	"发送过于频繁"
//	@Router			/common/email/send-code [post]
func (h *CommonHandler) SendVerifyCode(c *gin.Context) {
	var req authdto.SendVerifyCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errcode.Fail(errcode.CodeParamInvalid, err.Error()))
		return
	}

	if _, err := h.users.Service().SendVerifyCode(c.Request.Context(), &userpb.SendVerifyCodeRequest{
		Email: req.Email,
		Scene: req.Scene,
	}); err != nil {
		writeGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, errcode.SuccessGin(c, nil))
}

// Refresh 刷新访问令牌
//
//	@Summary		刷新访问令牌
//	@Description	从 Cookie 读取 refresh token 换取新的 access token，刷新后旧 refresh 轮转
//	@Tags			common-公共
//	@Produce		json
//	@Success		200	{object}	errcode.Response{data=authdto.RefreshData}	"刷新成功"
//	@Failure		401	{object}	errcode.Response				"刷新令牌无效或已过期"
//	@Failure		403	{object}	errcode.Response				"设备校验失败"
//	@Router			/common/refresh [post]
func (h *CommonHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie(refreshCookieName)
	if err != nil || refreshToken == "" {
		c.JSON(http.StatusUnauthorized, errcode.ErrorGin(c, errcode.CodeMissingRefresh))
		return
	}
	deviceID, err := c.Cookie(deviceCookieName)
	if err != nil || deviceID == "" {
		c.JSON(http.StatusUnauthorized, errcode.ErrorGin(c, errcode.CodeMissingDevice))
		return
	}

	resp, err := h.users.Service().Refresh(c.Request.Context(), &userpb.RefreshRequest{
		RefreshToken: refreshToken,
		Device: &userpb.DeviceContext{
			DeviceId:  deviceID,
			UserAgent: c.Request.UserAgent(),
			Ip:        c.ClientIP(),
		},
	})
	if err != nil {
		// 刷新失败说明会话已不可用，清除 Cookie 避免客户端反复重试
		if status.Code(err) == codes.Unauthenticated {
			clearCookies(c, h.cfg)
		}
		writeGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, errcode.SuccessGin(c, authdto.RefreshData{
		AccessToken: resp.GetAccessToken(),
		TokenType:   "Bearer",
		ExpiresIn:   resp.GetExpiresIn(),
	}))
}
