// Package task 提供异步任务进度的查询与订阅能力。
//
// 典型链路：业务接口投递 asynq 任务 -> Worker 消费并把进度写入 Redis ->
// gRPC WatchTask 订阅 -> 网关以 SSE 推送给前端。
package task

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/museflow/user-service/internal/pkg/queue"
	"github.com/museflow/user-service/internal/repository"
)

// 进度订阅的默认上限：避免客户端忘记关闭连接导致协程长期挂起。
const (
	// DefaultWatchTimeout 单次订阅的最长持续时间。
	DefaultWatchTimeout = 3 * time.Minute
	// ReconnectHint 建议客户端的重连间隔，通过 SSE 的 retry 字段下发。
	ReconnectHint = 3 * time.Second
)

// ErrTaskNotFound 任务不存在或状态已过期。
var ErrTaskNotFound = errors.New("任务不存在或已过期，请重新发起请求")

// Service 任务进度服务。
type Service struct {
	store repository.TaskStore
}

// NewService 构造任务进度服务。
func NewService(store repository.TaskStore) *Service {
	return &Service{store: store}
}

// Get 查询任务当前状态；不存在时返回 ErrTaskNotFound。
func (s *Service) Get(ctx context.Context, taskID string) (*queue.TaskStatus, error) {
	if s.store == nil {
		return nil, fmt.Errorf("任务状态存储未初始化")
	}
	st, err := s.store.Get(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, ErrTaskNotFound
	}
	return st, nil
}

// Stream 持续读取任务进度，直到终态、ctx 结束或超时。
//
// 这是对 Watch 的便捷封装：把「先快照后订阅」的样板逻辑收敛到一处，
// 供 gRPC WatchTask 直接调用。
func (s *Service) Stream(ctx context.Context, taskID string, timeout time.Duration, emit func(queue.TaskStatus) error) error {
	if timeout <= 0 {
		timeout = DefaultWatchTimeout
	}

	watchCtx, stop := context.WithTimeout(ctx, timeout)
	defer stop()

	// 先确认任务存在，避免为不存在的任务维护订阅
	if _, err := s.Get(watchCtx, taskID); err != nil {
		return err
	}

	// 顺序很重要：先订阅、再读快照。
	// 反过来的话，订阅建立那瞬间完成的任务，其事件会被漏掉，客户端只能干等超时。
	events, cancel, err := s.store.Subscribe(watchCtx, taskID)
	if err != nil {
		return err
	}
	defer cancel()

	// 补发当前快照：即使任务已经结束，客户端也能立刻拿到结果
	snapshot, err := s.store.Get(watchCtx, taskID)
	if err != nil {
		return err
	}
	if snapshot == nil {
		return ErrTaskNotFound
	}
	if err := emit(*snapshot); err != nil {
		return err
	}
	if queue.TerminalStatus(snapshot.Status) {
		return nil
	}

	for {
		select {
		case <-watchCtx.Done():
			return nil
		case st, ok := <-events:
			if !ok {
				return nil
			}
			// 丢弃订阅建立前产生的旧事件，快照已经覆盖到最新状态
			if st.UpdatedAt < snapshot.UpdatedAt {
				continue
			}
			if err := emit(st); err != nil {
				return err
			}
			if queue.TerminalStatus(st.Status) {
				return nil
			}
		}
	}
}
