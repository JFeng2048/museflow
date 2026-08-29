package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/museflow/api-gateway/internal/client"
	"github.com/museflow/api-gateway/internal/dto"
	"github.com/museflow/api-gateway/internal/middleware"
	"github.com/museflow/pkg/errcode"
	"github.com/museflow/pkg/logger"
	userpb "github.com/museflow/proto/user"
)

// UserManageHandler 用户自助管理处理器（资料、密码、会话、第三方账号）。
type UserManageHandler struct {
	users *client.UserClient
}

// NewUserManageHandler 构造用户自助管理处理器。
func NewUserManageHandler(users *client.UserClient) *UserManageHandler {
	return &UserManageHandler{users: users}
}

// UpdateProfile 更新个人信息
//
//	@Summary		更新当前用户信息
//	@Description	更新昵称 / 头像 / 简介，传入空字符串的字段不做修改
//	@Tags			用户
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.UpdateProfileRequest	true	"更新信息"
//	@Success		200		{object}	errcode.Response{data=dto.UserInfo}	"更新成功"
//	@Failure		400		{object}	errcode.Response			"参数校验失败"
//	@Failure		401		{object}	errcode.Response			"未认证"
//	@Router			/user/profile [put]
func (h *UserManageHandler) UpdateProfile(c *gin.Context) {
	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errcode.Fail(errcode.CodeParamInvalid, err.Error()))
		return
	}

	uid := middleware.CurrentUserUUID(c)
	resp, err := h.users.Service().UpdateProfile(c.Request.Context(), &userpb.UpdateProfileRequest{
		Uuid:      uid,
		Nickname:  req.Nickname,
		AvatarUrl: req.AvatarURL,
		Bio:       req.Bio,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, errcode.SuccessGin(c, toUserInfo(resp.GetUser())))
}

// ChangePassword 修改密码
//
//	@Summary		修改当前用户密码
//	@Description	校验旧密码后设置新密码，成功后清理该用户的权限缓存
//	@Tags			用户
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.ChangePasswordRequest	true	"新旧密码"
//	@Success		200		{object}	errcode.Response	"修改成功"
//	@Failure		400		{object}	errcode.Response	"参数校验失败"
//	@Failure		401		{object}	errcode.Response	"旧密码错误"
//	@Router			/user/password [put]
func (h *UserManageHandler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errcode.Fail(errcode.CodeParamInvalid, err.Error()))
		return
	}

	uid := middleware.CurrentUserUUID(c)
	if _, err := h.users.Service().ChangePassword(c.Request.Context(), &userpb.ChangePasswordRequest{
		Uuid:        uid,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	}); err != nil {
		writeGRPCError(c, err)
		return
	}

	logger.InfoContext(c.Request.Context(), "网关修改密码成功", logger.UserUUID(uid))
	c.JSON(http.StatusOK, errcode.SuccessGin(c, nil))
}

// ListSessions 查看当前用户的活跃会话
//
//	@Summary		获取当前用户会话列表
//	@Description	返回当前账号的登录设备与登录时间，可据此强制下线指定设备
//	@Tags			用户
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	errcode.Response{data=dto.SessionList}	"获取成功"
//	@Failure		401	{object}	errcode.Response			"未认证"
//	@Router			/user/sessions [get]
func (h *UserManageHandler) ListSessions(c *gin.Context) {
	uid := middleware.CurrentUserUUID(c)
	resp, err := h.users.Service().ListSessions(c.Request.Context(), &userpb.ListSessionsRequest{Uuid: uid})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	sessions := make([]dto.SessionInfo, 0, len(resp.GetSessions()))
	for _, s := range resp.GetSessions() {
		sessions = append(sessions, dto.SessionInfo{
			TokenID:         s.GetTokenId(),
			DeviceID:        s.GetDeviceId(),
			DeviceName:      s.GetDeviceName(),
			LoginTime:       s.GetLoginTime(),
			LastRefreshTime: s.GetLastRefreshTime(),
		})
	}

	c.JSON(http.StatusOK, errcode.SuccessGin(c, dto.SessionList{Sessions: sessions}))
}

