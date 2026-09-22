package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/museflow/config-service/internal/model"
)

var (
	// ErrUserModelNotFound 自定义模型不存在，或不属于当前用户。
	ErrUserModelNotFound = errors.New("自定义模型不存在")
)

// UserModelListFilter 用户自定义模型列表过滤条件。
type UserModelListFilter struct {
	UserUUID uuid.UUID
	// ProviderID 0 表示不限渠道。
	ProviderID int64
	Offset     int
	Limit      int
}

// UserModelRepository 用户自定义模型数据访问接口。
type UserModelRepository interface {
	// List 按归属分页查询自定义模型，排序为 id 升序。
	List(ctx context.Context, filter UserModelListFilter) ([]model.UserModel, int64, error)
	// GetByID 按主键查询（无归属限制）。
	GetByID(ctx context.Context, id int64) (*model.UserModel, error)
	// GetOwnedByID 按主键 + 归属查询。
	GetOwnedByID(ctx context.Context, id int64, userUUID uuid.UUID) (*model.UserModel, error)
	// Create 新增自定义模型，同一渠道下调用标识重复时返回 ErrModelDuplicated。
	Create(ctx context.Context, m *model.UserModel) error
	// Update 按主键 + 归属更新指定列。
	Update(ctx context.Context, id int64, userUUID uuid.UUID, updates map[string]any) error
	// Delete 按主键 + 归属删除。
	Delete(ctx context.Context, id int64, userUUID uuid.UUID) error
	// ListActiveByUserAndType 用户端可见的自定义模型：只取启用中的。
	// 同时要求所属渠道处于启用状态——与平台侧的 ListPlatformActive 保持同一口径，
	// 否则用户停用渠道后，其下模型仍会出现在可选列表里。
	ListActiveByUserAndType(ctx context.Context, userUUID uuid.UUID, modelType string) ([]model.UserModel, error)
}

type userModelRepository struct {
	db *gorm.DB
}

// NewUserModelRepository 构造用户自定义模型仓储。
func NewUserModelRepository(db *gorm.DB) UserModelRepository {
	return &userModelRepository{db: db}
}

func (r *userModelRepository) List(ctx context.Context, filter UserModelListFilter) ([]model.UserModel, int64, error) {
	query := r.db.WithContext(ctx).
		Model(&model.UserModel{}).
		Where("user_uuid = ?", filter.UserUUID)

	if filter.ProviderID > 0 {
		query = query.Where("user_provider_id = ?", filter.ProviderID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}

	var models []model.UserModel
	err := query.
		Order("id ASC").
		Offset(filter.Offset).
		Limit(filter.Limit).
		Find(&models).Error
	if err != nil {
		return nil, 0, err
	}

	return models, total, nil
}

func (r *userModelRepository) GetByID(ctx context.Context, id int64) (*model.UserModel, error) {
	var m model.UserModel
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserModelNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *userModelRepository) GetOwnedByID(ctx context.Context, id int64, userUUID uuid.UUID) (*model.UserModel, error) {
	var m model.UserModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_uuid = ?", id, userUUID).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserModelNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *userModelRepository) Create(ctx context.Context, m *model.UserModel) error {
	err := r.db.WithContext(ctx).Create(m).Error
	if err == nil || !isUniqueViolation(err) {
		return err
	}

	// user_model 只有 (user_uuid, user_provider_id, api_model) 一个唯一索引
	if strings.Contains(uniqueConstraintName(err), "idx_user_model_user_api_model") {
		return ErrModelDuplicated
	}
	return err
}

func (r *userModelRepository) Update(ctx context.Context, id int64, userUUID uuid.UUID, updates map[string]any) error {
	res := r.db.WithContext(ctx).
		Model(&model.UserModel{}).
		Where("id = ? AND user_uuid = ?", id, userUUID).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrUserModelNotFound
	}
	return nil
}

func (r *userModelRepository) Delete(ctx context.Context, id int64, userUUID uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND user_uuid = ?", id, userUUID).
		Delete(&model.UserModel{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrUserModelNotFound
	}
	return nil
}

func (r *userModelRepository) ListActiveByUserAndType(ctx context.Context, userUUID uuid.UUID, modelType string) ([]model.UserModel, error) {
	query := r.db.WithContext(ctx).
		Model(&model.UserModel{}).
		Joins(`JOIN "config_svc"."user_model_provider" p ON p.id = "config_svc"."user_model"."user_provider_id" AND p.user_uuid = "config_svc"."user_model"."user_uuid"`).
		Where(`"config_svc"."user_model"."user_uuid" = ? AND "config_svc"."user_model"."is_active" = ? AND p.is_active = ?`, userUUID, true, true)

	if t := trimmed(modelType); t != "" {
		query = query.Where("model_type = ?", t)
	}

	var models []model.UserModel
	if err := query.Order("id DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	return models, nil
}
