package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/museflow/pkg/logger"
	"github.com/museflow/user-service/internal/model"
	"github.com/museflow/user-service/internal/service/audit"
	"github.com/museflow/user-service/internal/service/dto"
	"github.com/museflow/user-service/internal/service/mfa"
)

// 2FA 业务错误。
var (
	// ErrMFANotEnabled 用户尚未开启 2FA。
	ErrMFANotEnabled = errors.New("尚未开启双因素认证")
	// ErrMFAAlreadyEnabled 用户已开启 2FA，需先关闭再重新设置。
	ErrMFAAlreadyEnabled = errors.New("已开启双因素认证")
	// ErrMFASecretMissing 尚未生成密钥，需先调用 SetupMFA。
	ErrMFASecretMissing = errors.New("尚未生成双因素认证密钥")
	// ErrMFACodeInvalid 验证码或恢复码无效。
	ErrMFACodeInvalid = errors.New("验证码无效")
	// ErrMFATicketInvalid 2FA 中间票据无效或已过期。
	ErrMFATicketInvalid = errors.New("登录会话已过期，请重新登录")
)

// MFASetupResult 2FA 设置结果（密钥与绑定 URL，此时尚未启用）。
type MFASetupResult struct {
	Secret     string
	OtpauthURL string
}

// SetupMFA 生成 TOTP 密钥并返回绑定 URL。
//
// 此时只是暂存密钥（mfa_enabled 仍为 false），
// 需用户提交一次正确验证码后调用 VerifyMFA 才正式启用，
// 避免用户扫码失败后账号被锁定在「已开启但无法验证」的状态。
func (s *AuthService) SetupMFA(ctx context.Context, userUUID string) (*MFASetupResult, error) {
	id, err := uuid.Parse(userUUID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	u, err := s.users.FindByUUID(ctx, id)
	if err != nil {
		return nil, err
	}
	if u.MFAEnabled {
		return nil, ErrMFAAlreadyEnabled
	}

	secret, err := mfa.GenerateSecret(u.Email, s.mfaCfg.Issuer)
	if err != nil {
		return nil, err
	}
	if err := s.users.SaveMFASecret(ctx, id, secret.Raw); err != nil {
		return nil, fmt.Errorf("保存 2FA 密钥失败: %w", err)
	}

	s.audit.Record(ctx, audit.Entry{
		UserUUID:   userUUID,
		Action:     model.AuditActionMFASetup,
		Resource:   model.AuditResourceUser,
		ResourceID: userUUID,
	})

	return &MFASetupResult{Secret: secret.Raw, OtpauthURL: secret.URL}, nil
}

// VerifyMFA 验证验证码并正式启用 2FA，生成 8 个恢复码。
//
// 恢复码明文仅在本次返回，服务端不做持久化明文（此处按需求以明文存储于
// mfa_recovery_codes 数组，生产建议改为存储 bcrypt 哈希）。
func (s *AuthService) VerifyMFA(ctx context.Context, userUUID, code string) ([]string, error) {
	id, err := uuid.Parse(userUUID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	u, err := s.users.FindByUUID(ctx, id)
	if err != nil {
		return nil, err
	}
	if u.MFAEnabled {
		return nil, ErrMFAAlreadyEnabled
	}
	if u.MFASecret == nil || *u.MFASecret == "" {
		return nil, ErrMFASecretMissing
	}
	if !mfa.ValidateCode(code, *u.MFASecret, s.mfaCfg.CodeSkew) {
		s.recordMFAVerifyFail(ctx, userUUID, "enable")
		return nil, ErrMFACodeInvalid
	}

	codes, err := mfa.GenerateRecoveryCodes(s.mfaCfg.RecoveryCodeCount, s.mfaCfg.RecoveryCodeLength)
	if err != nil {
		return nil, err
	}
	if err := s.users.EnableMFA(ctx, id, codes); err != nil {
		return nil, fmt.Errorf("启用 2FA 失败: %w", err)
	}

	s.audit.Record(ctx, audit.Entry{
		UserUUID:   userUUID,
		Action:     model.AuditActionMFAEnable,
		Resource:   model.AuditResourceUser,
		ResourceID: userUUID,
	})

	return codes, nil
}

// DisableMFA 验证验证码后关闭 2FA，清空密钥与恢复码。
func (s *AuthService) DisableMFA(ctx context.Context, userUUID, code string) error {
	id, err := uuid.Parse(userUUID)
	if err != nil {
		return ErrUserNotFound
	}

	u, err := s.users.FindByUUID(ctx, id)
	if err != nil {
		return err
	}
	if !u.MFAEnabled {
		return ErrMFANotEnabled
	}
	if u.MFASecret == nil || *u.MFASecret == "" {
		return ErrMFASecretMissing
	}
	// 允许用验证码或恢复码关闭
	if !mfa.ValidateCode(code, *u.MFASecret, s.mfaCfg.CodeSkew) && !s.consumeRecoveryCode(ctx, u, code) {
		s.recordMFAVerifyFail(ctx, userUUID, "disable")
		return ErrMFACodeInvalid
	}

	if err := s.users.DisableMFA(ctx, id); err != nil {
		return fmt.Errorf("关闭 2FA 失败: %w", err)
	}

	s.audit.Record(ctx, audit.Entry{
		UserUUID:   userUUID,
		Action:     model.AuditActionMFADisable,
		Resource:   model.AuditResourceUser,
		ResourceID: userUUID,
	})
	return nil
}

// RegenerateRecoveryCodes 验证验证码后重新生成 8 个恢复码，替换旧码。
func (s *AuthService) RegenerateRecoveryCodes(ctx context.Context, userUUID, code string) ([]string, error) {
	id, err := uuid.Parse(userUUID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	u, err := s.users.FindByUUID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !u.MFAEnabled {
		return nil, ErrMFANotEnabled
	}
	if u.MFASecret == nil || *u.MFASecret == "" {
		return nil, ErrMFASecretMissing
	}
	if !mfa.ValidateCode(code, *u.MFASecret, s.mfaCfg.CodeSkew) && !s.consumeRecoveryCode(ctx, u, code) {
		s.recordMFAVerifyFail(ctx, userUUID, "regen_codes")
		return nil, ErrMFACodeInvalid
	}

	codes, err := mfa.GenerateRecoveryCodes(s.mfaCfg.RecoveryCodeCount, s.mfaCfg.RecoveryCodeLength)
	if err != nil {
		return nil, err
	}
	if err := s.users.UpdateRecoveryCodes(ctx, id, codes); err != nil {
		return nil, fmt.Errorf("重新生成恢复码失败: %w", err)
	}

	s.audit.Record(ctx, audit.Entry{
		UserUUID:   userUUID,
		Action:     model.AuditActionMFARegenCodes,
		Resource:   model.AuditResourceUser,
		ResourceID: userUUID,
	})

	return codes, nil
}

// MFAStatus 当前用户的 2FA 状态。
type MFAStatus struct {
	Enabled bool
	// RemainingRecoveryCodes 剩余可用恢复码数量。
	RemainingRecoveryCodes int
}

// GetMFAStatus 查询用户 2FA 开启状态与剩余恢复码数量。
func (s *AuthService) GetMFAStatus(ctx context.Context, userUUID string) (*MFAStatus, error) {
	id, err := uuid.Parse(userUUID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	u, err := s.users.FindByUUID(ctx, id)
	if err != nil {
		return nil, err
	}

	remaining := 0
	if u.MFARecoveryCodes != nil {
		remaining = len(u.MFARecoveryCodes)
	}
	return &MFAStatus{Enabled: u.MFAEnabled, RemainingRecoveryCodes: remaining}, nil
}

// VerifyMFALogin 登录流程的 2FA 二次验证：校验票据与验证码（或恢复码），
// 通过后签发双令牌。
//
// 恢复码一次性使用：命中后立即从库中剔除。
// 返回登录结果，并使用恢复码时置 UsedRecoveryCode=true，
// 由上层提示用户尽快重新生成恢复码。
func (s *AuthService) VerifyMFALogin(ctx context.Context, ticket, code string, dev dto.Device) (*LoginResult, bool, error) {
	claims, err := s.tm.ParseMFATicket(ticket)
	if err != nil {
		return nil, false, ErrMFATicketInvalid
	}

	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, false, ErrMFATicketInvalid
	}

	u, err := s.users.FindByUUID(ctx, id)
	if err != nil {
		return nil, false, err
	}
	if !u.MFAEnabled || u.MFASecret == nil || *u.MFASecret == "" {
		return nil, false, ErrMFANotEnabled
	}
	if u.Status != model.StatusNormal {
		return nil, false, ErrAccountUnavailable
	}

	// 优先按验证码校验，其次尝试恢复码
	usedRecovery := false
	if mfa.ValidateCode(code, *u.MFASecret, s.mfaCfg.CodeSkew) {
		// 验证码通过
	} else if s.consumeRecoveryCode(ctx, u, code) {
		usedRecovery = true
	} else {
		s.recordMFAVerifyFail(ctx, u.UUID.String(), "login")
		return nil, false, ErrMFACodeInvalid
	}

	pair, err := s.issueTokens(ctx, u.UUID.String(), dev)
	if err != nil {
		return nil, false, err
	}

	if err := s.users.UpdateLoginInfo(ctx, u.UUID, dev.IP, dev.DeviceName); err != nil {
		logger.WarnContext(ctx, "更新登录信息失败", logger.UserUUID(u.UUID.String()), logger.Err(err))
	}
	if s.rbac != nil {
		if _, perr := s.rbac.GetUserPermissions(ctx, u.UUID); perr != nil {
			logger.WarnContext(ctx, "写入用户权限缓存失败", logger.UserUUID(u.UUID.String()), logger.Err(perr))
		}
	}

	action := model.AuditActionLogin
	if usedRecovery {
		action = model.AuditActionMFARecovery
	}
	s.audit.Record(ctx, audit.Entry{
		UserUUID:   u.UUID.String(),
		Action:     action,
		Resource:   model.AuditResourceAuth,
		ResourceID: u.UUID.String(),
		IP:         dev.IP,
		UserAgent:  dev.UserAgent,
		Detail:     map[string]string{"device_id": dev.DeviceID, "mfa": "true"},
	})

	return &LoginResult{TokenPair: pair, User: u}, usedRecovery, nil
}

// consumeRecoveryCode 尝试消费一个恢复码：命中则移除并返回 true。
// 未命中返回 false，不改变数据。
func (s *AuthService) consumeRecoveryCode(ctx context.Context, u *model.User, code string) bool {
	if u.MFARecoveryCodes == nil || len(u.MFARecoveryCodes) == 0 {
		return false
	}
	idx := mfa.MatchRecoveryCode(code, u.MFARecoveryCodes)
	if idx < 0 {
		return false
	}

	// 移除已使用的恢复码（一次性）
	rest := make([]string, 0, len(u.MFARecoveryCodes)-1)
	rest = append(rest, u.MFARecoveryCodes[:idx]...)
	rest = append(rest, u.MFARecoveryCodes[idx+1:]...)

	if err := s.users.UpdateRecoveryCodes(ctx, u.UUID, rest); err != nil {
		logger.WarnContext(ctx, "移除已使用的恢复码失败", logger.UserUUID(u.UUID.String()), logger.Err(err))
		// 移除失败仍允许本次登录通过，避免用户被彻底锁死
		return true
	}
	// 同步更新内存中的用户对象，便于同一次请求内后续判断
	u.MFARecoveryCodes = rest
	return true
}

// recordMFAVerifyFail 记录 2FA 校验失败审计。
func (s *AuthService) recordMFAVerifyFail(ctx context.Context, userUUID, scene string) {
	s.audit.Record(ctx, audit.Entry{
		UserUUID:   userUUID,
		Action:     model.AuditActionMFAVerifyFail,
		Resource:   model.AuditResourceAuth,
		ResourceID: userUUID,
		Detail:     map[string]string{"scene": scene},
	})
}

// MFAConfig 2FA 相关配置。
type MFAConfig struct {
	// Issuer 发行方名称，展示在验证器 App 中。
	Issuer string
	// CodeSkew 允许的时钟偏移步数（每步 30 秒），默认 1。
	CodeSkew int
	// RecoveryCodeCount 恢复码数量，默认 8。
	RecoveryCodeCount int
	// RecoveryCodeLength 单个恢复码长度，默认 10。
	RecoveryCodeLength int
}

// applyDefaults 补齐 2FA 配置的默认值。
func (c *MFAConfig) applyDefaults() {
	if c.Issuer == "" {
		c.Issuer = mfa.DefaultIssuer
	}
	if c.CodeSkew <= 0 {
		c.CodeSkew = 1
	}
	if c.RecoveryCodeCount <= 0 {
		c.RecoveryCodeCount = mfa.DefaultRecoveryCodeCount
	}
	if c.RecoveryCodeLength <= 0 {
		c.RecoveryCodeLength = mfa.DefaultRecoveryCodeLength
	}
}
