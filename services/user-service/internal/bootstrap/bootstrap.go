// Package bootstrap 负责服务启动时的最小必要播种（seed）。
//
// 为什么播种放在代码而不是 SQL：schema 仍由 database/user_svc.sql 维护，但管理员
// 账号的密码属于环境配置（.env / k8s Secret），写死在 SQL 里既不方便也不安全。
// 放在启动流程里，则「换一个环境、填一次配置」就能得到可登录的管理员，且天然幂等。
//
// 播种范围只覆盖「能登录进去」所需的部分：
//   - 系统角色（super_admin / admin / user）缺失则补建；权限明细仍由 SQL 维护，
//     这里只保证角色本身存在，不去创建权限或角色-权限关系；
//   - 管理员账号缺失则创建并授予 super_admin。
//
// 已存在账号的密码不会被覆盖，除非显式开启 ResetPassword（忘记密码时的应急开关）。
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/museflow/pkg/logger"
	"github.com/museflow/user-service/internal/model"
	"github.com/museflow/user-service/internal/repository"
	"github.com/museflow/user-service/internal/service/rbac"
)

// maxPasswordBytes 与 auth 服务一致：bcrypt 只处理前 72 字节，超出会被静默截断。
const maxPasswordBytes = 72

// DefaultAdminNickname 管理员账号默认昵称（数据库 nickname 字段非空）。
// 同时作为 USER_ADMIN_NICKNAME 未配置时的取值。
const DefaultAdminNickname = "系统管理员"

// UserStore 播种所需的用户读写能力，由 repository.UserRepository 实现。
// 这里只声明用到的少数方法，便于测试替换。
type UserStore interface {
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	Create(ctx context.Context, u *model.User) error
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error
}

// RBACStore 播种所需的角色与权限读写能力，由 repository.RBACRepository 实现。
type RBACStore interface {
	ListRoles(ctx context.Context) ([]model.Role, error)
	CreateRole(ctx context.Context, role *model.Role) error
	ListPermissions(ctx context.Context) ([]model.Permission, error)
	CreatePermissions(ctx context.Context, perms []model.Permission) error
	GetRolePermissions(ctx context.Context, roleID int16) ([]string, error)
	SetRolePermissions(ctx context.Context, roleID int16, permCodes []string) error
	GetUserRoleCodes(ctx context.Context, userUUID uuid.UUID) ([]string, error)
	AssignRole(ctx context.Context, userUUID uuid.UUID, roleCode string, grantedBy uuid.UUID) error
}

// Config 管理员账号播种配置，对应 USER_ADMIN_* 环境变量。
type Config struct {
	// Email 管理员邮箱；为空表示不播种管理员账号（只保证系统角色存在）。
	Email string
	// Password 管理员初始密码，仅在账号不存在或需要重置时使用。
	Password string
	// Nickname 管理员昵称，为空时使用 DefaultAdminNickname。
	Nickname string
	// ResetPassword 为 true 时每次启动都把密码重置为 Password。
	// 仅用于忘记管理员密码时的应急恢复，平时保持 false，避免重启即改密。
	ResetPassword bool
	// BcryptCost bcrypt 成本，与业务侧保持一致。
	BcryptCost int
}

// systemRoles 需要保证存在的系统角色。名称与描述与 database/user_svc.sql 中的种子一致，
// 权限明细（role_permission）仍由 SQL 维护。
var systemRoles = []model.Role{
	{Code: rbac.RoleSuperAdmin, Name: "超级管理员", Description: "拥有全部系统权限", IsSystem: true},
	{Code: rbac.RoleAdmin, Name: "管理员", Description: "管理用户和内容", IsSystem: true},
	{Code: rbac.RoleUser, Name: "普通用户", Description: "可创作和发布", IsSystem: true},
}

// systemPermissions 需要保证存在的权限定义，与 database/user_svc.sql 的种子一一对应
// （顺序也保持一致，因此全新库上生成的 id 与 SQL 导出相同）。
var systemPermissions = []model.Permission{
	{Code: "user:read", Name: "查看用户", Resource: "user", Action: "read"},
	{Code: "user:write", Name: "编辑用户", Resource: "user", Action: "write"},
	{Code: "user:delete", Name: "删除用户", Resource: "user", Action: "delete"},
	{Code: "user:admin", Name: "管理用户", Resource: "user", Action: "admin"},
	{Code: "novel:read", Name: "查看作品", Resource: "novel", Action: "read"},
	{Code: "novel:write", Name: "创作作品", Resource: "novel", Action: "write"},
	{Code: "novel:delete", Name: "删除作品", Resource: "novel", Action: "delete"},
	{Code: "novel:publish", Name: "发布作品", Resource: "novel", Action: "publish"},
	{Code: "novel:admin", Name: "管理所有作品", Resource: "novel", Action: "admin"},
	{Code: "material:read", Name: "查看素材", Resource: "material", Action: "read"},
	{Code: "material:write", Name: "管理素材", Resource: "material", Action: "write"},
	{Code: "publish:read", Name: "查看发布", Resource: "publish", Action: "read"},
	{Code: "publish:write", Name: "执行发布", Resource: "publish", Action: "write"},
	{Code: "publish:admin", Name: "管理发布", Resource: "publish", Action: "admin"},
	{Code: "system:admin", Name: "系统管理", Resource: "system", Action: "admin"},
	{Code: "hotspot:read", Name: "查看热点", Resource: "hotspot", Action: "read"},
	{Code: "hotspot:write", Name: "管理热点", Resource: "hotspot", Action: "write"},
}

