// Package oauth 实现第三方账号的绑定、解绑与登录。
//
// 依赖方向：oauth -> repository + rbac + audit（不依赖 auth），
// 因此 auth 与 oauth 互不反向依赖，避免循环。
//
// 登录策略：
//   - 已绑定：直接返回对应用户，并加载权限缓存
//   - 未绑定且允许自动注册：创建新用户（密码为空，仅支持第三方登录），
//     授予默认角色 user，并自动建立绑定
package oauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/museflow/pkg/logger"
	"github.com/museflow/user-service/internal/model"
	"github.com/museflow/user-service/internal/repository"
	"github.com/museflow/user-service/internal/service/audit"
	"github.com/museflow/user-service/internal/service/rbac"
)

// ErrOAuthNotFound 未绑定该第三方账号。
var ErrOAuthNotFound = repository.ErrOAuthNotFound

// ErrProviderNotSupported 不支持的第三方平台。
var ErrProviderNotSupported = errors.New("不支持的第三方登录平台")

// ErrAlreadyBound 该第三方账号已绑定到其他用户。
var ErrAlreadyBound = errors.New("该第三方账号已绑定到其他用户")

// supportedProviders 支持的第三方平台白名单。
var supportedProviders = map[string]bool{
	model.ProviderGitHub: true,
	model.ProviderGoogle: true,
	model.ProviderWeChat: true,
	model.ProviderQQ:     true,
	model.ProviderApple:  true,
}

// Service 第三方登录服务。
type Service struct {
	oauth  repository.OAuthRepository
	users  repository.UserRepository
	rbac   *rbac.Service
	audit  *audit.Service
}

// NewService 构造第三方登录服务。
func NewService(
	oauthRepo repository.OAuthRepository,
	users repository.UserRepository,
	rbacSvc *rbac.Service,
	auditSvc *audit.Service,
) *Service {
	return &Service{oauth: oauthRepo, users: users, rbac: rbacSvc, audit: auditSvc}
}

// Profile 第三方平台返回的用户资料（由调用方适配各平台后传入）。
type Profile struct {
	Provider     string // 平台标识，如 github
	ProviderUID  string // 第三方用户唯一标识
	Email        string // 邮箱快照
	Nickname     string // 昵称快照
	AvatarURL    string // 头像快照
	AccessToken  string // 第三方 access token
	RefreshToken string // 第三方 refresh token
	ExpiresAt    *time.Time
	Extra        string // 平台特有字段（JSON）
}

// assignDefaultRole 授予默认角色；rbac 未注入或失败时仅告警，不阻断流程。
func (s *Service) assignDefaultRole(ctx context.Context, u *model.User) {
	if s.rbac == nil {
		return
	}
	if err := s.rbac.AssignRole(ctx, u.UUID, rbac.RoleUser, u.UUID); err != nil {
		logger.WarnContext(ctx, "授予默认角色失败", logger.UserUUID(u.UUID.String()), logger.Err(err))
	}
}

// loadPermissions 加载权限缓存；rbac 未注入或失败时仅告警。
func (s *Service) loadPermissions(ctx context.Context, userUUID uuid.UUID) {
	if s.rbac == nil {
		return
	}
	if _, err := s.rbac.GetUserPermissions(ctx, userUUID); err != nil {
		logger.WarnContext(ctx, "加载权限缓存失败", logger.UserUUID(userUUID.String()), logger.Err(err))
	}
}

// record 写入审计日志；audit 未注入时静默跳过。
func (s *Service) record(ctx context.Context, e audit.Entry) {
	if s.audit == nil {
		return
	}
	s.audit.Record(ctx, e)
}

// Bind 为已登录用户绑定第三方账号。
func (s *Service) Bind(ctx context.Context, userUUID uuid.UUID, p Profile) error {
	if !supportedProviders[p.Provider] {
		return ErrProviderNotSupported
	}
	if p.ProviderUID == "" {
		return fmt.Errorf("第三方用户标识不能为空")
	}

	// 该第三方账号若已绑定到其他用户，拒绝重复绑定
	if exist, err := s.oauth.FindByProvider(ctx, p.Provider, p.ProviderUID); err == nil && exist != nil {
		if exist.UserUUID != userUUID {
			return ErrAlreadyBound
		}
		return nil // 已绑定到当前用户，幂等
	}

	now := time.Now()
	o := &model.OAuth{
		UserUUID:         userUUID,
		Provider:         p.Provider,
		ProviderUserID:   p.ProviderUID,
		IsActive:         true,
		LastLoginAt:      &now,
	}
	applyProfile(o, p)

	if err := s.oauth.Create(ctx, o); err != nil {
		return fmt.Errorf("绑定第三方账号失败: %w", err)
	}

	s.record(ctx, audit.Entry{
		UserUUID:   userUUID.String(),
		Action:     "bind_oauth",
		Resource:   "oauth",
		ResourceID: p.Provider,
		Detail:     map[string]string{"provider": p.Provider},
	})
	return nil
}

