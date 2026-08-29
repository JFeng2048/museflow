// Package admin 实现管理后台的业务逻辑：用户列表 / 详情 / 状态管理、
// 角色与权限管理。
//
// 依赖方向：admin -> repository + rbac（不反向依赖 auth / handler），
// 避免出现循环依赖。权限变更操作会经由 rbac 清理相关用户的 Redis 缓存。
package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/museflow/user-service/internal/model"
	"github.com/museflow/user-service/internal/repository"
	"github.com/museflow/user-service/internal/service/audit"
	"github.com/museflow/user-service/internal/service/rbac"
)

// Service 管理后台服务。
type Service struct {
	users repository.UserRepository
	rbac  *rbac.Service
	audit *audit.Service
}

// NewService 构造管理后台服务。
func NewService(users repository.UserRepository, rbacSvc *rbac.Service, auditSvc *audit.Service) *Service {
	return &Service{users: users, rbac: rbacSvc, audit: auditSvc}
}

// UserItem 管理后台用户列表项（用户 + 角色编码）。
type UserItem struct {
	User  *model.User
	Roles []string
}

// ListUsers 分页查询用户（含角色），支持关键字与状态过滤。
func (s *Service) ListUsers(ctx context.Context, keyword string, status int16, orderBy string, desc bool, offset, limit int) ([]UserItem, int64, error) {
	users, err := s.users.ListUsers(ctx, keyword, status, orderBy, desc, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("查询用户列表失败: %w", err)
	}
	total, err := s.users.CountUsers(ctx, keyword, status)
	if err != nil {
		return nil, 0, fmt.Errorf("统计用户数量失败: %w", err)
	}

	items := make([]UserItem, 0, len(users))
	for i := range users {
		roles, rerr := s.rbac.GetUserRoleCodes(ctx, users[i].UUID)
		if rerr != nil {
			// 角色查询失败不阻断列表返回，角色留空
			roles = nil
		}
		items = append(items, UserItem{User: &users[i], Roles: roles})
	}
	return items, total, nil
}

// GetUserDetail 查询用户详情（含角色与最终权限）。
func (s *Service) GetUserDetail(ctx context.Context, userUUID string) (*UserItem, []string, error) {
	id, err := uuid.Parse(userUUID)
	if err != nil {
		return nil, nil, repository.ErrUserNotFound
	}
	u, err := s.users.FindByUUID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	roles, err := s.rbac.GetUserRoleCodes(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	perms, err := s.rbac.GetUserPermissions(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return &UserItem{User: u, Roles: roles}, perms, nil
}

// UpdateUserStatus 更新用户状态（冻结 / 解冻 / 注销）。
func (s *Service) UpdateUserStatus(ctx context.Context, userUUID string, status int16) error {
	id, err := uuid.Parse(userUUID)
	if err != nil {
		return repository.ErrUserNotFound
	}
	if err := s.users.UpdateStatus(ctx, id, status); err != nil {
		return fmt.Errorf("更新用户状态失败: %w", err)
	}
	s.audit.Record(ctx, audit.Entry{
		UserUUID:   userUUID,
		Action:     model.AuditActionUpdateStat,
		Resource:   model.AuditResourceUser,
		ResourceID: userUUID,
		Detail:     map[string]int16{"status": status},
	})
	// 状态变更（如冻结）后清理权限缓存，确保下次校验按最新状态生效
	return s.rbac.ClearUserCache(ctx, id)
}

// AssignRole 为用户分配角色（分配后清理其权限缓存）。
func (s *Service) AssignRole(ctx context.Context, targetUUID, roleCode, operatorUUID string) error {
	target, err := uuid.Parse(targetUUID)
	if err != nil {
		return repository.ErrUserNotFound
	}
	operator, err := parseOptionalUUID(operatorUUID)
	if err != nil {
		return err
	}
	if err := s.rbac.AssignRole(ctx, target, roleCode, operator); err != nil {
		return err
	}
	s.audit.Record(ctx, audit.Entry{
		UserUUID:   operatorUUID,
		Action:     model.AuditActionAssignRole,
		Resource:   model.AuditResourceRole,
		ResourceID: roleCode,
		Detail:     map[string]string{"target_uuid": targetUUID, "role_code": roleCode},
	})
	return nil
}

// ListRoles 列出角色。
func (s *Service) ListRoles(ctx context.Context) ([]model.Role, error) {
	return s.rbac.ListRoles(ctx)
}

// CreateRole 创建角色。
func (s *Service) CreateRole(ctx context.Context, code, name, description string) (*model.Role, error) {
	role := &model.Role{Code: code, Name: name, Description: description}
	if err := s.rbac.CreateRole(ctx, role); err != nil {
		return nil, err
	}
	s.audit.Record(ctx, audit.Entry{
		Action:     model.AuditActionCreateRole,
		Resource:   model.AuditResourceRole,
		ResourceID: code,
		Detail:     map[string]string{"name": name},
	})
	return role, nil
}

// UpdateRole 编辑角色（系统角色不可改）。
func (s *Service) UpdateRole(ctx context.Context, roleID int16, name, description string) error {
	if err := s.rbac.UpdateRole(ctx, roleID, name, description); err != nil {
		return err
	}
	s.audit.Record(ctx, audit.Entry{
		Action:     model.AuditActionUpdateRole,
		Resource:   model.AuditResourceRole,
		ResourceID: fmt.Sprintf("%d", roleID),
		Detail:     map[string]string{"name": name},
	})
	return nil
}

// DeleteRole 删除角色（系统角色不可删，并清理相关用户缓存）。
func (s *Service) DeleteRole(ctx context.Context, roleID int16) error {
	if err := s.rbac.DeleteRole(ctx, roleID); err != nil {
		return err
	}
	s.audit.Record(ctx, audit.Entry{
		Action:     model.AuditActionDeleteRole,
		Resource:   model.AuditResourceRole,
		ResourceID: fmt.Sprintf("%d", roleID),
	})
	return nil
}

// ListPermissions 列出权限。
func (s *Service) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	return s.rbac.ListPermissions(ctx)
}

// SetRolePermissions 为角色分配权限（清理相关用户缓存）。
func (s *Service) SetRolePermissions(ctx context.Context, roleID int16, permCodes []string) error {
	if err := s.rbac.SetRolePermissions(ctx, roleID, permCodes); err != nil {
		return err
	}
	s.audit.Record(ctx, audit.Entry{
		Action:     model.AuditActionSetRolePerm,
		Resource:   model.AuditResourcePerm,
		ResourceID: fmt.Sprintf("%d", roleID),
		Detail:     map[string]any{"role_id": roleID, "permissions": permCodes},
	})
	return nil
}

// ListAuditLogs 分页查询审计日志（按用户 / 操作类型 / 时间范围筛选）。
func (s *Service) ListAuditLogs(ctx context.Context, userUUID, action string, from, to *time.Time, offset, limit int) ([]model.AuditLog, int64, error) {
	return s.audit.List(ctx, userUUID, action, from, to, offset, limit)
}

func parseOptionalUUID(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(s)
}
