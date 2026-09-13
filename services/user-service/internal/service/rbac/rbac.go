// Package rbac 提供角色 / 权限的读写与缓存能力。
//
// 依赖方向：rbac -> repository（token/repository）。rbac 不反向依赖 auth，
// 因此 auth 包可安全地依赖 rbac（auth -> rbac -> repository），不形成循环。
//
// 角色与权限数据的来源：启动播种（internal/bootstrap）负责保证系统角色、权限定义与
// 角色默认权限映射存在，管理员可在后台调整映射；本包不写入，只做读取与缓存。
//
// super_admin 视为通配：不逐条授予权限，缓存里只存通配标记，CheckPermission 命中即放行。
// 这样新增权限后无需再给系统管理员补授权，也不存在「缓存里缺新权限」的窗口期；
// 对外查询权限列表时再展开成全部权限编码，前端菜单/按钮无需特殊处理。
//
// 权限缓存设计（与需求一致）：
//   - key: perm:user:{userUUID}，value: 逗号分隔的权限编码字符串（super_admin 为 "*"）
//   - TTL: 与 Refresh Token 有效期一致（USER_REFRESH_TTL_SECONDS，默认 30 天）
//   - 登录成功 / 显式查询时写入；权限变更时删除（降级到查库）
package rbac

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/museflow/pkg/logger"
	"github.com/museflow/user-service/internal/model"
	"github.com/museflow/user-service/internal/repository"
)

// ErrRoleNotFound / ErrPermissionNotFound 透传仓储层错误，便于 handler 映射。
var (
	ErrRoleNotFound       = repository.ErrRoleNotFound
	ErrPermissionNotFound = repository.ErrPermissionNotFound
)

// 内置角色编码。
//
// 这三个系统角色由启动播种保证存在（见 internal/bootstrap），权限映射可在后台调整。
const (
	// RoleSuperAdmin 超级管理员：免权限校验（通配），由启动播种授予配置里的管理员账号。
	RoleSuperAdmin = "super_admin"
	// RoleAdmin 管理员。
	RoleAdmin = "admin"
	// RoleUser 默认注册用户，注册与第三方登录自动授予该角色。
	RoleUser = "user"
)

// PermWildcard 权限缓存中的通配标记。
// 拥有 super_admin 角色的用户在缓存里只存该标记：校验直接放行，
// 对外查询权限列表时展开为全部权限编码。
const PermWildcard = "*"

// Service RBAC 业务服务。
type Service struct {
	repo    repository.RBACRepository
	store   repository.TokenStore
	permTTL time.Duration
}

// NewService 构造 RBAC 服务。
func NewService(repo repository.RBACRepository, store repository.TokenStore, permTTL time.Duration) *Service {
	return &Service{repo: repo, store: store, permTTL: permTTL}
}

// GetUserPermissions 返回用户的完整权限编码列表（供前端渲染菜单与按钮）。
// super_admin 在此展开为权限表中的全部权限编码。
func (s *Service) GetUserPermissions(ctx context.Context, userUUID uuid.UUID) ([]string, error) {
	perms, err := s.cachedPermissions(ctx, userUUID)
	if err != nil {
		return nil, err
	}
	if isWildcard(perms) {
		return s.allPermissionCodes(ctx)
	}
	return perms, nil
}

// CheckPermission 校验用户是否拥有指定权限（走缓存优先策略）。
// 拥有 super_admin 的用户命中通配标记，直接放行。
func (s *Service) CheckPermission(ctx context.Context, userUUID uuid.UUID, perm string) (bool, error) {
	perms, err := s.cachedPermissions(ctx, userUUID)
	if err != nil {
		return false, err
	}
	for _, p := range perms {
		if p == perm || p == PermWildcard {
			return true, nil
		}
	}
	return false, nil
}

// cachedPermissions 取用户权限（缓存优先）；super_admin 返回通配标记而非逐条权限。
// 缓存不可用时降级查库，但会留痕——否则缓存故障只表现为「接口变慢」，很难定位。
func (s *Service) cachedPermissions(ctx context.Context, userUUID uuid.UUID) ([]string, error) {
	cached, err := s.store.GetUserPermissions(ctx, userUUID.String())
	if err != nil {
		logger.WarnContext(ctx, "读取用户权限缓存失败，本次降级查库",
			logger.UserUUID(userUUID.String()), logger.Err(err))
	} else if cached != nil {
		return cached, nil
	}

	perms, err := s.resolvePermissions(ctx, userUUID)
	if err != nil {
		return nil, err
	}
	// 回写缓存：失败不阻断主流程（下次仍会查库），但同样需要告警，
	// 否则缓存一直写不进去、每次鉴权都穿透到数据库，只能靠延迟升高才发现
	if werr := s.store.SetUserPermissions(ctx, userUUID.String(), perms, s.permTTL); werr != nil {
		logger.WarnContext(ctx, "回写用户权限缓存失败，后续请求将穿透到数据库",
			logger.UserUUID(userUUID.String()), logger.Err(werr))
	}
	return perms, nil
}

// resolvePermissions 计算用户的权限集合：
// super_admin 记通配（不逐条授权，新增权限无需补授权），其余角色按 role_permission 汇总。
func (s *Service) resolvePermissions(ctx context.Context, userUUID uuid.UUID) ([]string, error) {
	roles, err := s.repo.GetUserRoleCodes(ctx, userUUID)
	if err != nil {
		return nil, err
	}
	if slices.Contains(roles, RoleSuperAdmin) {
		return []string{PermWildcard}, nil
	}
	return s.repo.GetUserPermissionCodes(ctx, userUUID)
}

// allPermissionCodes 返回权限表中的全部权限编码。
func (s *Service) allPermissionCodes(ctx context.Context) ([]string, error) {
	perms, err := s.repo.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(perms))
	for _, p := range perms {
		codes = append(codes, p.Code)
	}
	return codes, nil
}

// isWildcard 判断权限集合是否为通配标记。
func isWildcard(perms []string) bool {
	return len(perms) == 1 && perms[0] == PermWildcard
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
