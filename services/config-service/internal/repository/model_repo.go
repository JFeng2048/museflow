package repository

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/museflow/config-service/internal/model"
)

var (
	// ErrModelNotFound 模型不存在。
	ErrModelNotFound = errors.New("模型不存在")
	// ErrModelCodeExists 模型编码已被占用（全局唯一）。
	ErrModelCodeExists = errors.New("模型编码已存在")
	// ErrModelDuplicated 同一渠道下已存在相同调用标识的模型。
	ErrModelDuplicated = errors.New("该渠道下已存在相同调用标识的模型")
)

// ModelListFilter 平台模型列表过滤条件。
type ModelListFilter struct {
	ProviderID int64  // 0 表示不限渠道
	ModelType  string // 空表示不限类型
	Keyword    string // 按 code / name / api_model 模糊匹配
	OnlyActive bool   // 只返回已上架的模型
	Offset     int
	Limit      int
}

// ModelRepository 平台模型数据访问接口。
type ModelRepository interface {
	// List 按条件分页查询模型，排序为 sort_order 升序、id 升序。
	List(ctx context.Context, filter ModelListFilter) ([]model.Model, int64, error)
	// GetByID 按主键查询模型。
	GetByID(ctx context.Context, id int64) (*model.Model, error)
	// Create 新增模型，编码或调用标识冲突时返回对应领域错误。
	Create(ctx context.Context, m *model.Model) error
	// Update 按主键更新指定列，模型不存在时返回 ErrModelNotFound。
	Update(ctx context.Context, id int64, updates map[string]any) error
	// Delete 删除模型。
	Delete(ctx context.Context, id int64) error
	// ListPlatformActive 用户端可见的平台模型：模型启用，且所属渠道也启用。
	// 按渠道排序、模型排序输出，即用户端默认展示顺序。
	ListPlatformActive(ctx context.Context, modelType string) ([]model.Model, error)
}

type modelRepository struct {
	db *gorm.DB
}

// NewModelRepository 构造平台模型仓储。
func NewModelRepository(db *gorm.DB) ModelRepository {
	return &modelRepository{db: db}
}

func (r *modelRepository) List(ctx context.Context, filter ModelListFilter) ([]model.Model, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.Model{})

	if filter.ProviderID > 0 {
		query = query.Where("provider_id = ?", filter.ProviderID)
	}
	if t := trimmed(filter.ModelType); t != "" {
		query = query.Where("model_type = ?", t)
	}
	if keyword := trimmed(filter.Keyword); keyword != "" {
		pattern := likePattern(keyword)
		query = query.Where("code ILIKE ? OR name ILIKE ? OR api_model ILIKE ?", pattern, pattern, pattern)
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

	var models []model.Model
	err := query.
		Order("sort_order ASC, id ASC").
		Offset(filter.Offset).
		Limit(filter.Limit).
		Find(&models).Error
	if err != nil {
		return nil, 0, err
	}

	return models, total, nil
}

func (r *modelRepository) GetByID(ctx context.Context, id int64) (*model.Model, error) {
	var m model.Model
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrModelNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *modelRepository) Create(ctx context.Context, m *model.Model) error {
	err := r.db.WithContext(ctx).Create(m).Error
	if err == nil || !isUniqueViolation(err) {
		return err
	}

	// model 上有两个唯一索引，按约束名分流，文案才能指到具体字段
	if strings.Contains(uniqueConstraintName(err), "idx_model_code") {
		return ErrModelCodeExists
	}
	return ErrModelDuplicated
}

func (r *modelRepository) Update(ctx context.Context, id int64, updates map[string]any) error {
	res := r.db.WithContext(ctx).
		Model(&model.Model{}).
		Where("id = ?", id).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	// 同 providerRepository.Update：触发器保证匹配到的行一定计入 RowsAffected
	if res.RowsAffected == 0 {
		return ErrModelNotFound
	}
	return nil
}

func (r *modelRepository) Delete(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Model{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrModelNotFound
	}
	return nil
}

func (r *modelRepository) ListPlatformActive(ctx context.Context, modelType string) ([]model.Model, error) {
	query := r.db.WithContext(ctx).
		Model(&model.Model{}).
		Joins(`JOIN "config_svc"."model_provider" p ON p.id = "config_svc"."model"."provider_id"`).
		Where(`"config_svc"."model"."is_active" = ? AND p.is_active = ?`, true, true)

	if t := trimmed(modelType); t != "" {
		query = query.Where(`"config_svc"."model"."model_type" = ?`, t)
	}

	// 渠道停用时其下模型一律不可见，这里是用户端清单的唯一入口，
	// 漏掉 p.is_active 会把已停用渠道的模型继续刷给用户。
	var models []model.Model
	err := query.
		Order(`p.sort_order ASC, "config_svc"."model"."sort_order" ASC, "config_svc"."model"."id" ASC`).
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	return models, nil
}
