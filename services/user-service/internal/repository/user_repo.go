// Package repository 提供数据访问层，屏蔽底层 GORM 细节。
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/museflow/user-service/internal/model"
)

// ErrUserNotFound 用户不存在。
var ErrUserNotFound = errors.New("用户不存在")

// 登录锁定策略（与需求一致：失败 5 次锁定 15 分钟）。
const (
	maxLoginFails = 5
	lockDuration  = 15 * time.Minute
)

// UserRepository 用户数据访问接口。
type UserRepository interface {
	Create(ctx context.Context, u *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByUUID(ctx context.Context, id uuid.UUID) (*model.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	UpdateLoginInfo(ctx context.Context, id uuid.UUID, ip, platform string) error
	// IncrementLoginFails 登录失败计数 +1，达到阈值时锁定 locked_until。
	// 返回更新后的用户（含最新计数与锁定时间）。
	IncrementLoginFails(ctx context.Context, email string) (*model.User, error)
	// ResetLoginFails 重置登录失败计数与锁定时间（登录成功时调用）。
	ResetLoginFails(ctx context.Context, id uuid.UUID) error
	// UpdateProfile 更新昵称 / 头像 / 简介（仅更新非空字段）。
	UpdateProfile(ctx context.Context, id uuid.UUID, nickname, avatarURL, bio string) error
	// UpdatePasswordHash 更新密码哈希。
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error
	// UpdateStatus 更新账号状态（冻结 / 解冻 / 注销）。
	UpdateStatus(ctx context.Context, id uuid.UUID, status int16) error
	// SaveMFASecret 保存 2FA 共享密钥（启用前暂存，此时 mfa_enabled 仍为 false）。
	SaveMFASecret(ctx context.Context, id uuid.UUID, secret string) error
	// EnableMFA 启用 2FA：置 mfa_enabled=true 并写入恢复码。
	EnableMFA(ctx context.Context, id uuid.UUID, recoveryCodes []string) error
	// DisableMFA 关闭 2FA：置 mfa_enabled=false，清空密钥与恢复码。
	DisableMFA(ctx context.Context, id uuid.UUID) error
	// UpdateRecoveryCodes 覆盖更新恢复码列表。
	UpdateRecoveryCodes(ctx context.Context, id uuid.UUID, recoveryCodes []string) error
	// SetEmailVerified 设置邮箱是否已验证。
	SetEmailVerified(ctx context.Context, id uuid.UUID, verified bool) error
	// UpdateEmail 修改账号邮箱（同时标记为已验证）。
	UpdateEmail(ctx context.Context, id uuid.UUID, email string) error
	// ListUsers 分页查询用户（支持关键字与状态过滤、排序）。
	ListUsers(ctx context.Context, keyword string, status int16, orderBy string, desc bool, offset, limit int) ([]model.User, error)
	// CountUsers 统计用户总数（与 ListUsers 过滤条件一致）。
	CountUsers(ctx context.Context, keyword string, status int16) (int64, error)
}

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 构造用户仓储。
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, u *model.User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) FindByUUID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).Where("uuid = ?", id).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("email = ?", email).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// UpdateLoginInfo 登录成功后更新最后登录信息。
// 使用 Updates 只更新指定列，避免覆盖其他字段。
func (r *userRepository) UpdateLoginInfo(ctx context.Context, id uuid.UUID, ip, platform string) error {
	updates := map[string]any{
		"last_login_at": time.Now(),
	}
	// inet 类型不接受空字符串，IP 为空时跳过该列
	if ip != "" {
		updates["last_login_ip"] = ip
	}
	if platform != "" {
		updates["last_login_platform"] = platform
	}

	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("uuid = ?", id).
		Updates(updates).Error
}

// IncrementLoginFails 登录失败计数 +1；达到阈值（maxLoginFails）时写 locked_until 锁定。
func (r *userRepository) IncrementLoginFails(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	fails := u.LoginFailCount + 1
	updates := map[string]any{"login_fail_count": fails}
	// 达到阈值即锁定，锁定时间顺延；未达阈值不改锁定时间
	if fails >= maxLoginFails {
		lu := time.Now().Add(lockDuration)
		updates["locked_until"] = lu
	}
	if err := r.db.WithContext(ctx).
		Model(&model.User{}).Where("uuid = ?", u.UUID).
		Updates(updates).Error; err != nil {
		return nil, err
	}
	u.LoginFailCount = fails
	if fails >= maxLoginFails {
		lu := time.Now().Add(lockDuration)
		u.LockedUntil = &lu
	}
	return &u, nil
}

