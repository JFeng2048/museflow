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
	"github.com/museflow/user-service/internal/pkg/queue"
	"github.com/museflow/user-service/internal/repository"
	"github.com/museflow/user-service/internal/service/audit"
	"github.com/museflow/user-service/internal/service/dto"
	"github.com/museflow/user-service/internal/service/oauth"
	"github.com/museflow/user-service/internal/service/rbac"
	"github.com/museflow/user-service/internal/service/token"
)

// 业务错误，由 gRPC 处理器映射为对应的 status code。
var (
	ErrEmailExists        = errors.New("邮箱已被注册")
	ErrInvalidCredentials = errors.New("邮箱或密码错误")
	ErrUserNotFound       = errors.New("用户不存在")
	ErrAccountUnavailable = errors.New("账号状态异常")
	ErrAccountLocked      = errors.New("账号已锁定，请稍后再试")
	ErrTokenInvalid       = errors.New("令牌无效或已失效")
	ErrDeviceMismatch     = errors.New("设备校验失败")
	ErrEmailAlreadyUsed   = errors.New("该邮箱已被其他账号使用，请换一个邮箱")
)

// bcrypt 最长只处理 72 字节，超长部分会被静默截断，这里显式拒绝以避免安全误解。
const maxPasswordBytes = 72

// TaskProducer 异步任务投递抽象（由 internal/pkg/queue.Client 实现）。
//
// 邮件属于慢速外部依赖，服务层只负责投递任务、不直接发信，
// 依赖抽象便于单元测试注入内存替身。
type TaskProducer interface {
	// EnqueueEmailVerifyCode 投递邮箱验证码任务，返回任务 ID。
	EnqueueEmailVerifyCode(ctx context.Context, p queue.EmailVerifyCodePayload) (string, error)
	// EnqueueEmailWelcome 投递欢迎邮件任务，返回任务 ID。
	EnqueueEmailWelcome(ctx context.Context, p queue.EmailWelcomePayload) (string, error)
}

// CaptchaVerifier 人机验证抽象（由 internal/pkg/turnstile.Client 实现）。
//
// 只有 Verify 一个方法，依赖抽象可让单测注入替身，不必真的访问 Cloudflare。
type CaptchaVerifier interface {
	Verify(ctx context.Context, token, remoteIP, expectedAction string) error
}

// SendVerifyCodeInput 发送邮箱验证码的入参。
//
// 单独成结构体而非逐个传参：除了 email/scene，还要带人机验证令牌与客户端 IP
// 用于服务端核验，位置参数容易传错。
type SendVerifyCodeInput struct {
	// Email 目标邮箱，内部会规范化为小写。
	Email string
	// Scene 业务场景：register/login/reset_password/change_email。
	Scene string
	// CaptchaToken Cloudflare Turnstile 令牌，一次性。
	CaptchaToken string
	// ClientIP 客户端 IP，随令牌一并提交给校验服务。
	ClientIP string
}

// AuthService 认证业务逻辑。
type AuthService struct {
	users      repository.UserRepository
	tokens     repository.TokenStore
	tm         *token.TokenManager
	rbac       *rbac.Service
	audit      *audit.Service
	oauth      *oauth.Service
	codes      repository.VerifyCodeStore // 验证码存储（密码重置 / 注册校验 / 验证码登录）
	producer   TaskProducer               // 异步任务投递（邮件等慢速操作）
	captcha    CaptchaVerifier            // 人机验证（防脚本刷验证码），未启用时为 nil
	reset      ResetServiceConfig         // 密码重置配置
	emailCfg   EmailCodeConfig            // 邮箱验证码（注册校验 / 验证码登录）配置
	mfaCfg     MFAConfig                  // 2FA 配置
	bcryptCost int
}