// rolePermissionCodes 各系统角色的默认权限，与 database/user_svc.sql 的
// role_permission 种子一致：super_admin 拥有全部，user 拥有创作相关的 8 条。
//
// super_admin 的映射在校验上其实是冗余的（该角色在 rbac 服务里走通配，直接放行），
// 之所以仍然写入，是为了和 SQL 导出保持一致、并在后台「角色权限」界面上可见。
//
// admin 角色在种子数据里没有任何权限，这里不擅自扩大它的范围；
// 需要时在后台「角色权限」里配置（配置后播种不会再覆盖，见 ensureRolePermissions）。
var rolePermissionCodes = map[string][]string{
	rbac.RoleSuperAdmin: permissionCodes(),
	rbac.RoleUser: {
		"novel:read", "novel:write", "novel:publish",
		"material:read", "material:write",
		"publish:read", "publish:write",
		"hotspot:read",
	},
}

// permissionCodes 返回全部系统权限编码（按 systemPermissions 顺序）。
func permissionCodes() []string {
	codes := make([]string, 0, len(systemPermissions))
	for _, p := range systemPermissions {
		codes = append(codes, p.Code)
	}
	return codes
}

// Run 幂等执行播种：角色缺失则补建，管理员账号缺失则创建。
//
// 返回的错误不一定要阻断启动（服务已能对外提供登录等服务），由调用方决定；
// cmd/server 的做法是只记录日志，避免播种失败导致 Pod 反复重启。
func Run(ctx context.Context, users UserStore, rbacStore RBACStore, cfg Config) error {
	if users == nil || rbacStore == nil {
		return errors.New("播种依赖未注入")
	}
	if err := ensureRoles(ctx, rbacStore); err != nil {
		return err
	}
	if err := ensurePermissions(ctx, rbacStore); err != nil {
		return err
	}
	if err := ensureRolePermissions(ctx, rbacStore); err != nil {
		return err
	}
	return ensureAdmin(ctx, users, rbacStore, cfg)
}

