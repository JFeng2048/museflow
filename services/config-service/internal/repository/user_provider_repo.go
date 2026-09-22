package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/museflow/config-service/internal/model"
)

var (
	// ErrUserProviderNotFound 自定义渠道不存在，或不属于当前用户。
	// 归属校验失败与真的不存在返回同一错误，不暴露他人资源是否存在。
	ErrUserProviderNotFound = errors.New("自定义渠道不存在")
	// ErrUserProviderNameExists 同一用户下已有同名自定义渠道。
	ErrUserProviderNameExists = errors.New("同名自定义渠道已存在")
	// ErrUserProviderHasModels 渠道下仍挂有自定义模型，不允许直接删除。
	ErrUserProviderHasModels = errors.New("该渠道下仍有模型，请先删除这些模型")
)

// UserProviderRepository 用户自定义渠道数据访问接口。
//
// 所有写操作都带 user_uuid 条件：即便上层漏传归属，数据层也不会越权修改。
type UserProviderRepository interface {
	// ListByUser 列出某用户的全部自定义渠道，按创建顺序。
	ListByUser(ctx context.Context, userUUID uuid.UUID) ([]model.UserModelProvider, error)
	// GetByID 按主键查询（无归属限制，仅供内部联动使用）。
	GetByID(ctx context.Context, id int64) (*model.UserModelProvider, error)
	// GetOwnedByID 按主键 + 归属查询，越权或不存在都返回 ErrUserProviderNotFound。
	GetOwnedByID(ctx context.Context, id int64, userUUID uuid.UUID) (*model.UserModelProvider, error)
	// Create 新增自定义渠道，重名时返回 ErrUserProviderNameExists。
	Create(ctx context.Context, provider *model.UserModelProvider) error
	// Update 按主键 + 归属更新指定列。
	Update(ctx context.Context, id int64, userUUID uuid.UUID, updates map[string]any) error
	// Delete 按主键 + 归属删除，仍挂有模型时返回 ErrUserProviderHasModels。
	Delete(ctx context.Context, id int64, userUUID uuid.UUID) error
	// ExistsByName 判断同一用户下是否已有同名渠道。
	ExistsByName(ctx context.Context, userUUID uuid.UUID, name string) (bool, error)
}

type userProviderRepository struct {
	db *gorm.DB
}

// NewUserProviderRepository 构造用户自定义渠道仓储。
func NewUserProviderRepository(db *gorm.DB) UserProviderRepository {
	return &userProviderRepository{db: db}
}

func (r *userProviderRepository) ListByUser(ctx context.Context, userUUID uuid.UUID) ([]model.UserModelProvider, error) {
	var providers []model.UserModelProvider
	err := r.db.WithContext(ctx).
		Where("user_uuid = ?", userUUID).
		Order("id ASC").
		Find(&providers).Error
	if err != nil {
		return nil, err
	}
	return providers, nil
}

func (r *userProviderRepository) GetByID(ctx context.Context, id int64) (*model.UserModelProvider, error) {
	var provider model.UserModelProvider
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&provider).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserProviderNotFound
	}
	if err != nil {
		return nil, err
	}
	return &provider, nil
}

func (r *userProviderRepository) GetOwnedByID(ctx context.Context, id int64, userUUID uuid.UUID) (*model.UserModelProvider, error) {
	var provider model.UserModelProvider
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_uuid = ?", id, userUUID).
		First(&provider).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserProviderNotFound
	}
	if err != nil {
		return nil, err
	}
	return &provider, nil
}

func (r *userProviderRepository) Create(ctx context.Context, provider *model.UserModelProvider) error {
	err := r.db.WithContext(ctx).Create(provider).Error
	return mapUniqueViolation(err, ErrUserProviderNameExists)
}

func (r *userProviderRepository) Update(ctx context.Context, id int64, userUUID uuid.UUID, updates map[string]any) error {
	res := r.db.WithContext(ctx).
		Model(&model.UserModelProvider{}).
		Where("id = ? AND user_uuid = ?", id, userUUID).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrUserProviderNotFound
	}
	return nil
}

func (r *userProviderRepository) Delete(ctx context.Context, id int64, userUUID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 渠道与自定义模型之间没有物理外键，删除前先确认无残留
		var count int64
		if err := tx.Model(&model.UserModel{}).
			Where("user_provider_id = ? AND user_uuid = ?", id, userUUID).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrUserProviderHasModels
		}

		res := tx.Where("id = ? AND user_uuid = ?", id, userUUID).Delete(&model.UserModelProvider{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrUserProviderNotFound
		}
		return nil
	})
}

func (r *userProviderRepository) ExistsByName(ctx context.Context, userUUID uuid.UUID, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.UserModelProvider{}).
		Where("user_uuid = ? AND name = ?", userUUID, name).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