// RevokeSession 强制下线指定会话
//
//	@Summary		强制下线指定会话
//	@Description	吊销指定设备的刷新令牌，使其 access token 到期后无法续期
//	@Tags			用户
//	@Produce		json
//	@Security		BearerAuth
//	@Param			token	path		string	true	"会话标识"
//	@Success		200		{object}	errcode.Response	"下线成功"
//	@Failure		401		{object}	errcode.Response	"未认证"
//	@Router			/user/sessions/{token} [delete]
func (h *UserManageHandler) RevokeSession(c *gin.Context) {
	tokenID := c.Param("token")
	if tokenID == "" {
		c.JSON(http.StatusBadRequest, errcode.ErrorGin(c, errcode.CodeParamInvalid))
		return
	}

	uid := middleware.CurrentUserUUID(c)
	if _, err := h.users.Service().RevokeSession(c.Request.Context(), &userpb.RevokeSessionRequest{
		Uuid:    uid,
		TokenId: tokenID,
	}); err != nil {
		writeGRPCError(c, err)
		return
	}

	logger.InfoContext(c.Request.Context(), "网关强制下线会话成功", logger.UserUUID(uid))
	c.JSON(http.StatusOK, errcode.SuccessGin(c, nil))
}

// MyPermissions 获取当前用户的权限列表
//
//	@Summary		获取当前用户权限
//	@Description	返回当前用户拥有的全部权限编码，供前端做菜单与按钮级控制
//	@Tags			用户
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	errcode.Response{data=dto.PermissionListData}	"获取成功"
//	@Failure		401	{object}	errcode.Response				"未认证"
//	@Router			/user/permissions [get]
func (h *UserManageHandler) MyPermissions(c *gin.Context) {
	uid := middleware.CurrentUserUUID(c)
	resp, err := h.users.Service().GetUserPermissions(c.Request.Context(), &userpb.GetUserPermissionsRequest{Uuid: uid})
	if err != nil {
		writeGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, errcode.SuccessGin(c, dto.PermissionListData{Permissions: resp.GetPermissions()}))
}

// ListOAuthBindings 列出已绑定的第三方账号
//
//	@Summary		第三方账号绑定列表
//	@Tags			用户
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	errcode.Response{data=dto.OAuthBindingList}	"获取成功"
//	@Failure		401	{object}	errcode.Response				"未认证"
//	@Router			/user/oauth [get]
func (h *UserManageHandler) ListOAuthBindings(c *gin.Context) {
	uid := middleware.CurrentUserUUID(c)
	resp, err := h.users.Service().ListOAuthBindings(c.Request.Context(), &userpb.ListOAuthBindingsRequest{Uuid: uid})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	bindings := make([]dto.OAuthBinding, 0, len(resp.GetBindings()))
	for _, b := range resp.GetBindings() {
		bindings = append(bindings, dto.OAuthBinding{
			Provider:         b.GetProvider(),
			ProviderUserID:   b.GetProviderUserId(),
			ProviderEmail:    b.GetProviderEmail(),
			ProviderNickname: b.GetProviderNickname(),
			ProviderAvatar:   b.GetProviderAvatar(),
			LastLoginAt:      b.GetLastLoginAt(),
			CreatedAt:        b.GetCreatedAt(),
		})
	}

	c.JSON(http.StatusOK, errcode.SuccessGin(c, dto.OAuthBindingList{Bindings: bindings}))
}

// UnbindOAuth 解绑第三方账号
//
//	@Summary		解绑第三方账号
//	@Tags			用户
//	@Produce		json
//	@Security		BearerAuth
//	@Param			provider	path		string	true	"平台标识，如 github"
//	@Success		200			{object}	errcode.Response	"解绑成功"
//	@Failure		401			{object}	errcode.Response	"未认证"
//	@Failure		404			{object}	errcode.Response	"未绑定该账号"
//	@Router			/user/oauth/{provider} [delete]
func (h *UserManageHandler) UnbindOAuth(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, errcode.ErrorGin(c, errcode.CodeParamInvalid))
		return
	}

	uid := middleware.CurrentUserUUID(c)
	if _, err := h.users.Service().UnbindOAuth(c.Request.Context(), &userpb.UnbindOAuthRequest{
		Uuid:     uid,
		Provider: provider,
	}); err != nil {
		writeGRPCError(c, err)
		return
	}

	logger.InfoContext(c.Request.Context(), "网关解绑第三方账号成功", logger.UserUUID(uid), "provider", provider)
	c.JSON(http.StatusOK, errcode.SuccessGin(c, nil))
}

// parsePage 解析分页参数，返回 page / pageSize / offset。
func parsePage(c *gin.Context) (page, pageSize, offset int32) {
	page = int32(parseIntDefault(c.Query("page"), 1))
	pageSize = int32(parseIntDefault(c.Query("page_size"), 20))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize, (page - 1) * pageSize
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return def
	}
	return v
}
