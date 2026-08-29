package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/museflow/user-service/internal/model"
)

// ErrOAuthNotFound 第三方账号未绑定。
var ErrOAuthNotFound = errors.New("未绑定该第三方账号")

// OAuthRepository 第三方登录数据访问接口。
type OAuthRepository interface {
	// Create 创建绑定记录。
	Create(ctx context.Context, o *model.OAuth) error
	// FindByProvider 按平台与第三方用户标识查询绑定记录。
	FindByProvider(ctx context.Context, provider, providerUserID string) (*model.OAuth, error)
	// ListByUser 列出用户已绑定的全部第三方账号。
	ListByUser(ctx context.Context, userUUID uuid.UUID) ([]model.OAuth, error)
	// Delete 解绑（删除绑定记录）。
	Delete(ctx context.Context, userUUID uuid.UUID, provider string) error
	// TouchLastLogin 更新最后登录时间。
	TouchLastLogin(ctx context.Context, id int64) error
}

type oauthRepository struct {
	db *gorm.DB
}

// NewOAuthRepository 构造第三方登录仓储。
func NewOAuthRepository(db *gorm.DB) OAuthRepository {
	return &oauthRepository{db: db}
}

func (r *oauthRepository) Create(ctx context.Context, o *model.OAuth) error {
	return r.db.WithContext(ctx).Create(o).Error
}

func (r *oauthRepository) FindByProvider(ctx context.Context, provider, providerUserID string) (*model.OAuth, error) {
	var o model.OAuth
	err := r.db.WithContext(ctx).
		Where("provider = ? AND provider_user_id = ? AND is_active = ?", provider, providerUserID, true).
		First(&o).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrOAuthNotFound
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *oauthRepository) ListByUser(ctx context.Context, userUUID uuid.UUID) ([]model.OAuth, error) {
	var list []model.OAuth
	if err := r.db.WithContext(ctx).
		Where("user_uuid = ? AND is_active = ?", userUUID, true).
		Order("created_at ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *oauthRepository) Delete(ctx context.Context, userUUID uuid.UUID, provider string) error {
	res := r.db.WithContext(ctx).
		Where("user_uuid = ? AND provider = ?", userUUID, provider).
		Delete(&model.OAuth{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrOAuthNotFound
	}
	return nil
}

func (r *oauthRepository) TouchLastLogin(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Model(&model.OAuth{}).
		Where("id = ?", id).
		Update("last_login_at", gorm.Expr("CURRENT_TIMESTAMP")).Error
}
