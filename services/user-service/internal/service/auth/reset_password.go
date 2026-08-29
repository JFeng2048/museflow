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
)

// 密码重置业务错误。
var (
	// ErrCodeNotSent 验证码不存在或已过期，需重新获取。
	ErrCodeNotSent = errors.New("验证码已过期或尚未获取，请重新发送验证码")
	// ErrCodeMismatch 验证码错误。
	ErrCodeMismatch = errors.New("验证码不正确，请检查后重新输入")
	// ErrResendTooSoon 发送过于频繁，处于冷却期内。
	ErrResendTooSoon = errors.New("验证码已发送，请稍候再试")
)

// resetScene 验证码场景标识，用于区分不同用途的验证码。
const resetScene = "reset_password"

// ResetServiceConfig 密码重置相关配置。
type ResetServiceConfig struct {
	CodeTTL      time.Duration // 验证码有效期
	CodeLength   int           // 验证码位数
	CodeResendCD time.Duration // 重发冷却
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
