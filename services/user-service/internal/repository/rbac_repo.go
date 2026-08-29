package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/museflow/user-service/internal/model"
)

// ErrRoleNotFound 角色不存在。
var ErrRoleNotFound = errors.New("角色不存在")
// ErrPermissionNotFound 权限不存在。
var ErrPermissionNotFound = errors.New("权限不存在")

// RBACRepository 角色与权限数据访问接口。
type RBACRepository interface {
	// GetUserPermissionCodes 返回用户拥有的全部权限编码（经 user_roles -> role_permissions -> permissions）。
	GetUserPermissionCodes(ctx context.Context, userUUID uuid.UUID) ([]string, error)
	// AssignRole 为用户分配角色（按角色 code），重复分配幂等。
	AssignRole(ctx context.Context, userUUID uuid.UUID, roleCode string, grantedBy uuid.UUID) error
	// ListRoles 列出全部角色。
	ListRoles(ctx context.Context) ([]model.Role, error)
	// ListPermissions 列出全部权限。
	ListPermissions(ctx context.Context) ([]model.Permission, error)
	// GetRolePermissions 返回某角色拥有的权限编码列表。
	GetRolePermissions(ctx context.Context, roleID int16) ([]string, error)
	// SetRolePermissions 覆盖设置某角色的权限（先删后插）。
	SetRolePermissions(ctx context.Context, roleID int16, permCodes []string) error
	// CreatePermission 创建权限（种子数据使用）。
	CreatePermission(ctx context.Context, perm *model.Permission) error
	// CreateRole 创建角色。
	CreateRole(ctx context.Context, role *model.Role) error
	// UpdateRole 更新角色基本信息（名称/描述），系统角色不可改 code。
	UpdateRole(ctx context.Context, roleID int16, name, description string) error
	// DeleteRole 删除角色，系统角色不可删。
	DeleteRole(ctx context.Context, roleID int16) error
	// ListRoleUserUUIDs 返回拥有指定角色的全部用户 UUID（用于权限变更后批量清缓存）。
	ListRoleUserUUIDs(ctx context.Context, roleID int16) ([]uuid.UUID, error)
}

type rbacRepository struct {
	db *gorm.DB
}

// NewRBACRepository 构造 RBAC 仓储。
func NewRBACRepository(db *gorm.DB) RBACRepository {
	return &rbacRepository{db: db}
}

// GetUserPermissionCodes 通过三表连接汇总用户权限编码。
func (r *rbacRepository) GetUserPermissionCodes(ctx context.Context, userUUID uuid.UUID) ([]string, error) {
	var codes []string
	err := r.db.WithContext(ctx).
		Table("user_svc.user_roles ur").
		Select("p.code").
		Joins("JOIN user_svc.role_permissions rp ON rp.role_id = ur.role_id").
		Joins("JOIN user_svc.permissions p ON p.id = rp.permission_id").
		Where("ur.user_uuid = ?", userUUID).
		Pluck("p.code", &codes).Error
	if err != nil {
		return nil, err
	}
	return codes, nil
}

// AssignRole 为用户分配角色（按角色 code 查找角色 ID 后写入 user_roles）。
func (r *rbacRepository) AssignRole(ctx context.Context, userUUID uuid.UUID, roleCode string, grantedBy uuid.UUID) error {
	var role model.Role
	if err := r.db.WithContext(ctx).Where("code = ?", roleCode).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRoleNotFound
		}
		return err
	}

	ur := model.UserRole{
		UserUUID: userUUID,
		RoleID:   role.ID,
		GrantedBy: grantedBy,
		GrantedAt: time.Now(),
	}
	// 幂等：冲突则忽略
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_uuid"}, {Name: "role_id"}},
			DoNothing: true,
		}).
		Create(&ur).Error
	return err
}

// ListRoles 列出全部角色。
func (r *rbacRepository) ListRoles(ctx context.Context) ([]model.Role, error) {
	var roles []model.Role
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// ListPermissions 列出全部权限。
func (r *rbacRepository) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	var perms []model.Permission
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

// GetRolePermissions 返回某角色拥有的权限编码列表。
func (r *rbacRepository) GetRolePermissions(ctx context.Context, roleID int16) ([]string, error) {
	var codes []string
	err := r.db.WithContext(ctx).
		Table("user_svc.role_permissions rp").
		Select("p.code").
		Joins("JOIN user_svc.permissions p ON p.id = rp.permission_id").
		Where("rp.role_id = ?", roleID).
		Pluck("p.code", &codes).Error
	if err != nil {
		return nil, err
	}
	return codes, nil
}

// SetRolePermissions 覆盖设置某角色的权限（先删除旧关联，再批量插入）。
func (r *rbacRepository) SetRolePermissions(ctx context.Context, roleID int16, permCodes []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).Delete(&model.RolePermission{}).Error; err != nil {
			return err
		}
		if len(permCodes) == 0 {
			return nil
		}
		// 按 code 查 permission id
		var perms []model.Permission
		if err := tx.Where("code IN ?", permCodes).Find(&perms).Error; err != nil {
			return err
		}
		if len(perms) != len(permCodes) {
			return ErrPermissionNotFound
		}
		rps := make([]model.RolePermission, 0, len(perms))
		for _, p := range perms {
			rps = append(rps, model.RolePermission{RoleID: roleID, PermissionID: p.ID})
		}
		return tx.Create(&rps).Error
	})
}

// CreatePermission 创建权限。
func (r *rbacRepository) CreatePermission(ctx context.Context, perm *model.Permission) error {
	return r.db.WithContext(ctx).Create(perm).Error
}

// CreateRole 创建角色。
func (r *rbacRepository) CreateRole(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

// UpdateRole 更新角色名称与描述。
func (r *rbacRepository) UpdateRole(ctx context.Context, roleID int16, name, description string) error {
	res := r.db.WithContext(ctx).
		Model(&model.Role{}).
		Where("id = ? AND is_system = ?", roleID, false).
		Updates(map[string]any{"name": name, "description": description})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		// 可能是系统角色（不允许改）或不存在
		var cnt int64
		r.db.WithContext(ctx).Model(&model.Role{}).Where("id = ?", roleID).Count(&cnt)
		if cnt == 0 {
			return ErrRoleNotFound
		}
		return errors.New("系统角色不可修改")
	}
	return nil
}

// DeleteRole 删除角色（系统角色不可删）。
func (r *rbacRepository) DeleteRole(ctx context.Context, roleID int16) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND is_system = ?", roleID, false).
		Delete(&model.Role{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		var cnt int64
		r.db.WithContext(ctx).Model(&model.Role{}).Where("id = ?", roleID).Count(&cnt)
		if cnt == 0 {
			return ErrRoleNotFound
		}
		return errors.New("系统角色不可删除")
	}
	// 级联清理角色-权限与用户-角色关联
	r.db.WithContext(ctx).Where("role_id = ?", roleID).Delete(&model.RolePermission{})
	r.db.WithContext(ctx).Where("role_id = ?", roleID).Delete(&model.UserRole{})
	return nil
}

// ListRoleUserUUIDs 返回拥有指定角色的全部用户 UUID。
func (r *rbacRepository) ListRoleUserUUIDs(ctx context.Context, roleID int16) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).
		Model(&model.UserRole{}).
		Where("role_id = ?", roleID).
		Pluck("user_uuid", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}
