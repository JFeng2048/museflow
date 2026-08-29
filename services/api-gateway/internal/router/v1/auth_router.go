package v1

import (
	"github.com/gin-gonic/gin"
)

// registerAuthRoutes 注册认证相关路由（/api/v1/auth）。
//
// 注册与登录无需认证；refresh 走 HttpOnly Cookie 校验，不需要 access token；
// logout 需要 access token。
func registerAuthRoutes(r *gin.RouterGroup, h *Handlers, auth gin.HandlerFunc) {
	group := r.Group("/auth")
	{
		// 无需认证
		group.POST("/register", h.Auth.Register)
		group.POST("/login", h.Auth.Login)
		// 走 Cookie 校验，不需要 access token
		group.POST("/refresh", h.Auth.Refresh)
		// 密码重置（邮箱验证码），无需认证
		group.POST("/password/reset-code", h.Auth.SendResetCode)
		group.POST("/password/reset", h.Auth.ResetPassword)
		// 邮箱验证码（注册校验 / 补验证 / 免密登录），无需认证
		group.POST("/email/send-code", h.Auth.SendVerifyCode)
		group.POST("/email/verify", h.Auth.VerifyEmail)
		group.POST("/login/code", h.Auth.LoginWithCode)
		// 需要 access token
		group.POST("/logout", auth, h.Auth.Logout)
		// 登录二次验证（使用 mfa_ticket，无需 access token）
		group.POST("/mfa/verify-login", h.Auth.VerifyMFALogin)
	}

	// 双因素认证管理（需登录）
	mfa := r.Group("/mfa")
	{
		mfa.POST("/setup", auth, h.Auth.SetupMFA)
		mfa.POST("/verify", auth, h.Auth.VerifyMFA)
		mfa.POST("/disable", auth, h.Auth.DisableMFA)
		mfa.POST("/recovery-codes", auth, h.Auth.RegenerateRecoveryCodes)
		mfa.GET("/status", auth, h.Auth.GetMFAStatus)
	}
}
