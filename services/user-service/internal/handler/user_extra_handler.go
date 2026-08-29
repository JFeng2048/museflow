package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	userpb "github.com/museflow/proto/user"
	"github.com/museflow/pkg/logger"
	"github.com/museflow/user-service/internal/model"
	"github.com/museflow/user-service/internal/repository"
	"github.com/museflow/user-service/internal/service/rbac"
)

// ==================== 用户管理 ====================

// UpdateProfile 更新用户个人信息。
func (h *UserHandler) UpdateProfile(ctx context.Context, req *userpb.UpdateProfileRequest) (*userpb.UpdateProfileResponse, error) {
	u, err := h.auth.UpdateProfile(ctx, req.GetUuid(), req.GetNickname(), req.GetAvatarUrl(), req.GetBio())
	if err != nil {
		logger.WarnContext(ctx, "更新用户信息失败", logger.UserUUID(req.GetUuid()), logger.Err(err))
		return nil, mapError(err)
	}
	return &userpb.UpdateProfileResponse{User: toUserInfo(u)}, nil
}

// ChangePassword 修改密码（需验证旧密码）。
func (h *UserHandler) ChangePassword(ctx context.Context, req *userpb.ChangePasswordRequest) (*userpb.ChangePasswordResponse, error) {
	if err := h.auth.ChangePassword(ctx, req.GetUuid(), req.GetOldPassword(), req.GetNewPassword()); err != nil {
		logger.WarnContext(ctx, "修改密码失败", logger.UserUUID(req.GetUuid()), logger.Err(err))
		return nil, mapError(err)
	}
	logger.InfoContext(ctx, "修改密码成功", logger.UserUUID(req.GetUuid()))
	return &userpb.ChangePasswordResponse{Success: true}, nil
}

// GetUserByUUID 根据 UUID 查询用户（供其他服务调用）。
func (h *UserHandler) GetUserByUUID(ctx context.Context, req *userpb.GetUserByUUIDRequest) (*userpb.GetUserByUUIDResponse, error) {
	u, err := h.auth.GetProfile(ctx, req.GetUuid())
	if err != nil {
		return nil, mapError(err)
	}
	return &userpb.GetUserByUUIDResponse{User: toUserInfo(u)}, nil
}

// ==================== 权限 ====================

// GetUserPermissions 获取用户权限列表（供网关调用，缓存优先）。
func (h *UserHandler) GetUserPermissions(ctx context.Context, req *userpb.GetUserPermissionsRequest) (*userpb.GetUserPermissionsResponse, error) {
	perms, err := h.auth.GetPermissions(ctx, req.GetUuid())
	if err != nil {
		return nil, mapError(err)
	}
	return &userpb.GetUserPermissionsResponse{Permissions: perms}, nil
}

// CheckPermission 校验用户是否拥有指定权限。
func (h *UserHandler) CheckPermission(ctx context.Context, req *userpb.CheckPermissionRequest) (*userpb.CheckPermissionResponse, error) {
	allowed, err := h.auth.CheckPermission(ctx, req.GetUuid(), req.GetPermission())
	if err != nil {
		return nil, mapError(err)
	}
	return &userpb.CheckPermissionResponse{Allowed: allowed}, nil
}

// ClearUserCache 清理用户权限缓存。
func (h *UserHandler) ClearUserCache(ctx context.Context, req *userpb.ClearUserCacheRequest) (*userpb.ClearUserCacheResponse, error) {
	if err := h.auth.ClearUserCache(ctx, req.GetUuid()); err != nil {
		return nil, mapError(err)
	}
	return &userpb.ClearUserCacheResponse{Success: true}, nil
}

// ==================== 会话管理 ====================

// ListSessions 查看用户活跃会话列表。
func (h *UserHandler) ListSessions(ctx context.Context, req *userpb.ListSessionsRequest) (*userpb.ListSessionsResponse, error) {
	metas, err := h.auth.ListSessions(ctx, req.GetUuid())
	if err != nil {
		return nil, mapError(err)
	}
	sessions := make([]*userpb.SessionInfo, 0, len(metas))
	for _, m := range metas {
		sessions = append(sessions, &userpb.SessionInfo{
			TokenId:         m.TokenID,
			DeviceId:        m.DeviceID,
			DeviceName:      m.DeviceName,
			LoginTime:       m.LoginTime.Unix(),
			LastRefreshTime: m.LastRefreshTime.Unix(),
		})
	}
	return &userpb.ListSessionsResponse{Sessions: sessions}, nil
}

// RevokeSession 强制下线指定会话。
func (h *UserHandler) RevokeSession(ctx context.Context, req *userpb.RevokeSessionRequest) (*userpb.RevokeSessionResponse, error) {
	if err := h.auth.RevokeSession(ctx, req.GetUuid(), req.GetTokenId()); err != nil {
		return nil, mapError(err)
	}
	return &userpb.RevokeSessionResponse{Success: true}, nil
}

// ==================== 管理后台 ====================

// ListUsers 用户列表（分页、筛选、排序）。
func (h *UserHandler) ListUsers(ctx context.Context, req *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	page, pageSize := normalizePage(req.GetPagination())
	offset := (page - 1) * pageSize

	items, total, err := h.admin.ListUsers(ctx, req.GetKeyword(), int16(req.GetStatus()), req.GetOrderBy(), req.GetDesc(), offset, pageSize)
	if err != nil {
		return nil, mapError(err)
	}

	users := make([]*userpb.AdminUserItem, 0, len(items))
	for _, it := range items {
		users = append(users, &userpb.AdminUserItem{
			User:      toUserInfo(it.User),
			Roles:     it.Roles,
			UpdatedAt: it.User.UpdatedAt.Unix(),
		})
	}
	return &userpb.ListUsersResponse{Users: users, Total: total}, nil
}

