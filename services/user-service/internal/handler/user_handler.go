// Package handler 实现 user.proto 定义的 gRPC 服务端。
//
// 该层只负责协议转换与错误码映射，不含业务逻辑。
package handler

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	userpb "github.com/museflow/proto/user"
	"github.com/museflow/pkg/logger"
	"github.com/museflow/user-service/internal/model"
	"github.com/museflow/user-service/internal/service/admin"
	"github.com/museflow/user-service/internal/service/auth"
	"github.com/museflow/user-service/internal/service/dto"
)

// UserHandler gRPC 处理器。
type UserHandler struct {
	userpb.UnimplementedUserServiceServer
	auth  *auth.AuthService
	admin *admin.Service
}

// NewUserHandler 构造 gRPC 处理器。
func NewUserHandler(a *auth.AuthService, adm *admin.Service) *UserHandler {
	return &UserHandler{auth: a, admin: adm}
}

// Register 用户注册。
func (h *UserHandler) Register(ctx context.Context, req *userpb.RegisterRequest) (*userpb.RegisterResponse, error) {
	logger.InfoContext(ctx, "收到注册请求", "email", req.GetEmail())
	u, err := h.auth.Register(ctx, req.GetEmail(), req.GetPassword(), req.GetNickname(), req.GetCode())
	if err != nil {
		logger.WarnContext(ctx, "注册失败", "email", req.GetEmail(), logger.Err(err))
		return nil, mapError(err)
	}
	logger.InfoContext(ctx, "注册成功", logger.UserUUID(u.UUID.String()))
	return &userpb.RegisterResponse{User: toUserInfo(u)}, nil
}

// Login 用户登录。
//
// 若账号已开启 2FA，则不签发令牌，返回 requires_mfa=true 与中间票据，
// 由前端引导用户输入验证码后调用 VerifyMFALogin。
func (h *UserHandler) Login(ctx context.Context, req *userpb.LoginRequest) (*userpb.LoginResponse, error) {
	logger.InfoContext(ctx, "收到登录请求", "email", req.GetEmail())
	res, err := h.auth.Login(ctx, req.GetEmail(), req.GetPassword(), toDevice(req.GetDevice()))
	if err != nil {
		logger.WarnContext(ctx, "登录失败", "email", req.GetEmail(), logger.Err(err))
		return nil, mapError(err)
	}

	// 需要二次验证：不下发令牌，仅返回票据
	if res.RequiresMFA {
		logger.InfoContext(ctx, "账号已开启 2FA，等待二次验证", logger.UserUUID(res.User.UUID.String()))
		return &userpb.LoginResponse{
			User:        toUserInfo(res.User),
			RequiresMfa: true,
			MfaTicket:   res.MFATicket,
		}, nil
	}

	logger.InfoContext(ctx, "登录成功", logger.UserUUID(res.User.UUID.String()))
	return &userpb.LoginResponse{
		AccessToken:      res.TokenPair.AccessToken,
		RefreshToken:     res.TokenPair.RefreshToken,
		DeviceId:         res.TokenPair.DeviceID,
		ExpiresIn:        res.TokenPair.ExpiresIn,
		RefreshExpiresIn: res.TokenPair.RefreshExpiresIn,
		User:             toUserInfo(res.User),
	}, nil
}

// VerifyMFALogin 登录流程的 2FA 二次验证，通过后签发双令牌。
func (h *UserHandler) VerifyMFALogin(ctx context.Context, req *userpb.VerifyMFALoginRequest) (*userpb.LoginResponse, error) {
	res, usedRecovery, err := h.auth.VerifyMFALogin(ctx, req.GetMfaTicket(), req.GetCode(), toDevice(req.GetDevice()))
	if err != nil {
		logger.WarnContext(ctx, "2FA 登录验证失败", logger.Err(err))
		return nil, mapMFAError(err)
	}

	logger.InfoContext(ctx, "2FA 登录成功", logger.UserUUID(res.User.UUID.String()), "used_recovery", usedRecovery)

	resp := &userpb.LoginResponse{
		AccessToken:      res.TokenPair.AccessToken,
		RefreshToken:     res.TokenPair.RefreshToken,
		DeviceId:         res.TokenPair.DeviceID,
		ExpiresIn:        res.TokenPair.ExpiresIn,
		RefreshExpiresIn: res.TokenPair.RefreshExpiresIn,
		User:             toUserInfo(res.User),
	}
	// 使用恢复码登录时，在 user 信息中无法表达，交由上层通过日志/审计感知；
	// 前端可据此引导用户重新生成恢复码（此处借用 mfa_ticket 字段置空表示已完成）。
	return resp, nil
}

// SendVerifyCode 发送邮箱验证码（注册校验 / 验证码登录 / 补验证邮箱）。
func (h *UserHandler) SendVerifyCode(ctx context.Context, req *userpb.SendVerifyCodeRequest) (*userpb.SendVerifyCodeResponse, error) {
	if err := h.auth.SendVerifyCode(ctx, req.GetEmail(), req.GetScene()); err != nil {
		return nil, mapError(err)
	}
	return &userpb.SendVerifyCodeResponse{}, nil
}