// NewAuthService 构造认证服务。
func NewAuthService(
	users repository.UserRepository,
	tokens repository.TokenStore,
	tm *token.TokenManager,
	rbacSvc *rbac.Service,
	auditSvc *audit.Service,
	oauthSvc *oauth.Service,
	codes repository.VerifyCodeStore,
	producer TaskProducer,
	captcha CaptchaVerifier,
	reset ResetServiceConfig,
	emailCfg EmailCodeConfig,
	mfaCfg MFAConfig,
	bcryptCost int,
) *AuthService {
	mfaCfg.applyDefaults()
	emailCfg.applyDefaults()
	return &AuthService{
		users:      users,
		tokens:     tokens,
		tm:         tm,
		rbac:       rbacSvc,
		audit:      auditSvc,
		oauth:      oauthSvc,
		codes:      codes,
		producer:   producer,
		captcha:    captcha,
		reset:      reset,
		emailCfg:   emailCfg,
		mfaCfg:     mfaCfg,
		bcryptCost: bcryptCost,
	}
}

// Register 用户注册：校验邮箱验证码后创建账号（邮箱直接标记为已验证），并授予默认角色 user。
//
// 注册前需先调用 SendVerifyCode{scene:"register"} 获取验证码，code 校验失败返回 ErrCodeMismatch。
func (s *AuthService) Register(ctx context.Context, email, password, nickname, code string) (*model.User, error) {
	email = normalizeEmail(email)
	if email == "" || password == "" {
		return nil, fmt.Errorf("邮箱和密码不能为空")
	}
	if len(password) > maxPasswordBytes {
		return nil, fmt.Errorf("密码长度不能超过 %d 字节", maxPasswordBytes)
	}

	// 先校验邮箱验证码，避免邮箱枚举（无效验证码统一返回验证码错误）
	saved, err := s.codes.GetCode(ctx, "register", email)
	if err != nil {
		return nil, fmt.Errorf("读取邮箱验证码失败: %w", err)
	}
	if saved == "" {
		return nil, ErrCodeNotSent
	}
	if !strings.EqualFold(saved, strings.TrimSpace(code)) {
		return nil, ErrCodeMismatch
	}
	// 校验通过立即删除，防止重复使用
	if err := s.codes.DeleteCode(ctx, "register", email); err != nil {
		logger.WarnContext(ctx, "删除已用邮箱验证码失败", "email", email, logger.Err(err))
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
		UUID:          uuid.New(),
		Email:         email,
		PasswordHash:  &hashStr,
		Nickname:      nickname,
		Status:        model.StatusNormal,
		EmailVerified: true,
	}

	if err := s.users.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	// 注册即授予默认角色（user），并预清权限缓存，待登录时加载
	if s.rbac != nil {
		if err := s.rbac.AssignRole(ctx, u.UUID, rbac.RoleUser, u.UUID); err != nil {
			logger.WarnContext(ctx, "注册授予默认角色失败", logger.UserUUID(u.UUID.String()), logger.Err(err))
		}
	}

	// 注册成功后投递欢迎邮件：属于通知类任务，失败只记日志不影响注册结果
	if s.producer != nil {
		if _, err := s.producer.EnqueueEmailWelcome(ctx, queue.EmailWelcomePayload{
			To:       u.Email,
			Nickname: u.Nickname,
		}); err != nil {
			logger.WarnContext(ctx, "投递欢迎邮件任务失败", logger.UserUUID(u.UUID.String()), logger.Err(err))
		}
	}

	s.audit.Record(ctx, audit.Entry{
		UserUUID:   u.UUID.String(),
		Action:     model.AuditActionRegister,
		Resource:   model.AuditResourceUser,
		ResourceID: u.UUID.String(),
		Detail:     map[string]string{"email": u.Email},
	})

	return u, nil
}

// LoginResult 登录结果。
//
// 分两种情形：
//   - 未开启 2FA：TokenPair 与 User 均非空，RequiresMFA 为 false，可直接返回令牌
//   - 已开启 2FA：RequiresMFA 为 true 且 MFATicket 非空，此时不下发令牌，
//     需用户提交验证码后调用 VerifyMFALogin 换取令牌
type LoginResult struct {
	TokenPair *dto.TokenPair
	User      *model.User
	// RequiresMFA 是否需要二次验证（账号已开启 2FA）。
	RequiresMFA bool
	// MFATicket 2FA 中间票据，仅 RequiresMFA 为 true 时有值。
	MFATicket string
}

