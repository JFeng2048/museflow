package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/museflow/api-gateway/internal/client"
	userdto "github.com/museflow/api-gateway/internal/dto/user_dto"
	"github.com/museflow/api-gateway/internal/middleware"
	"github.com/museflow/pkg/errcode"
	userpb "github.com/museflow/proto/user"
)

// UserHandler 用户信息相关处理器。
type UserHandler struct {
	users *client.UserClient
}

// NewUserHandler 构造用户处理器。
func NewUserHandler(users *client.UserClient) *UserHandler {
	return &UserHandler{users: users}
}

// Profile 获取当前用户信息
//
//	@Summary		获取当前用户信息
//	@Description	根据 access token 中的用户标识返回当前登录用户的详细信息
//	@Tags			用户
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	errcode.Response{data=userdto.UserInfo}	"获取成功"
//	@Failure		401	{object}	errcode.Response			"未认证或令牌已失效"
//	@Failure		404	{object}	errcode.Response			"用户不存在"
//	@Router			/user/profile [get]
func (h *UserHandler) Profile(c *gin.Context) {
	// 由鉴权中间件写入
	uid := middleware.CurrentUserUUID(c)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, errcode.ErrorGin(c, errcode.CodeUnauthorized))
		return
	}

	resp, err := h.users.Service().GetProfile(c.Request.Context(), &userpb.GetProfileRequest{Uuid: uid})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, errcode.SuccessGin(c, toUserInfo(resp.GetUser())))
}

// toUserInfo 将 proto 用户消息转换为 HTTP 响应结构。
func toUserInfo(u *userpb.UserInfo) userdto.UserInfo {
	if u == nil {
		return userdto.UserInfo{}
	}
	return userdto.UserInfo{
		UUID:          u.GetUuid(),
		Email:         u.GetEmail(),
		Phone:         u.GetPhone(),
		Nickname:      u.GetNickname(),
		AvatarURL:     u.GetAvatarUrl(),
		Bio:           u.GetBio(),
		Status:        u.GetStatus(),
		EmailVerified: u.GetEmailVerified(),
		PhoneVerified: u.GetPhoneVerified(),
		MFAEnabled:    u.GetMfaEnabled(),
		LastLoginAt:   u.GetLastLoginAt(),
		CreatedAt:     u.GetCreatedAt(),
	}
}

// writeGRPCError 将 gRPC status 映射为 HTTP 状态码与统一错误响应。
func writeGRPCError(c *gin.Context, err error) {
	lang := errcode.LangFromGin(c)
	st, ok := status.FromError(err)
	if !ok {
		c.JSON(http.StatusInternalServerError, errcode.ErrorWithLang(errcode.CodeInternal, lang))
		return
	}

	var code errcode.Code
	switch st.Code() {
	case codes.InvalidArgument:
		code = errcode.CodeParamInvalid
	case codes.Unauthenticated:
		code = errcode.CodeUnauthorized
	case codes.PermissionDenied:
		code = errcode.CodeForbidden
	case codes.NotFound:
		code = errcode.CodeNotFound
	case codes.AlreadyExists:
		code = errcode.CodeConflict
	case codes.FailedPrecondition:
		code = errcode.CodeParamInvalid
	case codes.Unavailable, codes.DeadlineExceeded:
		code = errcode.CodeServiceUnavail
	default:
		code = errcode.CodeInternal
	}

	// gRPC 透传的业务提示优先；否则使用业务码默认消息（按请求语言）
	msg := st.Message()
	if msg == "" {
		msg = errcode.Message(code, lang)
	}
	c.JSON(errcode.Fail(code, msg).HTTPStatus(), errcode.Fail(code, msg))
}
