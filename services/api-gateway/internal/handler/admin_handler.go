package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/museflow/api-gateway/internal/client"
	"github.com/museflow/api-gateway/internal/dto"
	"github.com/museflow/api-gateway/internal/middleware"
	"github.com/museflow/pkg/errcode"
	userpb "github.com/museflow/proto/user"
)

// AdminHandler 管理后台处理器。
//
// 说明：管理后台接口需 admin 权限，由路由层的
// middleware.RequirePermission(userClient, "user:admin") 统一校验，
// 处理器本身不再重复做权限判断。
type AdminHandler struct {
	users *client.UserClient
}

// NewAdminHandler 构造管理后台处理器。
func NewAdminHandler(users *client.UserClient) *AdminHandler {
	return &AdminHandler{users: users}
}

// ListUsers 用户列表
//
//	@Summary		用户列表
//	@Description	分页查询用户，支持关键字（邮箱 / 昵称）与状态筛选、排序
//	@Tags			管理后台
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int		false	"页码，默认 1"
//	@Param			page_size	query		int		false	"每页条数，默认 20，最大 100"
//	@Param			keyword		query		string	false	"按邮箱或昵称模糊搜索"
//	@Param			status		query		int		false	"状态：1=正常 2=冻结 3=已注销 4=待审核"
//	@Param			order_by	query		string	false	"排序字段：created_at/updated_at/email/nickname/status"
//	@Param			desc		query		bool	false	"是否倒序"
//	@Success		200			{object}	errcode.Response{data=dto.UserList}	"查询成功"
//	@Failure		401			{object}	errcode.Response			"未认证"
//	@Failure		403			{object}	errcode.Response			"无权限"
//	@Router			/admin/users [get]
func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, pageSize, offset := parsePage(c)

	resp, err := h.users.Service().ListUsers(c.Request.Context(), &userpb.ListUsersRequest{
		Pagination: &userpb.Pagination{Page: page, PageSize: pageSize},
		Keyword:    c.Query("keyword"),
		Status:     int32(parseIntDefault(c.Query("status"), 0)),
		OrderBy:    c.Query("order_by"),
		Desc:       c.Query("desc") == "true",
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}
	_ = offset // 分页由 proto 的 Pagination 传递

	users := make([]dto.AdminUserItem, 0, len(resp.GetUsers()))
	for _, u := range resp.GetUsers() {
		users = append(users, dto.AdminUserItem{
			User:      toUserInfo(u.GetUser()),
			Roles:     u.GetRoles(),
			UpdatedAt: u.GetUpdatedAt(),
		})
	}

	c.JSON(http.StatusOK, errcode.SuccessGin(c, dto.UserList{
		Users:    users,
		Total:    resp.GetTotal(),
		Page:     page,
		PageSize: pageSize,
	}))
}

// GetUserDetail 用户详情
//
//	@Summary		用户详情
//	@Description	查看用户资料、角色与最终权限列表
//	@Tags			管理后台
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid	path		string	true	"用户 UUID"
//	@Success		200		{object}	errcode.Response{data=dto.UserDetail}	"查询成功"
//	@Failure		401		{object}	errcode.Response			"未认证"
//	@Failure		403		{object}	errcode.Response			"无权限"
//	@Failure		404		{object}	errcode.Response			"用户不存在"
//	@Router			/admin/users/{uuid} [get]
func (h *AdminHandler) GetUserDetail(c *gin.Context) {
	uuid := c.Param("uuid")
	resp, err := h.users.Service().GetUserDetail(c.Request.Context(), &userpb.GetUserDetailRequest{Uuid: uuid})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, errcode.SuccessGin(c, dto.UserDetail{
		User: dto.AdminUserItem{
			User:      toUserInfo(resp.GetUser().GetUser()),
			Roles:     resp.GetUser().GetRoles(),
			UpdatedAt: resp.GetUser().GetUpdatedAt(),
		},
		Permissions: resp.GetPermissions(),
	}))
}

// UpdateUserStatus 冻结 / 解冻 / 注销用户
//
//	@Summary		修改用户状态
//	@Description	冻结或解冻用户；状态变更后会清理该用户的权限缓存
//	@Tags			管理后台
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid	path		string						true	"用户 UUID"
//	@Param			body	body		dto.UpdateUserStatusRequest	true	"目标状态"
//	@Success		200		{object}	errcode.Response	"操作成功"
//	@Failure		400		{object}	errcode.Response	"参数校验失败"
//	@Failure		403		{object}	errcode.Response	"无权限"
//	@Router			/admin/users/{uuid}/status [put]
func (h *AdminHandler) UpdateUserStatus(c *gin.Context) {
	var req dto.UpdateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errcode.Fail(errcode.CodeParamInvalid, err.Error()))
		return
	}

	if _, err := h.users.Service().UpdateUserStatus(c.Request.Context(), &userpb.UpdateUserStatusRequest{
		Uuid:   c.Param("uuid"),
		Status: req.Status,
	}); err != nil {
		writeGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, errcode.SuccessGin(c, nil))
}

