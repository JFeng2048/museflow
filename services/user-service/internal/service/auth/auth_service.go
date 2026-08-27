// Package auth 实现 user-service 的认证业务逻辑层。
package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/museflow/pkg/logger"
	"github.com/museflow/user-service/internal/model"
	"github.com/museflow/user-service/internal/repository"
	"github.com/museflow/user-service/internal/service/dto"
	"github.com/museflow/user-service/internal/service/token"
)

// 业务错误，由 gRPC 处理器映射为对应的 status code。
var (
	ErrEmailExists        = errors.New("邮箱已被注册")
	ErrInvalidCredentials = errors.New("邮箱或密码错误")
	ErrUserNotFound       = errors.New("用户不存在")
	ErrAccountUnavailable = errors.New("账号状态异常")
	ErrTokenInvalid       = errors.New("令牌无效或已失效")
	ErrDeviceMismatch     = errors.New("设备校验失败")
)

// bcrypt 最长只处理 72 字节，超长部分会被静默截断，这里显式拒绝以避免安全误解。
const maxPasswordBytes = 72

// AuthService 认证业务逻辑。
type AuthService struct {
	users      repository.UserRepository
	tokens     repository.TokenStore
	tm         *token.TokenManager
	bcryptCost int
}

// NewAuthService 构造认证服务。
func NewAuthService(
	users repository.UserRepository,
	tokens repository.TokenStore,
	tm *token.TokenManager,
	bcryptCost int,
) *AuthService {
	return &AuthService{users: users, tokens: tokens, tm: tm, bcryptCost: bcryptCost}
}

// Register 用户注册：校验邮箱唯一性后以 bcrypt 存储密码。
func (s *AuthService) Register(ctx context.Context, email, password, nickname string) (*model.User, error) {
	email = normalizeEmail(email)
	if email == "" || password == "" {
		return nil, fmt.Errorf("邮箱和密码不能为空")
	}
	if len(password) > maxPasswordBytes {
		return nil, fmt.Errorf("密码长度不能超过 %d 字节", maxPasswordBytes)
	}

	exists, err := s.users.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("查询邮箱失败: %w", err)
	}
	if exists {
		return nil, ErrEmailExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	// 昵称为空时用邮箱前缀兜底（users.nickname 为 NOT NULL）
	if nickname == "" {
		nickname = email[:strings.Index(email, "@")]
	}

	hashStr := string(hash)
	u := &model.User{
		UUID:         uuid.New(),
		Email:        email,
		PasswordHash: &hashStr,
		Nickname:     nickname,
		Status:       model.StatusNormal,
	}

	if err := s.users.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}
	return u, nil
}

// Login 校验密码并签发双令牌。
//
// 说明：按当前需求，登录流程不使用 login_fail_count / locked_until，
// 仅做密码比对与 status 读取（不阻断）。
func (s *AuthService) Login(ctx context.Context, email, password string, dev dto.Device) (*dto.TokenPair, *model.User, error) {
	email = normalizeEmail(email)

	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			// 不区分“用户不存在”与“密码错误”，避免邮箱枚举
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 第三方登录用户可能没有密码，此时不允许密码登录
	if u.PasswordHash == nil || *u.PasswordHash == "" {
		return nil, nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	pair, err := s.issueTokens(ctx, u.UUID.String(), dev)
	if err != nil {
		return nil, nil, err
	}

	// 更新最后登录信息失败不阻断登录流程，仅记录日志
	if err := s.users.UpdateLoginInfo(ctx, u.UUID, dev.IP, dev.DeviceName); err != nil {
		logger.WarnContext(ctx, "更新登录信息失败", logger.UserUUID(u.UUID.String()), logger.Err(err))
	}

	return pair, u, nil
}

// issueTokens 生成 tokenId、签发双令牌并写入 Redis 白名单与设备列表。
func (s *AuthService) issueTokens(ctx context.Context, userUUID string, dev dto.Device) (*dto.TokenPair, error) {
	// 客户端未携带 device_id 时由服务端生成
	deviceID := dev.DeviceID
	if deviceID == "" {
		deviceID = uuid.NewString()
	}

	tokenID := uuid.NewString()
	fingerprint := token.DeviceFingerprint(deviceID, dev.UserAgent, dev.IP)

	accessToken, err := s.tm.GenerateAccess(userUUID, uuid.NewString())
	if err != nil {
		return nil, fmt.Errorf("签发访问令牌失败: %w", err)
	}
	refreshToken, err := s.tm.GenerateRefresh(userUUID, tokenID, deviceID, fingerprint)
	if err != nil {
		return nil, fmt.Errorf("签发刷新令牌失败: %w", err)
	}

	refreshTTL := s.tm.RefreshTTL()
	if err := s.tokens.SetRefreshValid(ctx, tokenID, refreshTTL); err != nil {
		return nil, fmt.Errorf("写入刷新令牌白名单失败: %w", err)
	}

	meta := repository.TokenMeta{
		TokenID:         tokenID,
		DeviceID:        deviceID,
		DeviceName:      dev.DeviceName,
		LoginTime:       time.Now(),
		LastRefreshTime: time.Now(),
	}
	if err := s.tokens.AppendUserToken(ctx, userUUID, meta, refreshTTL); err != nil {
		logger.WarnContext(ctx, "写入用户会话列表失败", logger.UserUUID(userUUID), logger.Err(err))
	}

	return &dto.TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		DeviceID:         deviceID,
		ExpiresIn:        int64(s.tm.AccessTTL().Seconds()),
		RefreshExpiresIn: int64(refreshTTL.Seconds()),
	}, nil
}

