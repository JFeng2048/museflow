package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hibiken/asynq"

	"github.com/museflow/user-service/internal/pkg/email"
	"github.com/museflow/user-service/internal/pkg/queue"
)

// ---- 测试替身 ----

// stubSender 记录发出的邮件，可按需返回错误。
type stubSender struct {
	msgs []email.Message
	err  error
}

func (s *stubSender) Send(_ context.Context, msg email.Message) error {
	if s.err != nil {
		return s.err
	}
	s.msgs = append(s.msgs, msg)
	return nil
}

// fakeStatusStore 内存版任务状态存储，记录状态变更顺序。
type fakeStatusStore struct {
	mu       sync.Mutex
	statuses []string
	latest   queue.TaskStatus
}

func (s *fakeStatusStore) Update(_ context.Context, st queue.TaskStatus, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statuses = append(s.statuses, st.Status)
	s.latest = st
	return nil
}

func (s *fakeStatusStore) Get(_ context.Context, taskID string) (*queue.TaskStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.latest.TaskID != taskID {
		return nil, nil
	}
	st := s.latest
	return &st, nil
}

// snapshot 返回状态变更序列的副本。
func (s *fakeStatusStore) snapshot() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.statuses))
	copy(out, s.statuses)
	return out
}

// withTaskContext 让 asynq 的 context 取值函数返回固定的任务 ID 与重试次数。
func withTaskContext(t *testing.T, taskID string, retry, maxRetry int) {
	t.Helper()
	origID, origRetry := taskIDFromContext, retryStateFromCtx
	t.Cleanup(func() {
		taskIDFromContext, retryStateFromCtx = origID, origRetry
	})
	taskIDFromContext = func(context.Context) (string, bool) { return taskID, true }
	retryStateFromCtx = func(context.Context) (int, int) { return retry, maxRetry }
}

func newTestHandler(t *testing.T) (*EmailHandler, *stubSender, *fakeStatusStore) {
	t.Helper()
	renderer, err := email.NewRenderer()
	if err != nil {
		t.Fatalf("构造模板渲染器失败: %v", err)
	}
	sender := &stubSender{}
	store := &fakeStatusStore{}
	h := NewEmailHandler(sender, renderer, EmailHandlerConfig{StatusStore: store, StatusTTL: time.Minute})
	return h, sender, store
}

// ---- 用例 ----

func TestProcessVerifyCodeSendsMailAndMarksSuccess(t *testing.T) {
	h, sender, store := newTestHandler(t)
	withTaskContext(t, "task-1", 0, 3)

	payload, err := json.Marshal(queue.EmailVerifyCodePayload{
		To:      "author@museflow.ai",
		Code:    "123456",
		Scene:   "register",
		Purpose: "注册 MuseFlow 账号",
		TTL:     600,
	})
	if err != nil {
		t.Fatalf("序列化载荷失败: %v", err)
	}

	if err := h.ProcessTask(context.Background(), asynq.NewTask(queue.TypeEmailVerifyCode, payload)); err != nil {
		t.Fatalf("处理任务失败: %v", err)
	}

	if len(sender.msgs) != 1 {
		t.Fatalf("应发出 1 封邮件，实际 %d 封", len(sender.msgs))
	}
	msg := sender.msgs[0]
	if msg.To != "author@museflow.ai" {
		t.Errorf("收件人不符: %s", msg.To)
	}
	// 验证码应同时出现在 HTML 与纯文本正文中（Worker 会按 3 位一组格式化）
	if !strings.Contains(msg.HTML, "123 456") || !strings.Contains(msg.Text, "123 456") {
		t.Errorf("正文未包含验证码，HTML=%q Text=%q", msg.HTML, msg.Text)
	}
	if msg.HTML == "" || msg.Text == "" {
		t.Error("HTML 与纯文本正文都应生成")
	}

	// 状态流转：sending -> success
	got := store.snapshot()
	want := []string{queue.StatusSending, queue.StatusSuccess}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("状态流转不符，期望 %v，实际 %v", want, got)
	}
}

