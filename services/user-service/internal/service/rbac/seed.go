package rbac

import (
	"context"

	"github.com/museflow/user-service/internal/model"
)

// 预置角色编码。
const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
	RoleUser       = "user"
	RoleEditor     = "editor"
	RoleViewer     = "viewer"
)

// 预置权限编码（与 database/user_svc.sql 中的种子数据一致）。
const (
	PermUserRead   = "user:read"
	PermUserWrite  = "user:write"
	PermUserManage = "user:manage"

	PermNovelRead    = "novel:read"
	PermNovelWrite   = "novel:write"
	PermNovelPublish = "novel:publish"

	PermMaterialRead  = "material:read"
	PermMaterialWrite = "material:write"

	PermPublishRead  = "publish:read"
	PermPublishWrite = "publish:write"

	PermSystemRead  = "system:read"
	PermSystemWrite = "system:write"

	PermHotspotRead  = "hotspot:read"
	PermHotspotWrite = "hotspot:write"
)

// defaultRoles 预置角色（is_system=true 不可改/删）。
var defaultRoles = []model.Role{
	{Code: RoleSuperAdmin, Name: "超级管理员", Description: "拥有系统全部权限", IsSystem: true},
	{Code: RoleAdmin, Name: "管理员", Description: "用户与内容管理权限", IsSystem: true},
	{Code: RoleUser, Name: "普通用户", Description: "默认注册用户", IsSystem: true},
	{Code: RoleEditor, Name: "编辑", Description: "内容创作与发布权限", IsSystem: true},
	{Code: RoleViewer, Name: "查看者", Description: "只读权限", IsSystem: true},
}

// defaultPermissions 预置权限。
var defaultPermissions = []model.Permission{
	{Code: PermUserRead, Name: "用户查看", Resource: "user", Action: "read"},
	{Code: PermUserWrite, Name: "用户编辑", Resource: "user", Action: "write"},
	{Code: PermUserManage, Name: "用户管理", Resource: "user", Action: "manage"},

	{Code: PermNovelRead, Name: "作品查看", Resource: "novel", Action: "read"},
	{Code: PermNovelWrite, Name: "作品编辑", Resource: "novel", Action: "write"},
	{Code: PermNovelPublish, Name: "作品发布", Resource: "novel", Action: "publish"},

	{Code: PermMaterialRead, Name: "素材查看", Resource: "material", Action: "read"},
	{Code: PermMaterialWrite, Name: "素材编辑", Resource: "material", Action: "write"},

	{Code: PermPublishRead, Name: "发布查看", Resource: "publish", Action: "read"},
	{Code: PermPublishWrite, Name: "发布编辑", Resource: "publish", Action: "write"},

	{Code: PermSystemRead, Name: "系统查看", Resource: "system", Action: "read"},
	{Code: PermSystemWrite, Name: "系统管理", Resource: "system", Action: "write"},

	{Code: PermHotspotRead, Name: "热点查看", Resource: "hotspot", Action: "read"},
	{Code: PermHotspotWrite, Name: "热点编辑", Resource: "hotspot", Action: "write"},
}

// rolePermissionMap 角色 -> 权限编码 的初始映射。
var rolePermissionMap = map[string][]string{
	RoleSuperAdmin: {
		PermUserRead, PermUserWrite, PermUserManage,
		PermNovelRead, PermNovelWrite, PermNovelPublish,
		PermMaterialRead, PermMaterialWrite,
		PermPublishRead, PermPublishWrite,
		PermSystemRead, PermSystemWrite,
		PermHotspotRead, PermHotspotWrite,
	},
	RoleAdmin: {
		PermUserRead, PermUserWrite, PermUserManage,
		PermNovelRead, PermNovelWrite, PermNovelPublish,
		PermMaterialRead, PermMaterialWrite,
		PermPublishRead, PermPublishWrite,
		PermHotspotRead, PermHotspotWrite,
	},
	RoleUser: {
		PermUserRead,
		PermNovelRead, PermNovelWrite, PermNovelPublish,
		PermMaterialRead, PermMaterialWrite,
		PermPublishRead, PermPublishWrite,
		PermHotspotRead,
	},
	RoleEditor: {
		PermUserRead,
		PermNovelRead, PermNovelWrite, PermNovelPublish,
		PermMaterialRead, PermMaterialWrite,
		PermPublishRead, PermPublishWrite,
		PermHotspotRead, PermHotspotWrite,
	},
	RoleViewer: {
		PermUserRead,
		PermNovelRead,
		PermMaterialRead,
		PermPublishRead,
		PermHotspotRead,
	},
}

// EnsureSeeded 在库为空（无角色）时插入预置角色、权限与角色-权限映射。
// 幂等：已存在角色则跳过。
func (s *Service) EnsureSeeded(ctx context.Context) error {
	existing, err := s.repo.ListRoles(ctx)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}
	// 1. 插入权限
	for i := range defaultPermissions {
		if err := s.repo.CreatePermission(ctx, &defaultPermissions[i]); err != nil {
			return err
		}
	}
	// 2. 插入角色
	for i := range defaultRoles {
		if err := s.repo.CreateRole(ctx, &defaultRoles[i]); err != nil {
			return err
		}
	}
	// 3. 建立角色-权限关联
	for roleCode, perms := range rolePermissionMap {
		role, err := s.findRoleByCode(ctx, roleCode)
		if err != nil {
			return err
		}
		if err := s.repo.SetRolePermissions(ctx, role.ID, perms); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) findRoleByCode(ctx context.Context, code string) (*model.Role, error) {
	roles, err := s.repo.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	for i := range roles {
		if roles[i].Code == code {
			return &roles[i], nil
		}
	}
	return nil, ErrRoleNotFound
}
