package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/museflow/pkg/logger"
	"github.com/museflow/user-service/internal/model"
	"github.com/museflow/user-service/internal/repository"
	"github.com/museflow/user-service/internal/service/audit"
	"github.com/museflow/user-service/internal/service/notify"
)

// 密码重置业务错误。
var (
	// ErrCodeNotSent 验证码不存在或已过期，需重新获取。
	ErrCodeNotSent = errors.New("验证码不存在或已过期，请重新获取")
	// ErrCodeMismatch 验证码错误。
	ErrCodeMismatch = errors.New("验证码错误")
	// ErrResendTooSoon 发送过于频繁，处于冷却期内。
	ErrResendTooSoon = errors.New("发送过于频繁，请稍后再试")
)

// resetScene 验证码场景标识，用于区分不同用途的验证码。
const resetScene = "reset_password"

// ResetServiceConfig 密码重置相关配置。
type ResetServiceConfig struct {
	CodeTTL      time.Duration // 验证码有效期
	CodeLength   int           // 验证码位数
	CodeResendCD time.Duration // 重发冷却
}

// SendResetCode 生成并向邮箱发送密码重置验证码。
//
// 安全考量：邮箱不存在时不返回错误（避免账号枚举），
// 但也不写入验证码（避免为不存在的账号消耗存储）。
func (s *AuthService) SendResetCode(ctx context.Context, email string) error {
	email = normalizeEmail(email)
	if email == "" {
		return fmt.Errorf("邮箱不能为空")
	}

	// 防重发：同一邮箱在冷却期内重复请求直接拒绝
	ok, err := s.codes.TryLockResend(ctx, resetScene, email, s.reset.CodeResendCD)
	if err != nil {
		// Redis 故障时放行，避免完全不可用（降级为不限制频率）
		logger.WarnContext(ctx, "验证码防重发检查失败，降级放行", logger.Err(err))
	} else if !ok {
		return ErrResendTooSoon
	}

	// 邮箱不存在：静默返回成功，不泄露账号是否存在
	if _, err := s.users.FindByEmail(ctx, email); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			logger.InfoContext(ctx, "密码重置请求邮箱不存在，静默忽略", "email", email)
			return nil
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}

	code, err := notify.GenerateNumericCode(s.reset.CodeLength)
	if err != nil {
		return err
	}
	if err := s.codes.SaveCode(ctx, resetScene, email, code, s.reset.CodeTTL); err != nil {
		return fmt.Errorf("保存验证码失败: %w", err)
	}

	// 邮件发送失败不影响验证码已生成，返回错误由调用方提示重试
	if err := s.mailer.Send(email, "MuseFlow 密码重置验证码", notify.ResetPasswordBody(code, s.reset.CodeTTL)); err != nil {
		logger.WarnContext(ctx, "发送重置密码邮件失败", "email", email, logger.Err(err))
		return fmt.Errorf("邮件发送失败，请稍后重试")
	}

	s.audit.Record(ctx, audit.Entry{
		Action:     model.AuditActionSendCode,
		Resource:   model.AuditResourceAuth,
		ResourceID: email,
		Detail:     map[string]string{"email": email},
	})

	return nil
}

// ResetPassword 校验验证码并重置密码。
//
// 成功后：更新密码哈希、删除验证码（防复用）、清理权限缓存、写审计日志。
func (s *AuthService) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	email = normalizeEmail(email)
	if email == "" || code == "" {
		return ErrCodeMismatch
	}
	if len(newPassword) > maxPasswordBytes {
		return fmt.Errorf("密码长度不能超过 %d 字节", maxPasswordBytes)
	}

	stored, err := s.codes.GetCode(ctx, resetScene, email)
	if err != nil {
		return fmt.Errorf("读取验证码失败: %w", err)
	}
	if stored == "" {
		return ErrCodeNotSent
	}
	// 常量时间比较，降低时序侧信道风险
	if !constantTimeEqual(stored, strings.TrimSpace(code)) {
		return ErrCodeMismatch
	}

	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}

	hash, err := bcryptGenerate(newPassword, s.bcryptCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}
	if err := s.users.UpdatePasswordHash(ctx, u.UUID, hash); err != nil {
		return fmt.Errorf("重置密码失败: %w", err)
	}

	// 验证码一次性使用
	if err := s.codes.DeleteCode(ctx, resetScene, email); err != nil {
		logger.WarnContext(ctx, "删除已使用验证码失败", logger.Err(err))
	}

	// 重置后清理权限缓存与登录失败锁定，确保新密码登录后状态干净
	if err := s.ClearUserCache(ctx, u.UUID.String()); err != nil {
		logger.WarnContext(ctx, "重置密码后清理权限缓存失败", logger.UserUUID(u.UUID.String()), logger.Err(err))
	}
	if err := s.users.ResetLoginFails(ctx, u.UUID); err != nil {
		logger.WarnContext(ctx, "重置密码后清理登录失败计数失败", logger.UserUUID(u.UUID.String()), logger.Err(err))
	}

	s.audit.Record(ctx, audit.Entry{
		UserUUID:   u.UUID.String(),
		Action:     model.AuditActionResetPwd,
		Resource:   model.AuditResourceUser,
		ResourceID: u.UUID.String(),
	})

	return nil
}

// bcryptGenerate 生成 bcrypt 密码哈希。
func bcryptGenerate(password string, cost int) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// constantTimeEqual 常量时间字符串比较。
func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := 0; i < len(a); i++ {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