// Login 校验密码并签发双令牌；成功后写入用户权限缓存。
// 包含登录失败锁定（5 次 / 15 分钟）与账号状态校验。
//
// 若用户已开启 2FA，则不下发令牌，而是返回 RequiresMFA=true 与中间票据。
func (s *AuthService) Login(ctx context.Context, email, password string, dev dto.Device) (*LoginResult, error) {
	email = normalizeEmail(email)

	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			// 不区分“用户不存在”与“密码错误”，避免邮箱枚举
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 账号锁定检查（locked_until 未过期则拒绝）
	if u.LockedUntil != nil && u.LockedUntil.After(time.Now()) {
		s.audit.Record(ctx, audit.Entry{
			UserUUID:   u.UUID.String(),
			Action:     model.AuditActionLoginFail,
			Resource:   model.AuditResourceAuth,
			ResourceID: u.UUID.String(),
			IP:         dev.IP,
			UserAgent:  dev.UserAgent,
			Detail:     map[string]string{"reason": "account_locked"},
		})
		return nil, ErrAccountLocked
	}

	// 第三方登录用户可能没有密码，此时不允许密码登录
	if u.PasswordHash == nil || *u.PasswordHash == "" {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(password)); err != nil {
		// 密码错误：累加失败计数，达到阈值锁定
		if _, failErr := s.users.IncrementLoginFails(ctx, email); failErr != nil {
			logger.WarnContext(ctx, "累加登录失败计数失败", logger.UserUUID(u.UUID.String()), logger.Err(failErr))
		}
		s.audit.Record(ctx, audit.Entry{
			UserUUID:   u.UUID.String(),
			Action:     model.AuditActionLoginFail,
			Resource:   model.AuditResourceAuth,
			ResourceID: u.UUID.String(),
			IP:         dev.IP,
			UserAgent:  dev.UserAgent,
			Detail:     map[string]string{"reason": "bad_password"},
		})
		return nil, ErrInvalidCredentials
	}

	// 密码校验通过：重置失败计数（后续成功与否都不再累加）
	if err := s.users.ResetLoginFails(ctx, u.UUID); err != nil {
		logger.WarnContext(ctx, "重置登录失败计数失败", logger.UserUUID(u.UUID.String()), logger.Err(err))
	}

	// 已开启 2FA：不下发令牌，签发中间票据等待二次验证
	if u.MFAEnabled && u.MFASecret != nil && *u.MFASecret != "" {
		ticket, err := s.tm.GenerateMFATicket(u.UUID.String(), uuid.NewString())
		if err != nil {
			return nil, fmt.Errorf("签发 2FA 票据失败: %w", err)
		}
		s.audit.Record(ctx, audit.Entry{
			UserUUID:   u.UUID.String(),
			Action:     model.AuditActionMFAChallenge,
			Resource:   model.AuditResourceAuth,
			ResourceID: u.UUID.String(),
			IP:         dev.IP,
			UserAgent:  dev.UserAgent,
		})
		return &LoginResult{User: u, RequiresMFA: true, MFATicket: ticket}, nil
	}

	pair, err := s.issueTokens(ctx, u.UUID.String(), dev)
	if err != nil {
		return nil, err
	}

	// 登录成功：更新登录信息 + 写入权限缓存
	if err := s.users.UpdateLoginInfo(ctx, u.UUID, dev.IP, dev.DeviceName); err != nil {
		logger.WarnContext(ctx, "更新登录信息失败", logger.UserUUID(u.UUID.String()), logger.Err(err))
	}
	if s.rbac != nil {
		if _, perr := s.rbac.GetUserPermissions(ctx, u.UUID); perr != nil {
			logger.WarnContext(ctx, "写入用户权限缓存失败", logger.UserUUID(u.UUID.String()), logger.Err(perr))
		}
	}

	s.audit.Record(ctx, audit.Entry{
		UserUUID:   u.UUID.String(),
		Action:     model.AuditActionLogin,
		Resource:   model.AuditResourceAuth,
		ResourceID: u.UUID.String(),
		IP:         dev.IP,
		UserAgent:  dev.UserAgent,
		Detail:     map[string]string{"device_id": dev.DeviceID},
	})

	return &LoginResult{TokenPair: pair, User: u}, nil
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
	var subject string

	if refreshTokenStr != "" {
		if claims, err := s.tm.ParseRefresh(refreshTokenStr); err == nil {
			subject = claims.Subject
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
			if subject == "" {
				subject = claims.Subject
			}
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

	// 登出后清理权限缓存（下次访问重新加载）
	if subject != "" {
		if err := s.ClearUserCache(ctx, subject); err != nil {
			logger.WarnContext(ctx, "清理用户权限缓存失败", logger.UserUUID(subject), logger.Err(err))
		}
		s.audit.Record(ctx, audit.Entry{
			UserUUID:   subject,
			Action:     model.AuditActionLogout,
			Resource:   model.AuditResourceAuth,
			ResourceID: subject,
		})
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

// EmailCodeConfig 邮箱验证码（注册校验 / 验证码登录）配置。
type EmailCodeConfig struct {
	CodeLength   int           // 验证码长度
	CodeTTL      time.Duration // 验证码有效期
	CodeResendCD time.Duration // 重发冷却
}

// applyDefaults 补全默认值。
func (c *EmailCodeConfig) applyDefaults() {
	if c.CodeLength <= 0 {
		c.CodeLength = 6
	}
	if c.CodeTTL <= 0 {
		c.CodeTTL = 10 * time.Minute
	}
	if c.CodeResendCD <= 0 {
		c.CodeResendCD = time.Minute
	}
}

// SendVerifyCode 生成邮箱验证码并投递异步发送任务。
//
// scene 取值：register（注册校验）/ login（验证码登录）/ reset_password（密码重置）/ change_email（修改邮箱）。
// 重发冷却内返回 ErrResendTooSoon。
//
// 返回值：task_id 用于订阅发送进度（见 gateway 的 SSE 端点），expires_in 为验证码有效期（秒）。
//
// 这里不直接调用 SMTP：邮件发送被下沉到 asynq 队列由独立 Worker 并发消费，
// 请求链路只保留 Redis 读写与一次入队，耗时从「数百毫秒到数秒」降到毫秒级。
func (s *AuthService) SendVerifyCode(ctx context.Context, in SendVerifyCodeInput) (string, int64, error) {
	email := normalizeEmail(in.Email)
	scene := in.Scene
	if email == "" {
		return "", 0, fmt.Errorf("邮箱不能为空")
	}
	if !supportedScene(scene) {
		return "", 0, fmt.Errorf("不支持的验证码场景: %s", scene)
	}
	if s.producer == nil {
		return "", 0, fmt.Errorf("异步任务队列未初始化")
	}

	// 人机验证放在最前面：未通过就不生成验证码、不占用重发冷却，
	// 避免机器人把冷却期刷满导致真实用户发不出验证码。
	if s.captcha != nil {
		// 用场景名作为 action，与前端 widget 的 action 保持一致（widget 未设置时后端跳过该校验）
		if err := s.captcha.Verify(ctx, in.CaptchaToken, in.ClientIP, scene); err != nil {
			return "", 0, fmt.Errorf("%w", err)
		}
	}

	ok, err := s.codes.TryLockResend(ctx, scene, email, s.emailCfg.CodeResendCD)
	if err != nil {
		return "", 0, fmt.Errorf("检查验证码重发频率失败: %w", err)
	}
	if !ok {
		return "", 0, ErrResendTooSoon
	}

	code, err := generateNumericCode(s.emailCfg.CodeLength)
	if err != nil {
		return "", 0, fmt.Errorf("生成邮箱验证码失败: %w", err)
	}
	if err := s.codes.SaveCode(ctx, scene, email, code, s.emailCfg.CodeTTL); err != nil {
		return "", 0, fmt.Errorf("保存邮箱验证码失败: %w", err)
	}

	taskID, err := s.producer.EnqueueEmailVerifyCode(ctx, queue.EmailVerifyCodePayload{
		To:      email,
		Code:    code,
		Scene:   scene,
		Purpose: emailPurpose(scene),
		TTL:     int64(s.emailCfg.CodeTTL.Seconds()),
	})
	if err != nil {
		// 入队失败意味着验证码永远送不出去，回滚验证码与冷却锁，
		// 避免用户白等一个冷却周期；对外只回通用提示，细节留在日志
		logger.ErrorContext(ctx, "投递邮箱验证码任务失败", "email", email, "scene", scene, logger.Err(err))
		s.rollbackSendCode(ctx, scene, email)
		return "", 0, fmt.Errorf("邮件发送服务暂时不可用，请稍后再试")
	}

	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			Action:   model.AuditActionEmailVerifySend,
			Resource: model.AuditResourceAuth,
			Detail:   "scene=" + scene,
		})
	}
	return taskID, int64(s.emailCfg.CodeTTL.Seconds()), nil
}

// rollbackSendCode 入队失败后的补偿：删除已写入的验证码并释放冷却锁。
func (s *AuthService) rollbackSendCode(ctx context.Context, scene, email string) {
	if err := s.codes.DeleteCode(ctx, scene, email); err != nil {
		logger.WarnContext(ctx, "回滚邮箱验证码失败", "email", email, logger.Err(err))
	}
	if err := s.codes.UnlockResend(ctx, scene, email); err != nil {
		logger.WarnContext(ctx, "释放验证码重发冷却失败", "email", email, logger.Err(err))
	}
}

// ChangeEmail 校验新邮箱验证码并将账号邮箱改为新邮箱。
//
// 用于已登录用户修改邮箱：先调用 SendVerifyCode{scene:"change_email"} 向新邮箱发码，
// 再携带 new_email + code 调用本方法。校验通过后更新邮箱并标记为已验证。
// 若 new_email 已被其他账号占用，返回 ErrEmailAlreadyUsed。
func (s *AuthService) ChangeEmail(ctx context.Context, userUUID uuid.UUID, newEmail, code string) error {
	newEmail = normalizeEmail(newEmail)
	if newEmail == "" {
		return fmt.Errorf("新邮箱不能为空")
	}

	// 防账号枚举：先确认当前用户存在
	u, err := s.users.FindByUUID(ctx, userUUID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}
	// 新邮箱与当前邮箱相同则无需变更
	if u.Email == newEmail {
		if err := s.codes.DeleteCode(ctx, "change_email", newEmail); err != nil {
			logger.WarnContext(ctx, "删除已用邮箱验证码失败", "email", newEmail, logger.Err(err))
		}
		return nil
	}
	// 新邮箱不可被其他账号占用
	used, err := s.users.ExistsByEmail(ctx, newEmail)
	if err != nil {
		return fmt.Errorf("检查邮箱占用失败: %w", err)
	}
	if used {
		return ErrEmailAlreadyUsed
	}

	saved, err := s.codes.GetCode(ctx, "change_email", newEmail)
	if err != nil {
		return fmt.Errorf("读取邮箱验证码失败: %w", err)
	}
	if saved == "" {
		return ErrCodeNotSent
	}
	if !strings.EqualFold(saved, strings.TrimSpace(code)) {
		return ErrCodeMismatch
	}
	if err := s.codes.DeleteCode(ctx, "change_email", newEmail); err != nil {
		logger.WarnContext(ctx, "删除已用邮箱验证码失败", "email", newEmail, logger.Err(err))
	}

	if err := s.users.UpdateEmail(ctx, userUUID, newEmail); err != nil {
		return fmt.Errorf("更新邮箱失败: %w", err)
	}

	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			UserUUID:   userUUID.String(),
			Action:     model.AuditActionEmailVerifySuccess,
			Resource:   model.AuditResourceAuth,
			ResourceID: userUUID.String(),
			Detail:     "change_email",
		})
	}
	return nil
}