// Unbind 解绑第三方账号。
func (s *Service) Unbind(ctx context.Context, userUUID uuid.UUID, provider string) error {
	if err := s.oauth.Delete(ctx, userUUID, provider); err != nil {
		return err
	}
	s.record(ctx, audit.Entry{
		UserUUID:   userUUID.String(),
		Action:     "unbind_oauth",
		Resource:   "oauth",
		ResourceID: provider,
	})
	return nil
}

// ListBindings 列出用户已绑定的第三方账号。
func (s *Service) ListBindings(ctx context.Context, userUUID uuid.UUID) ([]model.OAuth, error) {
	return s.oauth.ListByUser(ctx, userUUID)
}

// LoginOrRegister 通过第三方账号登录；未绑定时自动注册新用户并建立绑定。
// 返回用户、是否为新注册用户。
func (s *Service) LoginOrRegister(ctx context.Context, p Profile) (*model.User, bool, error) {
	if !supportedProviders[p.Provider] {
		return nil, false, ErrProviderNotSupported
	}
	if p.ProviderUID == "" {
		return nil, false, fmt.Errorf("第三方用户标识不能为空")
	}

	// 1. 已绑定 -> 直接登录
	if bound, err := s.oauth.FindByProvider(ctx, p.Provider, p.ProviderUID); err == nil && bound != nil {
		u, ferr := s.users.FindByUUID(ctx, bound.UserUUID)
		if ferr != nil {
			return nil, false, fmt.Errorf("查询绑定用户失败: %w", ferr)
		}
		if u.Status != model.StatusNormal {
			return nil, false, fmt.Errorf("账号状态异常，无法登录")
		}
		_ = s.oauth.TouchLastLogin(ctx, bound.ID)
		// 刷新权限缓存
		s.loadPermissions(ctx, u.UUID)
		s.record(ctx, audit.Entry{
			UserUUID:   u.UUID.String(),
			Action:     "oauth_login",
			Resource:   "auth",
			ResourceID: u.UUID.String(),
			Detail:     map[string]string{"provider": p.Provider},
		})
		return u, false, nil
	}

	// 2. 未绑定 -> 自动注册
	nickname := p.Nickname
	if nickname == "" {
		nickname = defaultNickname(p)
	}

	// 第三方用户无密码（password_hash 为空），不可用于密码登录
	u := &model.User{
		UUID:     uuid.New(),
		Email:    p.Email,
		Nickname: nickname,
		Status:   model.StatusNormal,
	}
	if p.AvatarURL != "" {
		u.AvatarURL = &p.AvatarURL
	}
	// 邮箱为空时使用占位值，避免唯一索引冲突（email 为 NOT NULL 且唯一）
	if u.Email == "" {
		placeholder := fmt.Sprintf("%s_%s@oauth.local", p.Provider, p.ProviderUID)
		u.Email = placeholder
	}

	if err := s.users.Create(ctx, u); err != nil {
		return nil, false, fmt.Errorf("创建第三方用户失败: %w", err)
	}

	// 授予默认角色
	s.assignDefaultRole(ctx, u)

	// 建立绑定
	now := time.Now()
	o := &model.OAuth{
		UserUUID:       u.UUID,
		Provider:       p.Provider,
		ProviderUserID: p.ProviderUID,
		IsActive:       true,
		LastLoginAt:    &now,
	}
	applyProfile(o, p)
	if err := s.oauth.Create(ctx, o); err != nil {
		return nil, false, fmt.Errorf("建立第三方绑定失败: %w", err)
	}

	// 加载权限缓存
	s.loadPermissions(ctx, u.UUID)

	s.record(ctx, audit.Entry{
		UserUUID:   u.UUID.String(),
		Action:     "oauth_register",
		Resource:   "user",
		ResourceID: u.UUID.String(),
		Detail:     map[string]string{"provider": p.Provider},
	})

	return u, true, nil
}

// applyProfile 把第三方资料快照写入绑定记录。
func applyProfile(o *model.OAuth, p Profile) {
	if p.Email != "" {
		o.ProviderEmail = &p.Email
	}
	if p.Nickname != "" {
		o.ProviderNickname = &p.Nickname
	}
	if p.AvatarURL != "" {
		o.ProviderAvatar = &p.AvatarURL
	}
	if p.AccessToken != "" {
		o.AccessToken = &p.AccessToken
	}
	if p.RefreshToken != "" {
		o.RefreshToken = &p.RefreshToken
	}
	if p.Extra != "" {
		o.Extra = p.Extra
	}
	o.ExpiresAt = p.ExpiresAt
}

// defaultNickname 第三方资料无昵称时生成默认昵称。
func defaultNickname(p Profile) string {
	base := p.Email
	if base == "" {
		base = p.ProviderUID
	}
	if i := strings.Index(base, "@"); i > 0 {
		base = base[:i]
	}
	if base == "" {
		base = p.Provider
	}
	return fmt.Sprintf("%s用户_%s", p.Provider, base)
}