// Refresh 用 refresh token 换取新 access token，refresh 本身不轮换。
//
// 校验顺序：验签/类型 → 设备 ID 比对 → 设备指纹比对 → Redis 白名单。
func (s *AuthService) Refresh(ctx context.Context, refreshTokenStr string, dev dto.Device) (string, int64, error) {
	claims, err := s.tm.ParseRefresh(refreshTokenStr)
	if err != nil {
		return "", 0, ErrTokenInvalid
	}

	// 设备 ID 必须与 Cookie 中的一致
	if dev.DeviceID == "" || claims.DeviceID != dev.DeviceID {
		return "", 0, ErrDeviceMismatch
	}

	// 重新计算指纹并比对，防止令牌被窃取后在其他设备使用
	if token.DeviceFingerprint(dev.DeviceID, dev.UserAgent, dev.IP) != claims.DeviceFingerprint {
		return "", 0, ErrDeviceMismatch
	}

	valid, err := s.tokens.IsRefreshValid(ctx, claims.TokenID)
	if err != nil {
		return "", 0, fmt.Errorf("校验刷新令牌失败: %w", err)
	}
	if !valid {
		return "", 0, ErrTokenInvalid
	}

	accessToken, err := s.tm.GenerateAccess(claims.Subject, uuid.NewString())
	if err != nil {
		return "", 0, fmt.Errorf("签发访问令牌失败: %w", err)
	}

	if err := s.tokens.TouchUserToken(ctx, claims.Subject, claims.TokenID, s.tm.RefreshTTL()); err != nil {
		logger.WarnContext(ctx, "更新会话刷新时间失败", logger.UserUUID(claims.Subject), logger.Err(err))
	}

	return accessToken, int64(s.tm.AccessTTL().Seconds()), nil
}

// Logout 登出：删除 refresh 白名单 + 将 access token 加入黑名单。
//
// 两个令牌都尽最大努力处理：任一解析失败不影响另一个的吊销，
// 保证客户端即使只带了其中一个也能完成登出。
func (s *AuthService) Logout(ctx context.Context, accessToken, refreshTokenStr string) error {
	if refreshTokenStr != "" {
		if claims, err := s.tm.ParseRefresh(refreshTokenStr); err == nil {
			if err := s.tokens.DeleteRefreshValid(ctx, claims.TokenID); err != nil {
				return fmt.Errorf("删除刷新令牌白名单失败: %w", err)
			}
			if err := s.tokens.RemoveUserToken(ctx, claims.Subject, claims.TokenID); err != nil {
				logger.WarnContext(ctx, "移除用户会话失败", logger.UserUUID(claims.Subject), logger.Err(err))
			}
		}
	}

	if accessToken != "" {
		// 已过期的 token 解析会失败，此时无需入黑名单
		if claims, err := s.tm.ParseAccess(accessToken); err == nil && claims.ID != "" {
			ttl := time.Duration(0)
			if claims.ExpiresAt != nil {
				// 黑名单 TTL = 令牌剩余有效期，到期自动清理
				ttl = time.Until(claims.ExpiresAt.Time)
			}
			if err := s.tokens.BlacklistAccess(ctx, claims.ID, ttl); err != nil {
				return fmt.Errorf("写入访问令牌黑名单失败: %w", err)
			}
		}
	}

	return nil
}

// ValidateAccess 校验 access token 并返回用户 uuid，包含黑名单检查。
func (s *AuthService) ValidateAccess(ctx context.Context, accessToken string) (string, error) {
	claims, err := s.tm.ParseAccess(accessToken)
	if err != nil {
		return "", ErrTokenInvalid
	}

	if claims.ID != "" {
		blacklisted, err := s.tokens.IsAccessBlacklisted(ctx, claims.ID)
		if err != nil {
			return "", fmt.Errorf("校验令牌黑名单失败: %w", err)
		}
		if blacklisted {
			return "", ErrTokenInvalid
		}
	}

	return claims.Subject, nil
}

// GetProfile 按 uuid 查询用户信息。
func (s *AuthService) GetProfile(ctx context.Context, userUUID string) (*model.User, error) {
	id, err := uuid.Parse(userUUID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	u, err := s.users.FindByUUID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return u, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