// LoginWithCode 邮箱验证码登录（免密）。
func (h *UserHandler) LoginWithCode(ctx context.Context, req *userpb.LoginWithCodeRequest) (*userpb.LoginResponse, error) {
	logger.InfoContext(ctx, "收到邮箱验证码登录请求", "email", req.GetEmail())
	res, err := h.auth.LoginWithCode(ctx, req.GetEmail(), req.GetCode(), toDevice(req.GetDevice()))
	if err != nil {
		logger.WarnContext(ctx, "邮箱验证码登录失败", "email", req.GetEmail(), logger.Err(err))
		return nil, mapError(err)
	}

	// 需要二次验证：不下发令牌，仅返回票据
	if res.RequiresMFA {
		logger.InfoContext(ctx, "账号已开启 2FA，等待二次验证", logger.UserUUID(res.User.UUID.String()))
		return &userpb.LoginResponse{
			User:        toUserInfo(res.User),
			RequiresMfa: true,
			MfaTicket:   res.MFATicket,
		}, nil
	}

	logger.InfoContext(ctx, "邮箱验证码登录成功", logger.UserUUID(res.User.UUID.String()))
	return &userpb.LoginResponse{
		AccessToken:      res.TokenPair.AccessToken,
		RefreshToken:     res.TokenPair.RefreshToken,
		DeviceId:         res.TokenPair.DeviceID,
		ExpiresIn:        res.TokenPair.ExpiresIn,
		RefreshExpiresIn: res.TokenPair.RefreshExpiresIn,
		User:             toUserInfo(res.User),
	}, nil
}

// Refresh 刷新 access token。
func (h *UserHandler) Refresh(ctx context.Context, req *userpb.RefreshRequest) (*userpb.RefreshResponse, error) {
	logger.InfoContext(ctx, "收到刷新令牌请求")
	token, expiresIn, err := h.auth.Refresh(ctx, req.GetRefreshToken(), toDevice(req.GetDevice()))
	if err != nil {
		logger.WarnContext(ctx, "刷新令牌失败", logger.Err(err))
		return nil, mapError(err)
	}
	return &userpb.RefreshResponse{AccessToken: token, ExpiresIn: expiresIn}, nil
}

// Logout 用户登出。
func (h *UserHandler) Logout(ctx context.Context, req *userpb.LogoutRequest) (*userpb.LogoutResponse, error) {
	logger.InfoContext(ctx, "收到登出请求")
	if err := h.auth.Logout(ctx, req.GetAccessToken(), req.GetRefreshToken()); err != nil {
		logger.WarnContext(ctx, "登出失败", logger.Err(err))
		return nil, mapError(err)
	}
	logger.InfoContext(ctx, "登出成功")
	return &userpb.LogoutResponse{Success: true}, nil
}

// GetProfile 获取用户信息。
func (h *UserHandler) GetProfile(ctx context.Context, req *userpb.GetProfileRequest) (*userpb.GetProfileResponse, error) {
	logger.InfoContext(ctx, "收到获取用户信息请求", logger.UserUUID(req.GetUuid()))
	u, err := h.auth.GetProfile(ctx, req.GetUuid())
	if err != nil {
		return nil, mapError(err)
	}
	return &userpb.GetProfileResponse{User: toUserInfo(u)}, nil
}

// ValidateToken 校验 access token，供网关鉴权使用。
func (h *UserHandler) ValidateToken(ctx context.Context, req *userpb.ValidateTokenRequest) (*userpb.ValidateTokenResponse, error) {
	uid, err := h.auth.ValidateAccess(ctx, req.GetAccessToken())
	if err != nil {
		if errors.Is(err, auth.ErrTokenInvalid) {
			// 令牌无效属于正常业务结果，不作为 gRPC 错误返回
			return &userpb.ValidateTokenResponse{Valid: false}, nil
		}
		return nil, mapError(err)
	}
	return &userpb.ValidateTokenResponse{Valid: true, Uuid: uid}, nil
}

// toDevice 将 proto 设备上下文转换为业务层结构。
func toDevice(d *userpb.DeviceContext) dto.Device {
	if d == nil {
		return dto.Device{}
	}
	return dto.Device{
		DeviceID:   d.GetDeviceId(),
		UserAgent:  d.GetUserAgent(),
		IP:         d.GetIp(),
		DeviceName: d.GetDeviceName(),
	}
}

// toUserInfo 将数据模型转换为对外的 proto 消息，处理可空字段。
func toUserInfo(u *model.User) *userpb.UserInfo {
	if u == nil {
		return nil
	}

	info := &userpb.UserInfo{
		Uuid:          u.UUID.String(),
		Email:         u.Email,
		Nickname:      u.Nickname,
		Status:        int32(u.Status),
		EmailVerified: u.EmailVerified,
		PhoneVerified: u.PhoneVerified,
		MfaEnabled:    u.MFAEnabled,
		CreatedAt:     u.CreatedAt.Unix(),
	}

	if u.Phone != nil {
		info.Phone = *u.Phone
	}
	if u.AvatarURL != nil {
		info.AvatarUrl = *u.AvatarURL
	}
	if u.Bio != nil {
		info.Bio = *u.Bio
	}
	if u.LastLoginAt != nil {
		info.LastLoginAt = u.LastLoginAt.Unix()
	}

	return info
}

// mapError 将业务错误映射为 gRPC status，便于网关转换为 HTTP 状态码。
func mapError(err error) error {
	switch {
	case errors.Is(err, auth.ErrEmailExists), errors.Is(err, auth.ErrEmailAlreadyUsed):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, auth.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, auth.ErrCodeNotSent), errors.Is(err, auth.ErrCodeMismatch):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, auth.ErrResendTooSoon):
		return status.Error(codes.ResourceExhausted, err.Error())
	case errors.Is(err, auth.ErrTokenInvalid):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, auth.ErrDeviceMismatch):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, auth.ErrUserNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, auth.ErrAccountUnavailable):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, auth.ErrAccountLocked):
		return status.Error(codes.ResourceExhausted, err.Error())
	case errors.Is(err, ErrInvalidUUID):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
