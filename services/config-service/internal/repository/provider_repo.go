package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/museflow/config-service/internal/model"
)

var (
	// ErrProviderNotFound 平台渠道不存在。
	ErrProviderNotFound = errors.New("模型渠道不存在")
	// ErrProviderCodeExists 渠道编码已被占用。
	ErrProviderCodeExists = errors.New("渠道编码已存在")
	// ErrProviderHasModels 渠道下仍挂有模型，不允许直接删除。
	ErrProviderHasModels = errors.New("该渠道下仍有模型，请先删除或转移这些模型")
)

// ProviderListFilter 平台渠道列表过滤条件。
type ProviderListFilter struct {
	Keyword    string // 按 code / name 模糊匹配，空表示不限
	OnlyActive bool   // 只返回启用中的渠道
	Offset     int
	Limit      int
}

// ProviderRepository 平台模型渠道数据访问接口。
//
// api_key 只写不读：仓储不提供任何解密入口，返回给上层的密文仅用于确认已落库。
type ProviderRepository interface {
	// List 按条件分页查询渠道，排序为 sort_order 升序、id 升序。
	List(ctx context.Context, filter ProviderListFilter) ([]model.ModelProvider, int64, error)
	// GetByID 按主键查询渠道。
	GetByID(ctx context.Context, id int64) (*model.ModelProvider, error)
	// Create 新增渠道，编码冲突时返回 ErrProviderCodeExists。
	Create(ctx context.Context, provider *model.ModelProvider) error
	// Update 按主键更新指定列，渠道不存在时返回 ErrProviderNotFound。
	Update(ctx context.Context, id int64, updates map[string]any) error
	// Delete 删除渠道，仍挂有模型时返回 ErrProviderHasModels。
	Delete(ctx context.Context, id int64) error
	// ExistsByCode 判断编码是否已被占用。
	ExistsByCode(ctx context.Context, code string) (bool, error)
}

type providerRepository struct {
	db *gorm.DB
}

// NewProviderRepository 构造平台渠道仓储。
func NewProviderRepository(db *gorm.DB) ProviderRepository {
	return &providerRepository{db: db}
}

func (r *providerRepository) List(ctx context.Context, filter ProviderListFilter) ([]model.ModelProvider, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.ModelProvider{})

	if keyword := trimmed(filter.Keyword); keyword != "" {
		query = query.Where("code ILIKE ? OR name ILIKE ?", likePattern(keyword), likePattern(keyword))
	}
	if filter.OnlyActive {
		query = query.Where("is_active = ?", true)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}

	var providers []model.ModelProvider
	err := query.
		Order("sort_order ASC, id ASC").
		Offset(filter.Offset).
		Limit(filter.Limit).
		Find(&providers).Error
	if err != nil {
		return nil, 0, err
	}

	return providers, total, nil
}

func (r *providerRepository) GetByID(ctx context.Context, id int64) (*model.ModelProvider, error) {
	var provider model.ModelProvider
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&provider).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrProviderNotFound
	}
	if err != nil {
		return nil, err
	}
	return &provider, nil
}

func (r *providerRepository) Create(ctx context.Context, provider *model.ModelProvider) error {
	err := r.db.WithContext(ctx).Create(provider).Error
	return mapUniqueViolation(err, ErrProviderCodeExists)
}

func (r *providerRepository) Update(ctx context.Context, id int64, updates map[string]any) error {
	res := r.db.WithContext(ctx).
		Model(&model.ModelProvider{}).
		Where("id = ?", id).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	// 表上有 BEFORE UPDATE 触发器刷新 updated_at，因此匹配到的行一定会被计入
	// RowsAffected：返回 0 只可能是主键不存在。
	if res.RowsAffected == 0 {
		return ErrProviderNotFound
	}
	return nil
}

func (r *providerRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 渠道下的模型没有物理外键级联，删除前先确认无残留，
		// 否则会留下一批指向不存在渠道的孤儿模型（用户端再也刷不出来，但后台仍能列出）。
		var count int64
		if err := tx.Model(&model.Model{}).Where("provider_id = ?", id).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrProviderHasModels
		}

		res := tx.Where("id = ?", id).Delete(&model.ModelProvider{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrProviderNotFound
		}
		return nil
	})
}

func (r *providerRepository) ExistsByCode(ctx context.Context, code string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ModelProvider{}).
		Where("code = ?", code).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