// ResetLoginFails 登录成功后清零失败计数并解除锁定。
func (r *userRepository) ResetLoginFails(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("uuid = ?", id).
		Updates(map[string]any{
			"login_fail_count": 0,
			"locked_until":     nil,
		}).Error
}

// UpdateProfile 更新昵称 / 头像 / 简介，仅更新非空字段。
func (r *userRepository) UpdateProfile(ctx context.Context, id uuid.UUID, nickname, avatarURL, bio string) error {
	updates := map[string]any{}
	if nickname != "" {
		updates["nickname"] = nickname
	}
	if avatarURL != "" {
		updates["avatar_url"] = avatarURL
	}
	if bio != "" {
		updates["bio"] = bio
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("uuid = ?", id).
		Updates(updates).Error
}

// UpdatePasswordHash 更新密码哈希。
func (r *userRepository) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("uuid = ?", id).
		Updates(map[string]any{"password_hash": hash}).Error
}

// UpdateStatus 更新账号状态。
func (r *userRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status int16) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("uuid = ?", id).
		Updates(map[string]any{"status": status}).Error
}

// SaveMFASecret 保存 2FA 共享密钥（尚未启用）。
func (r *userRepository) SaveMFASecret(ctx context.Context, id uuid.UUID, secret string) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("uuid = ?", id).
		Updates(map[string]any{"mfa_secret": secret}).Error
}

// EnableMFA 启用 2FA 并写入恢复码。
func (r *userRepository) EnableMFA(ctx context.Context, id uuid.UUID, recoveryCodes []string) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("uuid = ?", id).
		Updates(map[string]any{
			"mfa_enabled":        true,
			"mfa_recovery_codes": pq.StringArray(recoveryCodes),
		}).Error
}

// DisableMFA 关闭 2FA，清空密钥与恢复码。
func (r *userRepository) DisableMFA(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("uuid = ?", id).
		Updates(map[string]any{
			"mfa_enabled":        false,
			"mfa_secret":         nil,
			"mfa_recovery_codes": pq.StringArray{},
		}).Error
}

// UpdateRecoveryCodes 覆盖更新恢复码列表。
func (r *userRepository) UpdateRecoveryCodes(ctx context.Context, id uuid.UUID, recoveryCodes []string) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("uuid = ?", id).
		Updates(map[string]any{"mfa_recovery_codes": pq.StringArray(recoveryCodes)}).Error
}

// SetEmailVerified 设置邮箱是否已验证。
func (r *userRepository) SetEmailVerified(ctx context.Context, id uuid.UUID, verified bool) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("uuid = ?", id).
		Update("email_verified", verified).Error
}

// UpdateEmail 修改账号邮箱（同时标记为已验证）。
func (r *userRepository) UpdateEmail(ctx context.Context, id uuid.UUID, email string) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("uuid = ?", id).
		Updates(map[string]interface{}{
			"email":          email,
			"email_verified": true,
		}).Error
}

// ListUsers 分页查询用户，支持关键字（邮箱 / 昵称）与状态过滤。
func (r *userRepository) ListUsers(ctx context.Context, keyword string, status int16, orderBy string, desc bool, offset, limit int) ([]model.User, error) {
	q := r.db.WithContext(ctx).Model(&model.User{})
	q = applyUserFilters(q, keyword, status)

	// 排序字段白名单，避免 SQL 注入
	col, ok := userSortColumns[orderBy]
	if !ok {
		col = "created_at"
	}
	order := col + " ASC"
	if desc {
		order = col + " DESC"
	}

	var users []model.User
	if err := q.Order(order).Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// CountUsers 统计符合过滤条件的用户总数。
func (r *userRepository) CountUsers(ctx context.Context, keyword string, status int16) (int64, error) {
	var total int64
	q := r.db.WithContext(ctx).Model(&model.User{})
	q = applyUserFilters(q, keyword, status)
	if err := q.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// userSortColumns 允许的排序字段白名单（前端传值 -> 数据库列名）。
var userSortColumns = map[string]string{
	"created_at": "created_at",
	"updated_at": "updated_at",
	"email":      "email",
	"nickname":   "nickname",
	"status":     "status",
	"":           "created_at",
}

func applyUserFilters(q *gorm.DB, keyword string, status int16) *gorm.DB {
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("email ILIKE ? OR nickname ILIKE ?", like, like)
	}
	if status > 0 {
		q = q.Where("status = ?", status)
	}
	return q
}