// LoginWithCode 邮箱验证码登录（免密）。
//
// 校验通过后签发双令牌；若账号开启 2FA，则仅返回 mfa_ticket（需再调用 VerifyMFALogin）。
func (s *AuthService) LoginWithCode(ctx context.Context, email, code string, dev dto.Device) (*LoginResult, error) {
	email = normalizeEmail(email)
	if email == "" {
		return nil, fmt.Errorf("邮箱不能为空")
	}

	saved, err := s.codes.GetCode(ctx, "login", email)
	if err != nil {
		return nil, fmt.Errorf("读取邮箱验证码失败: %w", err)
	}
	if saved == "" {
		return nil, ErrCodeNotSent
	}
	if !strings.EqualFold(saved, strings.TrimSpace(code)) {
		return nil, ErrCodeMismatch
	}
	// 校验通过立即删除，防止重复使用
	if err := s.codes.DeleteCode(ctx, "login", email); err != nil {
		logger.WarnContext(ctx, "删除已用邮箱验证码失败", "email", email, logger.Err(err))
	}

	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 重置失败计数（验证码登录视为成功认证）
	if u.LoginFailCount != 0 || u.LockedUntil != nil {
		if err := s.users.ResetLoginFails(ctx, u.UUID); err != nil {
			logger.WarnContext(ctx, "重置登录失败计数失败", logger.UserUUID(u.UUID.String()), logger.Err(err))
		}
	}

	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			UserUUID:   u.UUID.String(),
			Action:     model.AuditActionEmailLoginCode,
			Resource:   model.AuditResourceAuth,
			ResourceID: u.UUID.String(),
			IP:         dev.IP,
			UserAgent:  dev.UserAgent,
		})
	}

	// 开启 2FA：返回票据，等待第二步
	if u.MFAEnabled {
		ticket, err := s.tm.GenerateMFATicket(u.UUID.String(), uuid.NewString())
		if err != nil {
			return nil, fmt.Errorf("签发 MFA 票据失败: %w", err)
		}
		return &LoginResult{User: u, RequiresMFA: true, MFATicket: ticket}, nil
	}

	tokens, err := s.issueTokens(ctx, u.UUID.String(), dev)
	if err != nil {
		return nil, err
	}
	return &LoginResult{TokenPair: tokens, User: u}, nil
}