// ensurePermissions 补齐缺失的权限定义。
func ensurePermissions(ctx context.Context, store RBACStore) error {
	existing, err := store.ListPermissions(ctx)
	if err != nil {
		return fmt.Errorf("查询权限失败: %w", err)
	}

	known := make(map[string]struct{}, len(existing))
	for _, p := range existing {
		known[p.Code] = struct{}{}
	}

	missing := make([]model.Permission, 0, len(systemPermissions))
	for _, want := range systemPermissions {
		if _, ok := known[want.Code]; !ok {
			missing = append(missing, want)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	if err := store.CreatePermissions(ctx, missing); err != nil {
		return fmt.Errorf("创建系统权限失败: %w", err)
	}

	logger.InfoContext(ctx, "已补齐系统权限", "count", len(missing))
	return nil
}

// ensureRolePermissions 为「当前没有任何权限」的系统角色写入默认权限映射。
//
// 只在角色权限为空时写入：全新库能自愈，同时不会覆盖后台里对角色权限的调整。
// 代价是把某个角色的权限全部清空属于有意为之的情况会在重启后被恢复默认，故打日志。
func ensureRolePermissions(ctx context.Context, store RBACStore) error {
	roles, err := store.ListRoles(ctx)
	if err != nil {
		return fmt.Errorf("查询角色失败: %w", err)
	}

	for _, role := range roles {
		codes := rolePermissionCodes[role.Code]
		if len(codes) == 0 {
			continue
		}
		current, err := store.GetRolePermissions(ctx, role.ID)
		if err != nil {
			return fmt.Errorf("查询角色 %s 的权限失败: %w", role.Code, err)
		}
		if len(current) > 0 {
			continue
		}
		if err := store.SetRolePermissions(ctx, role.ID, codes); err != nil {
			return fmt.Errorf("写入角色 %s 的默认权限失败: %w", role.Code, err)
		}
		logger.InfoContext(ctx, "已写入角色默认权限", "code", role.Code, "count", len(codes))
	}
	return nil
}

// ensureRoles 补齐缺失的系统角色。
func ensureRoles(ctx context.Context, store RBACStore) error {
	roles, err := store.ListRoles(ctx)
	if err != nil {
		return fmt.Errorf("查询角色失败: %w", err)
	}

	existing := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		existing[r.Code] = struct{}{}
	}

	for _, want := range systemRoles {
		if _, ok := existing[want.Code]; ok {
			continue
		}
		role := want // 复制一份，避免写入时改动包级变量
		// CreateRole 对 id 为 0 的角色会取当前最大 id + 1（role.id 没有绑定序列）
		if err := store.CreateRole(ctx, &role); err != nil {
			return fmt.Errorf("创建系统角色 %s 失败: %w", want.Code, err)
		}
		logger.InfoContext(ctx, "已补齐系统角色", "code", role.Code, "name", role.Name, "id", role.ID)
	}
	return nil
}

// ensureAdmin 保证管理员账号存在且拥有 super_admin 角色。
func ensureAdmin(ctx context.Context, users UserStore, rbacStore RBACStore, cfg Config) error {
	email := strings.ToLower(strings.TrimSpace(cfg.Email))
	if email == "" {
		logger.WarnContext(ctx, "未配置管理员账号（USER_ADMIN_EMAIL 为空），跳过管理员播种；首次部署请先配置后再启动")
		return nil
	}

	admin, err := users.FindByEmail(ctx, email)
	switch {
	case errors.Is(err, repository.ErrUserNotFound):
		admin, err = createAdmin(ctx, users, email, cfg)
		if err != nil {
			return err
		}
	case err != nil:
		return fmt.Errorf("查询管理员账号失败: %w", err)
	default:
		if err := syncPassword(ctx, users, admin, cfg); err != nil {
			return err
		}
	}

	return grantSuperAdmin(ctx, rbacStore, admin)
}

// createAdmin 创建管理员账号：邮箱直接标记为已验证，否则无法用密码登录。
func createAdmin(ctx context.Context, users UserStore, email string, cfg Config) (*model.User, error) {
	hash, err := hashPassword(cfg.Password, cfg.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("创建管理员账号失败: %w", err)
	}

	nickname := strings.TrimSpace(cfg.Nickname)
	if nickname == "" {
		nickname = DefaultAdminNickname
	}

	admin := &model.User{
		Email:         email,
		PasswordHash:  &hash,
		Nickname:      nickname,
		Status:        model.StatusNormal,
		EmailVerified: true,
	}
	if err := users.Create(ctx, admin); err != nil {
		return nil, fmt.Errorf("创建管理员账号失败: %w", err)
	}

	logger.InfoContext(ctx, "已创建管理员账号（密码来自 USER_ADMIN_PASSWORD）",
		"email", email, "nickname", nickname)
	return admin, nil
}

// syncPassword 只在「账号没有可用密码」或显式开启 ResetPassword 时改写密码哈希，
// 避免每次重启覆盖管理员自己修改过的密码。
func syncPassword(ctx context.Context, users UserStore, admin *model.User, cfg Config) error {
	noPassword := admin.PasswordHash == nil || strings.TrimSpace(*admin.PasswordHash) == ""
	if !noPassword && !cfg.ResetPassword {
		logger.InfoContext(ctx, "管理员账号已存在，跳过创建与改密",
			"email", admin.Email,
			"hint", "需要按配置重置密码请设 USER_ADMIN_RESET_PASSWORD=true")
		return nil
	}

	hash, err := hashPassword(cfg.Password, cfg.BcryptCost)
	if err != nil {
		return err
	}
	if err := users.UpdatePasswordHash(ctx, admin.UUID, hash); err != nil {
		return fmt.Errorf("重置管理员密码失败: %w", err)
	}

	if noPassword {
		logger.InfoContext(ctx, "管理员账号原本没有密码，已按配置补设", "email", admin.Email)
		return nil
	}
	logger.WarnContext(ctx, "已按 USER_ADMIN_RESET_PASSWORD 重置管理员密码", "email", admin.Email)
	return nil
}

// grantSuperAdmin 幂等地给管理员账号授予 super_admin 角色。
// granted_by 记录为该账号自身，与注册授予默认角色的做法一致。
func grantSuperAdmin(ctx context.Context, store RBACStore, admin *model.User) error {
	roles, err := store.GetUserRoleCodes(ctx, admin.UUID)
	if err != nil {
		return fmt.Errorf("查询管理员角色失败: %w", err)
	}
	if slices.Contains(roles, rbac.RoleSuperAdmin) {
		return nil
	}
	if err := store.AssignRole(ctx, admin.UUID, rbac.RoleSuperAdmin, admin.UUID); err != nil {
		return fmt.Errorf("授予管理员角色失败: %w", err)
	}
	logger.InfoContext(ctx, "已授予管理员角色", "email", admin.Email, "role", rbac.RoleSuperAdmin)
	return nil
}

// hashPassword 生成 bcrypt 哈希；cost 非法时回退 bcrypt 默认成本。
func hashPassword(password string, cost int) (string, error) {
	if password == "" {
		return "", errors.New("USER_ADMIN_PASSWORD 未配置，无法创建或重置管理员密码")
	}
	if len(password) > maxPasswordBytes {
		return "", fmt.Errorf("管理员密码超过 %d 字节，bcrypt 会静默截断，请改短", maxPasswordBytes)
	}
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("密码加密失败: %w", err)
	}
	return string(hash), nil
}
