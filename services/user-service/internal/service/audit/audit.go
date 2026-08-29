// Package audit 提供审计日志记录与查询能力。
//
// 依赖方向：audit -> repository（不依赖 auth / admin / rbac），
// 因此任何业务层都可安全引用 audit 记录操作日志。
//
// 设计要点：审计属于旁路能力，写入失败仅记录告警日志，
// 绝不阻断主业务流程（登录、注册、改密等）。
package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/museflow/pkg/logger"
	"github.com/museflow/user-service/internal/model"
	"github.com/museflow/user-service/internal/repository"
)

// Service 审计日志服务。
type Service struct {
	repo repository.AuditRepository
}

// NewService 构造审计服务。
func NewService(repo repository.AuditRepository) *Service {
	return &Service{repo: repo}
}

// Entry 单条审计日志的入参。
type Entry struct {
	UserUUID   string // 操作人 UUID，空表示系统操作
	Action     string // 操作类型，见 model.AuditAction* 常量
	Resource   string // 资源类型，见 model.AuditResource* 常量
	ResourceID string // 资源标识
	IP         string // 请求 IP，空则不记录
	UserAgent  string // 客户端信息
	Detail     any    // 详细数据，会被序列化为 JSON
}

// Record 写入一条审计日志。
// 失败时仅记录告警，不返回错误，避免影响主流程。
func (s *Service) Record(ctx context.Context, e Entry) {
	if s == nil || s.repo == nil {
		return
	}

	log := &model.AuditLog{
		Action:     e.Action,
		Resource:   e.Resource,
		ResourceID: e.ResourceID,
		UserAgent:  e.UserAgent,
	}

	if e.UserUUID != "" {
		if id, err := uuid.Parse(e.UserUUID); err == nil {
			log.UserUUID = id
		}
	}
	if e.IP != "" {
		log.IP = &e.IP
	}
	if e.Detail != nil {
		// 序列化失败不影响主流程，忽略 detail 即可
		if raw, err := json.Marshal(e.Detail); err == nil {
			log.Detail = string(raw)
		}
	}

	if err := s.repo.Create(ctx, log); err != nil {
		logger.WarnContext(ctx, "写入审计日志失败",
			"action", e.Action, "resource", e.Resource, logger.Err(err))
	}
}

// List 分页查询审计日志，支持按用户 / 操作类型 / 时间范围筛选。
func (s *Service) List(ctx context.Context, userUUID, action string, from, to *time.Time, offset, limit int) ([]model.AuditLog, int64, error) {
	logs, err := s.repo.List(ctx, userUUID, action, from, to, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.Count(ctx, userUUID, action, from, to)
	if err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