// UpdateProfile 更新用户个人信息（昵称 / 头像 / 简介）。
func (s *AuthService) UpdateProfile(ctx context.Context, userUUID, nickname, avatarURL, bio string) (*model.User, error) {
	id, err := uuid.Parse(userUUID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	if err := s.users.UpdateProfile(ctx, id, nickname, avatarURL, bio); err != nil {
		return nil, fmt.Errorf("更新用户信息失败: %w", err)
	}
	return s.users.FindByUUID(ctx, id)
}

// ChangePassword 修改密码：校验旧密码后写入新密码哈希，并清理权限缓存。
// 改密后清理缓存可确保下次访问按最新权限生效。
func (s *AuthService) ChangePassword(ctx context.Context, userUUID, oldPassword, newPassword string) error {
	id, err := uuid.Parse(userUUID)
	if err != nil {
		return ErrUserNotFound
	}
	if len(newPassword) > maxPasswordBytes {
		return fmt.Errorf("密码长度不能超过 %d 字节", maxPasswordBytes)
	}

	u, err := s.users.FindByUUID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}
	if u.PasswordHash == nil || *u.PasswordHash == "" {
		return ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}
	if err := s.users.UpdatePasswordHash(ctx, id, string(hash)); err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}

	// 改密后清理权限缓存
	if err := s.ClearUserCache(ctx, userUUID); err != nil {
		logger.WarnContext(ctx, "改密后清理权限缓存失败", logger.UserUUID(userUUID), logger.Err(err))
	}

	s.audit.Record(ctx, audit.Entry{
		UserUUID:   userUUID,
		Action:     model.AuditActionChangePwd,
		Resource:   model.AuditResourceUser,
		ResourceID: userUUID,
	})

	return nil
}

