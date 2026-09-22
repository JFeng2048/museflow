package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/museflow/config-service/internal/model"
)

// ErrSettingNotFound 系统配置不存在。
var ErrSettingNotFound = errors.New("系统配置不存在")

// SettingListFilter 系统配置列表过滤条件。
type SettingListFilter struct {
	// ConfigGroup 空表示全部分组。
	ConfigGroup string
	// OnlyPublic 只返回可下发前端的配置。
	OnlyPublic bool
}

// SettingRepository 通用系统配置数据访问接口。
type SettingRepository interface {
	// List 按条件查询配置，排序为 id 升序（即创建顺序）。
	List(ctx context.Context, filter SettingListFilter) ([]model.SystemSetting, error)
	// Get 按 (config_group, key) 查询配置。
	Get(ctx context.Context, configGroup, key string) (*model.SystemSetting, error)
	// Upsert 存在则更新、不存在则插入。
	// 执行后 setting 会被数据库中的真实行覆盖（含 id 与时间戳），
	// 因为 ON CONFLICT DO UPDATE 的 RETURNING 拿不到既有行的自增主键。
	Upsert(ctx context.Context, setting *model.SystemSetting) error
	// Delete 按 (config_group, key) 删除，不存在时返回 ErrSettingNotFound。
	Delete(ctx context.Context, configGroup, key string) error
}

type settingRepository struct {
	db *gorm.DB
}

// NewSettingRepository 构造系统配置仓储。
func NewSettingRepository(db *gorm.DB) SettingRepository {
	return &settingRepository{db: db}
}

func (r *settingRepository) List(ctx context.Context, filter SettingListFilter) ([]model.SystemSetting, error) {
	query := r.db.WithContext(ctx).Model(&model.SystemSetting{})

	if group := trimmed(filter.ConfigGroup); group != "" {
		query = query.Where("config_group = ?", group)
	}
	if filter.OnlyPublic {
		query = query.Where("is_public = ?", true)
	}

	var settings []model.SystemSetting
	if err := query.Order("id ASC").Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

func (r *settingRepository) Get(ctx context.Context, configGroup, key string) (*model.SystemSetting, error) {
	var setting model.SystemSetting
	err := r.db.WithContext(ctx).
		Where("config_group = ? AND key = ?", configGroup, key).
		First(&setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSettingNotFound
	}
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

func (r *settingRepository) Upsert(ctx context.Context, setting *model.SystemSetting) error {
	// id 不参与冲突更新：它是序列生成的内部标识，
	// 覆盖过去会让「同一个逻辑配置」在不同次写入后主键漂移。
	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "config_group"}, {Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"value", "secret_value", "is_secret", "value_type", "is_public", "description",
			}),
		}).
		Create(setting).Error; err != nil {
		return err
	}

	// created_at 不在更新列表里，冲突分支下 RETURNING 返回的是旧行的创建时间；
	// 重新取一次，保证回显给调用方的 id 与时间戳都是数据库里的真实值。
	fresh, err := r.Get(ctx, setting.ConfigGroup, setting.Key)
	if err != nil {
		return err
	}
	*setting = *fresh
	return nil
}

func (r *settingRepository) Delete(ctx context.Context, configGroup, key string) error {
	res := r.db.WithContext(ctx).
		Where("config_group = ? AND key = ?", configGroup, key).
		Delete(&model.SystemSetting{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrSettingNotFound
	}
	return nil
}
