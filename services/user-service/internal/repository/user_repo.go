// Package repository 提供数据访问层，屏蔽底层 GORM 细节。
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
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
