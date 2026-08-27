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

// UserRepository 用户数据访问接口。
type UserRepository interface {
	Create(ctx context.Context, u *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByUUID(ctx context.Context, id uuid.UUID) (*model.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	UpdateLoginInfo(ctx context.Context, id uuid.UUID, ip, platform string) error
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
