package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/museflow/user-service/internal/service/token"
)

// ---- 测试替身 ----

type fakeCodeStore struct {
	codes map[string]string
	locks map[string]bool
}

func newFakeCodeStore() *fakeCodeStore {
	return &fakeCodeStore{codes: map[string]string{}, locks: map[string]bool{}}
}

func (s *fakeCodeStore) SaveCode(_ context.Context, scene, target, code string, _ time.Duration) error {
	s.codes[scene+":"+target] = code
	return nil
}

func (s *fakeCodeStore) GetCode(_ context.Context, scene, target string) (string, error) {
	return s.codes[scene+":"+target], nil
}

func (s *fakeCodeStore) DeleteCode(_ context.Context, scene, target string) error {
	delete(s.codes, scene+":"+target)
	return nil
}

func (s *fakeCodeStore) TryLockResend(_ context.Context, scene, target string, _ time.Duration) (bool, error) {
	k := scene + ":" + target
	if s.locks[k] {
		return false, nil
	}
	s.locks[k] = true
	return true, nil
}

// stubMailer 记录邮件内容，不真实发送。
type stubMailer struct {
	sentTo   string
	sentBody string
}

func (m *stubMailer) Send(to, subject, body string) error {
	m.sentTo = to
	m.sentBody = body
	return nil
}

// testMFAConfig 返回测试用 2FA 配置（使用默认值）。
func testMFAConfig() MFAConfig { return MFAConfig{} }

// testEmailCodeConfig 返回测试用邮箱验证码配置（使用默认值）。
func testEmailCodeConfig() EmailCodeConfig { return EmailCodeConfig{} }

func testResetConfig() ResetServiceConfig {
	return ResetServiceConfig{
		CodeTTL:      10 * time.Minute,
		CodeLength:   6,
		CodeResendCD: 0, // 关闭冷却，避免用例互相干扰
	}
}

// newResetTestService 构造带验证码与邮件桩的测试服务。
func newResetTestService() (*AuthService, *fakeCodeStore, *stubMailer) {
	store := newFakeCodeStore()
	mailer := &stubMailer{}
	tm := token.NewTokenManager("test-secret", time.Hour, time.Hour, 5*time.Minute)
	svc := NewAuthService(
		newFakeUserRepo(), newFakeTokenStore(), tm,
		nil, nil, nil, // rbac / audit / oauth 传 nil
		store, mailer, testResetConfig(), testEmailCodeConfig(), testMFAConfig(), bcrypt.MinCost,
	)
	return svc, store, mailer
}

// ---- 用例 ----

func TestResetPasswordRejectsWrongCode(t *testing.T) {
	svc, store, _ := newResetTestService()
	ctx := context.Background()

	registerOK(t, svc, store, "reset2@museflow.ai", "oldpass1234", "n")
	if err := svc.SendVerifyCode(ctx, "reset2@museflow.ai", "reset_password"); err != nil {
		t.Fatalf("发送验证码失败: %v", err)
	}
	// 取出正确验证码后改写为错误值
	if err := store.SaveCode(ctx, "reset_password", "reset2@museflow.ai", "000000", time.Minute); err != nil {
		t.Fatalf("替换验证码失败: %v", err)
	}

	if err := svc.ResetPassword(ctx, "reset2@museflow.ai", "111111", "newpass5678"); !errors.Is(err, ErrCodeMismatch) {
		t.Errorf("错误验证码应被拒绝，实际: %v", err)
	}
}

func TestResetPasswordRejectsMissingCode(t *testing.T) {
	svc, codes, _ := newResetTestService()
	ctx := context.Background()

	registerOK(t, svc, codes, "reset3@museflow.ai", "oldpass1234", "n")

	// 从未发送过验证码 -> 应提示重新获取
	if err := svc.ResetPassword(ctx, "reset3@museflow.ai", "123456", "newpass5678"); !errors.Is(err, ErrCodeNotSent) {
		t.Errorf("未发送验证码时应返回重新获取，实际: %v", err)
	}
}

func TestResetPasswordConsumesCodeAndAllowsNewLogin(t *testing.T) {
	svc, store, _ := newResetTestService()
	ctx := context.Background()

	registerOK(t, svc, store, "reset4@museflow.ai", "oldpass1234", "n")
	if err := svc.SendVerifyCode(ctx, "reset4@museflow.ai", "reset_password"); err != nil {
		t.Fatalf("发送验证码失败: %v", err)
	}
	code, _ := store.GetCode(ctx, "reset_password", "reset4@museflow.ai")

	if err := svc.ResetPassword(ctx, "reset4@museflow.ai", code, "newpass5678"); err != nil {
		t.Fatalf("重置密码失败: %v", err)
	}

	// 验证码一次性：重用应失败
	if err := svc.ResetPassword(ctx, "reset4@museflow.ai", code, "another9999"); !errors.Is(err, ErrCodeNotSent) {
		t.Errorf("验证码不应可重复使用，实际: %v", err)
	}

	// 新密码可登录，旧密码失效
	if _, err := svc.Login(ctx, "reset4@museflow.ai", "newpass5678", testDevice()); err != nil {
		t.Errorf("新密码无法登录: %v", err)
	}
	if _, err := svc.Login(ctx, "reset4@museflow.ai", "oldpass1234", testDevice()); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("旧密码仍可登录: %v", err)
	}
}

func TestSendVerifyCodeEnforcesResendCooldown(t *testing.T) {
	ctx := context.Background()

	// 单独构造一个带冷却的服务，避免影响其他用例
	codes := newFakeCodeStore()
	withCD := NewAuthService(
		newFakeUserRepo(), newFakeTokenStore(),
		token.NewTokenManager("test-secret", time.Hour, time.Hour, 5*time.Minute),
		nil, nil, nil, codes, &stubMailer{},
		ResetServiceConfig{CodeTTL: 10 * time.Minute, CodeLength: 6, CodeResendCD: time.Minute},
		testEmailCodeConfig(), testMFAConfig(), bcrypt.MinCost,
	)

	if err := withCD.SendVerifyCode(ctx, "cooldown@museflow.ai", "reset_password"); err != nil {
		t.Fatalf("首次发送失败: %v", err)
	}
	if err := withCD.SendVerifyCode(ctx, "cooldown@museflow.ai", "reset_password"); !errors.Is(err, ErrResendTooSoon) {
		t.Errorf("冷却期内重复发送应被拒绝，实际: %v", err)
	}
}