// AssignRole 为用户分配角色
//
//	@Summary		分配用户角色
//	@Description	为用户分配角色，分配后自动清理该用户的权限缓存
//	@Tags			管理后台
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			uuid	path		string					true	"目标用户 UUID"
//	@Param			body	body		dto.AssignRoleRequest	true	"角色编码"
//	@Success		200		{object}	errcode.Response	"分配成功"
//	@Failure		403		{object}	errcode.Response	"无权限"
//	@Failure		404		{object}	errcode.Response	"角色不存在"
//	@Router			/admin/users/{uuid}/role [put]
func (h *AdminHandler) AssignRole(c *gin.Context) {
	var req dto.AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errcode.Fail(errcode.CodeParamInvalid, err.Error()))
		return
	}

	if _, err := h.users.Service().AssignRole(c.Request.Context(), &userpb.AssignRoleRequest{
		TargetUuid:   c.Param("uuid"),
		RoleCode:     req.RoleCode,
		OperatorUuid: middleware.CurrentUserUUID(c),
	}); err != nil {
		writeGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, errcode.SuccessGin(c, nil))
}

// ListRoles 角色列表
//
//	@Summary		角色列表
//	@Tags			管理后台
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	errcode.Response{data=dto.RoleList}	"查询成功"
//	@Failure		403	{object}	errcode.Response			"无权限"
//	@Router			/admin/roles [get]
func (h *AdminHandler) ListRoles(c *gin.Context) {
	resp, err := h.users.Service().ListRoles(c.Request.Context(), &userpb.ListRolesRequest{})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	roles := make([]dto.RoleInfo, 0, len(resp.GetRoles()))
	for _, r := range resp.GetRoles() {
		roles = append(roles, dto.RoleInfo{
			ID:          r.GetId(),
			Code:        r.GetCode(),
			Name:        r.GetName(),
			Description: r.GetDescription(),
			IsSystem:    r.GetIsSystem(),
			CreatedAt:   r.GetCreatedAt(),
		})
	}

	c.JSON(http.StatusOK, errcode.SuccessGin(c, dto.RoleList{Roles: roles}))
}

// CreateRole 创建角色
//
//	@Summary		创建角色
//	@Tags			管理后台
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.CreateRoleRequest	true	"角色信息"
//	@Success		200		{object}	errcode.Response{data=dto.RoleInfo}	"创建成功"
//	@Failure		403		{object}	errcode.Response			"无权限"
//	@Failure		409		{object}	errcode.Response			"角色编码已存在"
//	@Router			/admin/roles [post]
func (h *AdminHandler) CreateRole(c *gin.Context) {
	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errcode.Fail(errcode.CodeParamInvalid, err.Error()))
		return
	}

	resp, err := h.users.Service().CreateRole(c.Request.Context(), &userpb.CreateRoleRequest{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	r := resp.GetRole()
	c.JSON(http.StatusOK, errcode.SuccessGin(c, dto.RoleInfo{
		ID:          r.GetId(),
		Code:        r.GetCode(),
		Name:        r.GetName(),
		Description: r.GetDescription(),
		IsSystem:    r.GetIsSystem(),
		CreatedAt:   r.GetCreatedAt(),
	}))
}

// UpdateRole 编辑角色
//
//	@Summary		编辑角色
//	@Description	系统内置角色不可修改
//	@Tags			管理后台
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int						true	"角色 ID"
//	@Param			body	body		dto.UpdateRoleRequest	true	"角色信息"
//	@Success		200		{object}	errcode.Response	"更新成功"
//	@Failure		403		{object}	errcode.Response	"无权限或系统角色不可改"
//	@Failure		404		{object}	errcode.Response	"角色不存在"
//	@Router			/admin/roles/{id} [put]
func (h *AdminHandler) UpdateRole(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errcode.Fail(errcode.CodeParamInvalid, err.Error()))
		return
	}

	if _, err := h.users.Service().UpdateRole(c.Request.Context(), &userpb.UpdateRoleRequest{
		Id:          id,
		Name:        req.Name,
		Description: req.Description,
	}); err != nil {
		writeGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, errcode.SuccessGin(c, nil))
}

// DeleteRole 删除角色
//
//	@Summary		删除角色
//	@Description	系统内置角色不可删除；删除后清理相关用户的权限缓存
//	@Tags			管理后台
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"角色 ID"
//	@Success		200	{object}	errcode.Response	"删除成功"
//	@Failure		403	{object}	errcode.Response	"无权限或系统角色不可删"
//	@Failure		404	{object}	errcode.Response	"角色不存在"
//	@Router			/admin/roles/{id} [delete]
func (h *AdminHandler) DeleteRole(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	if _, err := h.users.Service().DeleteRole(c.Request.Context(), &userpb.DeleteRoleRequest{Id: id}); err != nil {
		writeGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, errcode.SuccessGin(c, nil))
}

