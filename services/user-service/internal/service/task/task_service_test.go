package task

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/museflow/user-service/internal/pkg/queue"
)

// ---- 测试替身 ----

// fakeTaskStore 内存版任务状态存储，可在运行中动态推进状态。
type fakeTaskStore struct {
	current  *queue.TaskStatus
	updates  []queue.TaskStatus
	eventCh  chan queue.TaskStatus
	watchErr error
}

func (s *fakeTaskStore) Update(_ context.Context, st queue.TaskStatus, _ time.Duration) error {
	s.updates = append(s.updates, st)
	cp := st
	s.current = &cp
	if s.eventCh != nil {
		s.eventCh <- st
	}
	return nil
}

func (s *fakeTaskStore) Get(_ context.Context, _ string) (*queue.TaskStatus, error) {
	return s.current, nil
}

// Subscribe 返回一个可用 Update 推进的事件通道。
func (s *fakeTaskStore) Subscribe(_ context.Context, _ string) (<-chan queue.TaskStatus, func(), error) {
	if s.watchErr != nil {
		return nil, nil, s.watchErr
	}
	s.eventCh = make(chan queue.TaskStatus, 4)
	return s.eventCh, func() {}, nil
}

// ---- 用例 ----

func TestGetReturnsNotFoundWhenStatusExpired(t *testing.T) {
	svc := NewService(&fakeTaskStore{})
	if _, err := svc.Get(context.Background(), "missing-task"); !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("状态不存在时应返回 ErrTaskNotFound，实际: %v", err)
	}
}

func TestStreamEmitsSnapshotThenEvents(t *testing.T) {
	store := &fakeTaskStore{}
	svc := NewService(store)

	// 任务已入队，先写入一个中间态快照
	if err := store.Update(context.Background(), queue.NewStatus("task-1", queue.StatusPending, "已排队"), time.Minute); err != nil {
		t.Fatalf("写入状态失败: %v", err)
	}

	var got []string
	err := svc.Stream(context.Background(), "task-1", 2*time.Second, func(st queue.TaskStatus) error {
		got = append(got, st.Status)
		// 收到中间态后推进到终态，模拟 Worker 完成发信
		if st.Status == queue.StatusPending {
			_ = store.Update(context.Background(), queue.NewStatus("task-1", queue.StatusSuccess, "已发送"), time.Minute)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("订阅进度失败: %v", err)
	}

	want := []string{queue.StatusPending, queue.StatusSuccess}
	if len(got) != len(want) {
		t.Fatalf("事件序列不符，期望 %v，实际 %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("第 %d 个事件期望 %s，实际 %s", i, want[i], got[i])
		}
	}
}

func TestStreamStopsAtTerminalSnapshot(t *testing.T) {
	store := &fakeTaskStore{current: &queue.TaskStatus{TaskID: "task-2", Status: queue.StatusSuccess}}
	svc := NewService(store)

	// 订阅前的终态快照：应收到一个事件后立即返回，不再建立订阅
	count := 0
	err := svc.Stream(context.Background(), "task-2", time.Second, func(queue.TaskStatus) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatalf("订阅进度失败: %v", err)
	}
	if count != 1 {
		t.Errorf("终态快照应只推送一次，实际 %d 次", count)
	}
}

func TestStreamReturnsErrorWhenStoreUnavailable(t *testing.T) {
	svc := NewService(nil)
	if err := svc.Stream(context.Background(), "task-3", time.Second, func(queue.TaskStatus) error { return nil }); err == nil {
		t.Error("存储未初始化时应返回错误")
	}
}

func TestStreamStopsOnEmitError(t *testing.T) {
	store := &fakeTaskStore{current: &queue.TaskStatus{TaskID: "task-4", Status: queue.StatusPending}}
	svc := NewService(store)

	sentinel := errors.New("写入客户端失败")
	err := svc.Stream(context.Background(), "task-4", time.Second, func(queue.TaskStatus) error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Errorf("emit 返回的错误应向上传递，实际: %v", err)
	}
}
