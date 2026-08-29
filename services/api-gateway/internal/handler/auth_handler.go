// Package handler 实现 api-gateway 的 HTTP 处理器，将 HTTP 请求转发到 gRPC 服务。
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
	"github.com/museflow/api-gateway/internal/middleware"
	"github.com/museflow/pkg/errcode"
	"github.com/museflow/pkg/logger"
	userpb "github.com/museflow/proto/user"
)

// Cookie 名称。
const (
	refreshCookieName = "refresh_token"
	deviceCookieName  = "device_id"
)

// AuthHandler 认证相关处理器。
type AuthHandler struct {
	users *client.UserClient
	cfg   *config.Config
}

// NewAuthHandler 构造认证处理器。
func NewAuthHandler(users *client.UserClient, cfg *config.Config) *AuthHandler {
	return &AuthHandler{users: users, cfg: cfg}
}

// Register 用户注册
//
//	@Summary		用户注册
//	@Description	使用邮箱和密码注册新用户，密码以 bcrypt 加密存储
//	@Tags			认证
//	@Accept			json
//	@Produce		json
//	@Param			body	body		authdto.RegisterRequest	true	"注册信息"
//	@Success		200		{object}	errcode.Response{data=userdto.UserInfo}	"注册成功"
//	@Failure		400		{object}	errcode.Response			"参数校验失败"
//	@Failure		409		{object}	errcode.Response			"邮箱已被注册"
//	@Failure		500		{object}	errcode.Response			"服务内部错误"
//	@Router			/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req authdto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errcode.Fail(errcode.CodeParamInvalid, err.Error()))
		return
	}

	logger.InfoContext(c.Request.Context(), "网关收到注册请求", "email", req.Email)
	resp, err := h.users.Service().Register(c.Request.Context(), &userpb.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
		Nickname: req.Nickname,
	})
	if err != nil {
		logger.WarnContext(c.Request.Context(), "网关注册失败", "email", req.Email)
		writeGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, errcode.SuccessGin(c, toUserInfo(resp.GetUser())))
}

// Login 用户登录
//
//	@Summary		用户登录
//	@Description	邮箱密码登录，成功后返回 access token（body）并将 refresh token 写入 HttpOnly Cookie
//	@Tags			认证
//	@Accept			json
//	@Produce		json
//	@Param			body	body		authdto.LoginRequest	true	"登录信息"
//	@Success		200		{object}	errcode.Response{data=authdto.LoginData}	"登录成功"
//	@Failure		400		{object}	errcode.Response				"参数校验失败"
//	@Failure		401		{object}	errcode.Response				"邮箱或密码错误"
//	@Failure		500		{object}	errcode.Response				"服务内部错误"
//	@Router			/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req authdto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errcode.Fail(errcode.CodeParamInvalid, err.Error()))
		return
	}

	// 复用客户端已有 device_id，实现同设备重复登录不新增设备记录
	deviceID, _ := c.Cookie(deviceCookieName)

	logger.InfoContext(c.Request.Context(), "网关收到登录请求", "email", req.Email)
	resp, err := h.users.Service().Login(c.Request.Context(), &userpb.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
		Device: &userpb.DeviceContext{
			DeviceId:   deviceID,
			UserAgent:  c.Request.UserAgent(),
			Ip:         c.ClientIP(),
			DeviceName: req.DeviceName,
		},
	})
	if err != nil {
		logger.WarnContext(c.Request.Context(), "网关登录失败", "email", req.Email)
		writeGRPCError(c, err)
		return
	}

	// refresh token 走 HttpOnly Cookie 防 XSS；device_id 需被 JS 读取故不设 HttpOnly
	h.setCookie(c, refreshCookieName, resp.GetRefreshToken(), int(resp.GetRefreshExpiresIn()), true)
	h.setCookie(c, deviceCookieName, resp.GetDeviceId(), int(resp.GetRefreshExpiresIn()), false)

	c.JSON(http.StatusOK, errcode.SuccessGin(c, authdto.LoginData{
		AccessToken: resp.GetAccessToken(),
		TokenType:   "Bearer",
		ExpiresIn:   resp.GetExpiresIn(),
		User:        toUserInfo(resp.GetUser()),
	}))
}