// ListPermissions 权限列表
//
//	@Summary		权限列表
//	@Tags			管理后台
//	@Produce		json
//	@Security		BearerAuth
//	@Param			resource	query		string	false	"按资源类型过滤，如 user / novel"
//	@Success		200			{object}	errcode.Response{data=dto.PermissionList}	"查询成功"
//	@Failure		403			{object}	errcode.Response			"无权限"
//	@Router			/admin/permissions [get]
func (h *AdminHandler) ListPermissions(c *gin.Context) {
	resp, err := h.users.Service().ListPermissions(c.Request.Context(), &userpb.ListPermissionsRequest{
		Resource: c.Query("resource"),
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	perms := make([]dto.PermissionInfo, 0, len(resp.GetPermissions()))
	for _, p := range resp.GetPermissions() {
		perms = append(perms, dto.PermissionInfo{
			ID:          p.GetId(),
			Code:        p.GetCode(),
			Name:        p.GetName(),
			Resource:    p.GetResource(),
			Action:      p.GetAction(),
			Description: p.GetDescription(),
		})
	}

	c.JSON(http.StatusOK, errcode.SuccessGin(c, dto.PermissionList{Permissions: perms}))
}

// SetRolePermissions 为角色分配权限
//
//	@Summary		为角色分配权限
//	@Description	覆盖式设置角色的权限集合，变更后清理该角色下所有用户的权限缓存
//	@Tags			管理后台
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int								true	"角色 ID"
//	@Param			body	body		dto.SetRolePermissionsRequest	true	"权限编码列表"
//	@Success		200		{object}	errcode.Response	"分配成功"
//	@Failure		403		{object}	errcode.Response	"无权限"
//	@Failure		404		{object}	errcode.Response	"权限编码不存在"
//	@Router			/admin/roles/{id}/permissions [put]
func (h *AdminHandler) SetRolePermissions(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var req dto.SetRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errcode.Fail(errcode.CodeParamInvalid, err.Error()))
		return
	}

	if _, err := h.users.Service().SetRolePermissions(c.Request.Context(), &userpb.SetRolePermissionsRequest{
		RoleId:          id,
		PermissionCodes: req.PermissionCodes,
	}); err != nil {
		writeGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, errcode.SuccessGin(c, nil))
}

// ListAuditLogs 审计日志列表
//
//	@Summary		审计日志列表
//	@Description	按操作人、操作类型、时间范围筛选，分页返回
//	@Tags			管理后台
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int		false	"页码，默认 1"
//	@Param			page_size	query		int		false	"每页条数，默认 20"
//	@Param			user_uuid	query		string	false	"按操作人 UUID 过滤"
//	@Param			action		query		string	false	"按操作类型过滤，如 login / change_password"
//	@Param			from		query		int		false	"起始时间（Unix 秒）"
//	@Param			to			query		int		false	"结束时间（Unix 秒）"
//	@Success		200			{object}	errcode.Response{data=dto.AuditLogList}	"查询成功"
//	@Failure		403			{object}	errcode.Response			"无权限"
//	@Router			/admin/audit-logs [get]
func (h *AdminHandler) ListAuditLogs(c *gin.Context) {
	page, pageSize, _ := parsePage(c)

	resp, err := h.users.Service().ListAuditLogs(c.Request.Context(), &userpb.ListAuditLogsRequest{
		Pagination: &userpb.Pagination{Page: page, PageSize: pageSize},
		UserUuid:   c.Query("user_uuid"),
		Action:     c.Query("action"),
		From:       int64(parseIntDefault(c.Query("from"), 0)),
		To:         int64(parseIntDefault(c.Query("to"), 0)),
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	logs := make([]dto.AuditLogItem, 0, len(resp.GetLogs()))
	for _, l := range resp.GetLogs() {
		logs = append(logs, dto.AuditLogItem{
			ID:         l.GetId(),
			UserUUID:   l.GetUserUuid(),
			Action:     l.GetAction(),
			Resource:   l.GetResource(),
			ResourceID: l.GetResourceId(),
			IP:         l.GetIp(),
			UserAgent:  l.GetUserAgent(),
			Detail:     l.GetDetail(),
			CreatedAt:  l.GetCreatedAt(),
		})
	}

	c.JSON(http.StatusOK, errcode.SuccessGin(c, dto.AuditLogList{
		Logs:     logs,
		Total:    resp.GetTotal(),
		Page:     page,
		PageSize: pageSize,
	}))
}

// parseID 解析并校验路径中的角色 ID，失败时已写入响应并返回 false。
func parseID(c *gin.Context) (int32, bool) {
	id := parseIntDefault(c.Param("id"), 0)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, errcode.ErrorGin(c, errcode.CodeParamInvalid))
		return 0, false
	}
	return int32(id), true
}