func TestProcessVerifyCodeMarksRetryingBeforeFinalFailure(t *testing.T) {
	h, _, store := newTestHandler(t)
	withTaskContext(t, "task-2", 1, 3)

	// 用不可达的 SMTP 配置制造发送失败
	h.sender = &stubSender{err: errors.New("连接 SMTP 超时")}

	payload, _ := json.Marshal(queue.EmailVerifyCodePayload{
		To: "author@museflow.ai", Code: "654321", Scene: "reset_password", TTL: 600,
	})
	err := h.ProcessTask(context.Background(), asynq.NewTask(queue.TypeEmailVerifyCode, payload))
	if err == nil {
		t.Fatal("发送失败时应返回错误以触发 asynq 重试")
	}

	// 仍有重试额度时只标记 retrying，不该提前告知用户失败
	want := queue.StatusSending + "," + queue.StatusRetrying
	if got := store.snapshot(); strings.Join(got, ",") != want {
		t.Errorf("重试期间状态流转应为 %s，实际: %v", want, got)
	}
}

func TestProcessVerifyCodeMarksFailedAfterRetriesExhausted(t *testing.T) {
	h, _, store := newTestHandler(t)
	withTaskContext(t, "task-3", 3, 3)
	h.sender = &stubSender{err: errors.New("连接 SMTP 超时")}

	payload, _ := json.Marshal(queue.EmailVerifyCodePayload{
		To: "author@museflow.ai", Code: "654321", Scene: "login", TTL: 600,
	})
	if err := h.ProcessTask(context.Background(), asynq.NewTask(queue.TypeEmailVerifyCode, payload)); err == nil {
		t.Fatal("发送失败时应返回错误")
	}

	want := queue.StatusSending + "," + queue.StatusFailed
	if got := store.snapshot(); strings.Join(got, ",") != want {
		t.Errorf("重试耗尽后状态流转应为 %s，实际: %v", want, got)
	}
}

func TestProcessVerifyCodeSkipsRetryOnBrokenPayload(t *testing.T) {
	h, _, store := newTestHandler(t)
	withTaskContext(t, "task-4", 0, 3)

	err := h.ProcessTask(context.Background(), asynq.NewTask(queue.TypeEmailVerifyCode, []byte("不是 JSON")))
	if err == nil {
		t.Fatal("载荷损坏时应返回错误")
	}
	if !errors.Is(err, asynq.SkipRetry) {
		t.Errorf("载荷损坏应标记为不再重试，实际: %v", err)
	}
	if got := store.snapshot(); strings.Join(got, ",") != queue.StatusFailed {
		t.Errorf("载荷损坏应标记 failed，实际: %v", got)
	}
}

func TestProcessWelcomeSendsGreetingMail(t *testing.T) {
	h, sender, _ := newTestHandler(t)
	withTaskContext(t, "task-5", 0, 3)

	payload, _ := json.Marshal(queue.EmailWelcomePayload{To: "new@museflow.ai", Nickname: "墨流"})
	if err := h.ProcessTask(context.Background(), asynq.NewTask(queue.TypeEmailWelcome, payload)); err != nil {
		t.Fatalf("处理欢迎邮件失败: %v", err)
	}
	if len(sender.msgs) != 1 {
		t.Fatalf("应发出 1 封欢迎邮件，实际 %d 封", len(sender.msgs))
	}
	if !strings.Contains(sender.msgs[0].Text, "墨流") {
		t.Errorf("欢迎邮件应包含昵称，实际: %s", sender.msgs[0].Text)
	}
}

func TestProcessTaskRejectsUnknownType(t *testing.T) {
	h, _, _ := newTestHandler(t)
	withTaskContext(t, "task-6", 0, 3)

	if err := h.ProcessTask(context.Background(), asynq.NewTask("email:unknown", []byte("{}"))); err == nil {
		t.Fatal("未知任务类型应返回错误")
	}
}