// GetUserDetail 用户详情（含角色与权限）。
func (h *UserHandler) GetUserDetail(ctx context.Context, req *userpb.GetUserDetailRequest) (*userpb.GetUserDetailResponse, error) {
	item, perms, err := h.admin.GetUserDetail(ctx, req.GetUuid())
	if err != nil {
		return nil, mapError(err)
	}
	return &userpb.GetUserDetailResponse{
		User: &userpb.AdminUserItem{
			User:      toUserInfo(item.User),
			Roles:     item.Roles,
			UpdatedAt: item.User.UpdatedAt.Unix(),
		},
		Permissions: perms,
	}, nil
}

// UpdateUserStatus 冻结 / 解冻 / 注销用户。
func (h *UserHandler) UpdateUserStatus(ctx context.Context, req *userpb.UpdateUserStatusRequest) (*userpb.UpdateUserStatusResponse, error) {
	if err := h.admin.UpdateUserStatus(ctx, req.GetUuid(), int16(req.GetStatus())); err != nil {
		return nil, mapError(err)
	}
	return &userpb.UpdateUserStatusResponse{Success: true}, nil
}

// AssignRole 为用户分配角色。
func (h *UserHandler) AssignRole(ctx context.Context, req *userpb.AssignRoleRequest) (*userpb.AssignRoleResponse, error) {
	if err := h.admin.AssignRole(ctx, req.GetTargetUuid(), req.GetRoleCode(), req.GetOperatorUuid()); err != nil {
		return nil, mapAdminError(err)
	}
	return &userpb.AssignRoleResponse{Success: true}, nil
}

// ListRoles 角色列表。
func (h *UserHandler) ListRoles(ctx context.Context, req *userpb.ListRolesRequest) (*userpb.ListRolesResponse, error) {
	roles, err := h.admin.ListRoles(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*userpb.RoleInfo, 0, len(roles))
	for _, r := range roles {
		if req.GetOnlySystem() && !r.IsSystem {
			continue
		}
		out = append(out, toRoleInfo(r))
	}
	return &userpb.ListRolesResponse{Roles: out}, nil
}

// CreateRole 创建角色。
func (h *UserHandler) CreateRole(ctx context.Context, req *userpb.CreateRoleRequest) (*userpb.CreateRoleResponse, error) {
	role, err := h.admin.CreateRole(ctx, req.GetCode(), req.GetName(), req.GetDescription())
	if err != nil {
		return nil, mapAdminError(err)
	}
	return &userpb.CreateRoleResponse{Role: toRoleInfo(*role)}, nil
}

// UpdateRole 编辑角色。
func (h *UserHandler) UpdateRole(ctx context.Context, req *userpb.UpdateRoleRequest) (*userpb.UpdateRoleResponse, error) {
	if err := h.admin.UpdateRole(ctx, int16(req.GetId()), req.GetName(), req.GetDescription()); err != nil {
		return nil, mapAdminError(err)
	}
	return &userpb.UpdateRoleResponse{Success: true}, nil
}

// DeleteRole 删除角色。
func (h *UserHandler) DeleteRole(ctx context.Context, req *userpb.DeleteRoleRequest) (*userpb.DeleteRoleResponse, error) {
	if err := h.admin.DeleteRole(ctx, int16(req.GetId())); err != nil {
		return nil, mapAdminError(err)
	}
	return &userpb.DeleteRoleResponse{Success: true}, nil
}

// ListPermissions 权限列表。
func (h *UserHandler) ListPermissions(ctx context.Context, req *userpb.ListPermissionsRequest) (*userpb.ListPermissionsResponse, error) {
	perms, err := h.admin.ListPermissions(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*userpb.PermissionInfo, 0, len(perms))
	for _, p := range perms {
		if req.GetResource() != "" && p.Resource != req.GetResource() {
			continue
		}
		out = append(out, toPermissionInfo(p))
	}
	return &userpb.ListPermissionsResponse{Permissions: out}, nil
}

// SetRolePermissions 为角色分配权限。
func (h *UserHandler) SetRolePermissions(ctx context.Context, req *userpb.SetRolePermissionsRequest) (*userpb.SetRolePermissionsResponse, error) {
	if err := h.admin.SetRolePermissions(ctx, int16(req.GetRoleId()), req.GetPermissionCodes()); err != nil {
		return nil, mapAdminError(err)
	}
	return &userpb.SetRolePermissionsResponse{Success: true}, nil
}

// ==================== 辅助函数 ====================

// mapAdminError 将 RBAC / 管理后台业务错误映射为 gRPC status。
func mapAdminError(err error) error {
	if err == nil {
		return nil
	}
	switch err {
	case rbac.ErrRoleNotFound, repository.ErrUserNotFound:
		return status.Error(codes.NotFound, err.Error())
	case rbac.ErrPermissionNotFound:
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.FailedPrecondition, err.Error())
	}
}

func normalizePage(p *userpb.Pagination) (int, int) {
	page := int(p.GetPage())
	pageSize := int(p.GetPageSize())
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func toRoleInfo(r model.Role) *userpb.RoleInfo {
	return &userpb.RoleInfo{
		Id:          int32(r.ID),
		Code:        r.Code,
		Name:        r.Name,
		Description: r.Description,
		IsSystem:    r.IsSystem,
		CreatedAt:   r.CreatedAt.Unix(),
	}
}

func toPermissionInfo(p model.Permission) *userpb.PermissionInfo {
	return &userpb.PermissionInfo{
		Id:          int32(p.ID),
		Code:        p.Code,
		Name:        p.Name,
		Resource:    p.Resource,
		Action:      p.Action,
		Description: p.Description,
	}
}
