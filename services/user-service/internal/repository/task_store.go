package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/museflow/user-service/internal/pkg/queue"
)

// Redis 键前缀（异步任务状态）。
const (
	// taskStatusKeyPrefix 任务状态快照：task:status:{task_id} -> TaskStatus(JSON)
	taskStatusKeyPrefix = "task:status:"
	// taskEventChannelPrefix 任务状态广播频道：task:events:{task_id}
	taskEventChannelPrefix = "task:events:"
)

// TaskStore 异步任务状态存储。
//
// 在 queue.StatusStore（写入 + 快照读取）之外补充 Subscribe，
// 供 gRPC WatchTask / 网关 SSE 实时订阅任务进度。
type TaskStore interface {
	queue.StatusStore
	// Subscribe 订阅任务状态变更，返回事件通道与取消订阅函数。
	// 调用方必须在使用结束后调用 cancel 释放 Redis 订阅连接。
	Subscribe(ctx context.Context, taskID string) (<-chan queue.TaskStatus, func(), error)
}

type redisTaskStore struct {
	rdb *redis.Client
}

// NewTaskStore 构造基于 Redis 的任务状态存储。
func NewTaskStore(rdb *redis.Client) TaskStore {
	return &redisTaskStore{rdb: rdb}
}

func taskStatusKey(taskID string) string {
	return taskStatusKeyPrefix + taskID
}

func taskEventChannel(taskID string) string {
	return taskEventChannelPrefix + taskID
}

// Update 写入状态快照并广播，ttl 为状态保留时长。
//
// 写入与广播放在同一管道：即使没有订阅者，快照也能被后到的订阅者读到，
// 避免「任务已完成但客户端订阅时永远等不到事件」的竞态。
func (s *redisTaskStore) Update(ctx context.Context, st queue.TaskStatus, ttl time.Duration) error {
	if st.TaskID == "" {
		return fmt.Errorf("任务状态缺少 task_id")
	}
	data, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("序列化任务状态失败: %w", err)
	}

	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, taskStatusKey(st.TaskID), data, ttl)
	pipe.Publish(ctx, taskEventChannel(st.TaskID), data)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("写入任务状态失败: %w", err)
	}
	return nil
}

// Get 读取当前状态快照，不存在时返回 (nil, nil)。
func (s *redisTaskStore) Get(ctx context.Context, taskID string) (*queue.TaskStatus, error) {
	data, err := s.rdb.Get(ctx, taskStatusKey(taskID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取任务状态失败: %w", err)
	}
	var st queue.TaskStatus
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("解析任务状态失败: %w", err)
	}
	return &st, nil
}

// Subscribe 订阅任务状态变更事件。
//
// 返回的通道由内部协程写入，ctx 取消或调用 cancel 后协程退出并关闭通道。
func (s *redisTaskStore) Subscribe(ctx context.Context, taskID string) (<-chan queue.TaskStatus, func(), error) {
	pubsub := s.rdb.Subscribe(ctx, taskEventChannel(taskID))
	// 等待订阅在服务端建立，确保后续写入不会被漏掉
	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		return nil, nil, fmt.Errorf("订阅任务状态失败: %w", err)
	}

	out := make(chan queue.TaskStatus, 8)
	done := make(chan struct{})
	var closed bool

	go func() {
		defer close(out)
		defer close(done)
		for msg := range pubsub.Channel() {
			var st queue.TaskStatus
			if err := json.Unmarshal([]byte(msg.Payload), &st); err != nil {
				continue
			}
			select {
			case out <- st:
			case <-ctx.Done():
				return
			case <-done:
				return
			}
		}
	}()

	cancel := func() {
		if closed {
			return
		}
		closed = true
		_ = pubsub.Close()
		<-done
	}
	return out, cancel, nil
}
