package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/museflow/user-service/internal/model"
)

// AuditRepository 审计日志数据访问接口。
type AuditRepository interface {
	// Create 写入一条审计日志。
	Create(ctx context.Context, log *model.AuditLog) error
	// List 分页查询审计日志，支持按用户 / 操作类型 / 时间范围筛选。
	List(ctx context.Context, userUUID string, action string, from, to *time.Time, offset, limit int) ([]model.AuditLog, error)
	// Count 统计符合条件的审计日志总数。
	Count(ctx context.Context, userUUID string, action string, from, to *time.Time) (int64, error)
}

type auditRepository struct {
	db *gorm.DB
}

// NewAuditRepository 构造审计日志仓储。
func NewAuditRepository(db *gorm.DB) AuditRepository {
	return &auditRepository{db: db}
}

func (r *auditRepository) Create(ctx context.Context, log *model.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// List 分页查询审计日志（按创建时间倒序）。
func (r *auditRepository) List(ctx context.Context, userUUID string, action string, from, to *time.Time, offset, limit int) ([]model.AuditLog, error) {
	q := r.db.WithContext(ctx).Model(&model.AuditLog{})
	q = applyAuditFilters(q, userUUID, action, from, to)

	var logs []model.AuditLog
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

func (r *auditRepository) Count(ctx context.Context, userUUID string, action string, from, to *time.Time) (int64, error) {
	var total int64
	q := r.db.WithContext(ctx).Model(&model.AuditLog{})
	q = applyAuditFilters(q, userUUID, action, from, to)
	if err := q.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func applyAuditFilters(q *gorm.DB, userUUID string, action string, from, to *time.Time) *gorm.DB {
	if userUUID != "" {
		if id, err := uuid.Parse(userUUID); err == nil {
			q = q.Where("user_uuid = ?", id)
		}
	}
	if action != "" {
		q = q.Where("action = ?", action)
	}
	if from != nil {
		q = q.Where("created_at >= ?", *from)
	}
	if to != nil {
		q = q.Where("created_at <= ?", *to)
	}
	return q
}
