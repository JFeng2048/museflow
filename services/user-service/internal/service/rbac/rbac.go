// Package rbac 提供角色 / 权限的读写与缓存能力。
//
// 依赖方向：rbac -> repository（token/repository）。rbac 不反向依赖 auth，
// 因此 auth 包可安全地依赖 rbac（auth -> rbac -> repository），不形成循环。
//
// 权限缓存设计（与需求一致）：
//   - key: perm:user:{userUUID}，value: 逗号分隔的权限编码字符串
//   - TTL: 7 天（与 Refresh Token 有效期一致）
//   - 登录成功 / 显式查询时写入；权限变更时删除（降级到查库）
package rbac

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/museflow/user-service/internal/model"
	"github.com/museflow/user-service/internal/repository"
)

// ErrRoleNotFound / ErrPermissionNotFound 透传仓储层错误，便于 handler 映射。
var (
	ErrRoleNotFound      = repository.ErrRoleNotFound
	ErrPermissionNotFound = repository.ErrPermissionNotFound
)

// Service RBAC 业务服务。
type Service struct {
	repo     repository.RBACRepository
	store    repository.TokenStore
	permTTL  time.Duration
}

// NewService 构造 RBAC 服务。
func NewService(repo repository.RBACRepository, store repository.TokenStore, permTTL time.Duration) *Service {
	return &Service{repo: repo, store: store, permTTL: permTTL}
}

// GetUserPermissions 返回用户权限编码列表。
// 优先读 Redis 缓存；未命中则查库并回填缓存（缓存不可用时不报错，直接返回库结果）。
func (s *Service) GetUserPermissions(ctx context.Context, userUUID uuid.UUID) ([]string, error) {
	cached, err := s.store.GetUserPermissions(ctx, userUUID.String())
	if err == nil && cached != nil {
		return cached, nil
	}
	// 未命中或 Redis 故障：降级查库
	perms, err := s.repo.GetUserPermissionCodes(ctx, userUUID)
	if err != nil {
		return nil, err
	}
	// 回写缓存（失败仅告警，不阻断）
	if werr := s.store.SetUserPermissions(ctx, userUUID.String(), perms, s.permTTL); werr != nil {
		// TODO: logger.Warn("回写用户权限缓存失败", ...) 待 logger 注入
		_ = werr
	}
	return perms, nil
}

// CheckPermission 校验用户是否拥有指定权限（走缓存优先策略）。
func (s *Service) CheckPermission(ctx context.Context, userUUID uuid.UUID, perm string) (bool, error) {
	perms, err := s.GetUserPermissions(ctx, userUUID)
	if err != nil {
		return false, err
	}
	for _, p := range perms {
		if p == perm {
			return true, nil
		}
	}
	return false, nil
}

// ClearUserCache 删除用户权限缓存（权限变更后调用）。
// 缓存删除失败不阻断主流程，由调用方决定是否记录告警。
func (s *Service) ClearUserCache(ctx context.Context, userUUID uuid.UUID) error {
	return s.store.ClearUserPermissions(ctx, userUUID.String())
}

// GetUserRoleCodes 返回用户拥有的角色编码列表。
func (s *Service) GetUserRoleCodes(ctx context.Context, userUUID uuid.UUID) ([]string, error) {
	return s.repo.GetUserRoleCodes(ctx, userUUID)
}

// RemoveUserRole 移除用户角色并清理其权限缓存。
func (s *Service) RemoveUserRole(ctx context.Context, userUUID uuid.UUID, roleID int16) error {
	if err := s.repo.RemoveUserRole(ctx, userUUID, roleID); err != nil {
		return err
	}
	return s.ClearUserCache(ctx, userUUID)
}

// AssignRole 为用户分配角色，并清理其权限缓存（下次访问重新加载）。
func (s *Service) AssignRole(ctx context.Context, userUUID uuid.UUID, roleCode string, grantedBy uuid.UUID) error {
	if err := s.repo.AssignRole(ctx, userUUID, roleCode, grantedBy); err != nil {
		return err
	}
	return s.ClearUserCache(ctx, userUUID)
}

// SetRolePermissions 覆盖角色权限，并清理该角色下所有用户的权限缓存。
func (s *Service) SetRolePermissions(ctx context.Context, roleID int16, permCodes []string) error {
	if err := s.repo.SetRolePermissions(ctx, roleID, permCodes); err != nil {
		return err
	}
	return s.clearRoleUserCaches(ctx, roleID)
}

// ListRoles 列出全部角色。
func (s *Service) ListRoles(ctx context.Context) ([]model.Role, error) {
	return s.repo.ListRoles(ctx)
}

// ListPermissions 列出全部权限。
func (s *Service) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	return s.repo.ListPermissions(ctx)
}

// GetRolePermissions 返回角色权限编码列表。
func (s *Service) GetRolePermissions(ctx context.Context, roleID int16) ([]string, error) {
	return s.repo.GetRolePermissions(ctx, roleID)
}

// CreateRole 创建角色。
func (s *Service) CreateRole(ctx context.Context, role *model.Role) error {
	return s.repo.CreateRole(ctx, role)
}

// UpdateRole 更新角色。
func (s *Service) UpdateRole(ctx context.Context, roleID int16, name, description string) error {
	return s.repo.UpdateRole(ctx, roleID, name, description)
}

// DeleteRole 删除角色（并清理相关用户缓存）。
func (s *Service) DeleteRole(ctx context.Context, roleID int16) error {
	if err := s.repo.DeleteRole(ctx, roleID); err != nil {
		return err
	}
	return s.clearRoleUserCaches(ctx, roleID)
}

// clearRoleUserCaches 清理某角色下所有用户的权限缓存。
func (s *Service) clearRoleUserCaches(ctx context.Context, roleID int16) error {
	ids, err := s.repo.ListRoleUserUUIDs(ctx, roleID)
	if err != nil {
		return err
	}
	var firstErr error
	for _, id := range ids {
		if err := s.store.ClearUserPermissions(ctx, id.String()); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	// 缓存清理失败不阻断主流程，返回首个错误供调用方记录告警
	return firstErr
}

// ErrNotFound 通用未找到。
var ErrNotFound = errors.New("资源不存在")