// ListSessions 返回用户当前活跃会话列表。
func (s *AuthService) ListSessions(ctx context.Context, userUUID string) ([]repository.TokenMeta, error) {
	return s.tokens.ListUserTokens(ctx, userUUID)
}

// RevokeSession 吊销指定会话：删除 refresh 白名单并移除会话记录。
func (s *AuthService) RevokeSession(ctx context.Context, userUUID, tokenID string) error {
	if err := s.tokens.DeleteRefreshValid(ctx, tokenID); err != nil {
		return fmt.Errorf("吊销刷新令牌失败: %w", err)
	}
	if err := s.tokens.RemoveUserToken(ctx, userUUID, tokenID); err != nil {
		logger.WarnContext(ctx, "移除会话记录失败", logger.UserUUID(userUUID), logger.Err(err))
	}
	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// GetPermissions 返回用户权限编码列表（缓存优先，供网关校验权限）。
func (s *AuthService) GetPermissions(ctx context.Context, userUUID string) ([]string, error) {
	if s.rbac == nil {
		return nil, fmt.Errorf("RBAC 服务未初始化")
	}
	id, err := uuid.Parse(userUUID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return s.rbac.GetUserPermissions(ctx, id)
}

// CheckPermission 校验用户是否拥有指定权限。
func (s *AuthService) CheckPermission(ctx context.Context, userUUID, perm string) (bool, error) {
	if s.rbac == nil {
		return false, fmt.Errorf("RBAC 服务未初始化")
	}
	id, err := uuid.Parse(userUUID)
	if err != nil {
		return false, ErrUserNotFound
	}
	return s.rbac.CheckPermission(ctx, id, perm)
}

// OAuthLogin 通过第三方账号登录。
// 委托 oauth 服务完成「已绑定直接登录 / 未绑定自动注册」，
// 随后复用统一的 issueTokens 签发双令牌。
func (s *AuthService) OAuthLogin(ctx context.Context, p oauth.Profile, dev dto.Device) (*dto.TokenPair, *model.User, bool, error) {
	if s.oauth == nil {
		return nil, nil, false, fmt.Errorf("第三方登录服务未初始化")
	}

	u, isNew, err := s.oauth.LoginOrRegister(ctx, p)
	if err != nil {
		return nil, nil, false, err
	}

	pair, err := s.issueTokens(ctx, u.UUID.String(), dev)
	if err != nil {
		return nil, nil, false, err
	}

	if err := s.users.UpdateLoginInfo(ctx, u.UUID, dev.IP, dev.DeviceName); err != nil {
		logger.WarnContext(ctx, "更新第三方登录信息失败", logger.UserUUID(u.UUID.String()), logger.Err(err))
	}

	return pair, u, isNew, nil
}

// BindOAuth 为当前用户绑定第三方账号。
func (s *AuthService) BindOAuth(ctx context.Context, userUUID uuid.UUID, p oauth.Profile) error {
	if s.oauth == nil {
		return fmt.Errorf("第三方登录服务未初始化")
	}
	return s.oauth.Bind(ctx, userUUID, p)
}

// UnbindOAuth 解绑第三方账号。
func (s *AuthService) UnbindOAuth(ctx context.Context, userUUID uuid.UUID, provider string) error {
	if s.oauth == nil {
		return fmt.Errorf("第三方登录服务未初始化")
	}
	return s.oauth.Unbind(ctx, userUUID, provider)
}

// ListOAuthBindings 列出用户已绑定的第三方账号。
func (s *AuthService) ListOAuthBindings(ctx context.Context, userUUID uuid.UUID) ([]model.OAuth, error) {
	if s.oauth == nil {
		return nil, fmt.Errorf("第三方登录服务未初始化")
	}
	return s.oauth.ListBindings(ctx, userUUID)
}

// ClearUserCache 清理用户权限缓存（权限变更后调用）。
func (s *AuthService) ClearUserCache(ctx context.Context, userUUID string) error {
	if s.rbac == nil {
		return nil
	}
	id, err := uuid.Parse(userUUID)
	if err != nil {
		return ErrUserNotFound
	}
	return s.rbac.ClearUserCache(ctx, id)
}