// Refresh 刷新访问令牌
//
//	@Summary		刷新访问令牌
//	@Description	从 Cookie 读取 refresh token 换取新的 access token，refresh token 本身不轮换
//	@Tags			认证
//	@Produce		json
//	@Success		200	{object}	errcode.Response{data=authdto.RefreshData}	"刷新成功"
//	@Failure		401	{object}	errcode.Response				"刷新令牌无效或已过期"
//	@Failure		403	{object}	errcode.Response				"设备校验失败"
//	@Router			/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
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
			h.clearCookies(c)
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

// Logout 用户登出
//
//	@Summary		用户登出
//	@Description	删除 Redis 中的 refresh token 白名单、将当前 access token 加入黑名单，并清除 Cookie
//	@Tags			认证
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	errcode.Response		"登出成功"
//	@Failure		401	{object}	errcode.Response	"未认证"
//	@Router			/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// 中间件已校验通过，此处取出原始 token 用于加入黑名单
	accessToken := middleware.ExtractBearerToken(c)
	refreshToken, _ := c.Cookie(refreshCookieName)

	_, err := h.users.Service().Logout(c.Request.Context(), &userpb.LogoutRequest{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	h.clearCookies(c)
	logger.InfoContext(c.Request.Context(), "网关登出成功")
	c.JSON(http.StatusOK, errcode.SuccessGin(c, nil))
}

// SendResetCode 发送密码重置验证码
//
//	@Summary		发送密码重置验证码
//	@Description	向邮箱发送 6 位数字验证码；为避免账号枚举，邮箱不存在时也返回成功
//	@Tags			认证
//	@Accept			json
//	@Produce		json
//	@Param			body	body		authdto.SendResetCodeRequest	true	"邮箱"
//	@Success		200		{object}	errcode.Response	"发送成功"
//	@Failure		400		{object}	errcode.Response	"参数校验失败"
//	@Failure		429		{object}	errcode.Response	"发送过于频繁"
//	@Router			/auth/password/reset-code [post]
func (h *AuthHandler) SendResetCode(c *gin.Context) {
	var req authdto.SendResetCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errcode.Fail(errcode.CodeParamInvalid, err.Error()))
		return
	}

	if _, err := h.users.Service().SendResetCode(c.Request.Context(), &userpb.SendResetCodeRequest{Email: req.Email}); err != nil {
		writeGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, errcode.SuccessGin(c, nil))
}

// ResetPassword 重置密码
//
//	@Summary		通过邮箱验证码重置密码
//	@Description	校验验证码后设置新密码；验证码一次性使用，重置后清理权限缓存
//	@Tags			认证
//	@Accept			json
//	@Produce		json
//	@Param			body	body		authdto.ResetPasswordRequest	true	"邮箱、验证码与新密码"
//	@Success		200		{object}	errcode.Response	"重置成功"
//	@Failure		400		{object}	errcode.Response	"参数校验失败或验证码错误"
//	@Failure		404		{object}	errcode.Response	"用户不存在"
//	@Router			/auth/password/reset [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req authdto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errcode.Fail(errcode.CodeParamInvalid, err.Error()))
		return
	}

	if _, err := h.users.Service().ResetPassword(c.Request.Context(), &userpb.ResetPasswordRequest{
		Email:       req.Email,
		Code:        req.Code,
		NewPassword: req.NewPassword,
	}); err != nil {
		writeGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, errcode.SuccessGin(c, nil))
}

// setCookie 按配置写入 Cookie。
func (h *AuthHandler) setCookie(c *gin.Context, name, value string, maxAge int, httpOnly bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   h.cfg.CookieDomain,
		MaxAge:   maxAge,
		Secure:   h.cfg.CookieSecure,
		HttpOnly: httpOnly,
		SameSite: parseSameSite(h.cfg.CookieSameSite),
	})
}

// clearCookies 清除双令牌相关 Cookie（MaxAge<0 表示立即删除）。
func (h *AuthHandler) clearCookies(c *gin.Context) {
	h.setCookie(c, refreshCookieName, "", -1, true)
	h.setCookie(c, deviceCookieName, "", -1, false)
}

func parseSameSite(v string) http.SameSite {
	switch v {
	case "strict", "Strict":
		return http.SameSiteStrictMode
	case "none", "None":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
