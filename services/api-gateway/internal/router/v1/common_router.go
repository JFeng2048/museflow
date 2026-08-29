package v1

import (
	"github.com/gin-gonic/gin"
)

// registerCommonRoutes 注册公开通用路由（/api/v1/common）。
//
// 这些接口无需 access token，但属于对外公开的基础能力：
//   - /email/send-code 发送邮箱验证码（注册校验 / 验证码登录 / 密码重置 / 修改邮箱）
//   - /refresh        通过 HttpOnly Cookie 刷新访问令牌
func registerCommonRoutes(r *gin.RouterGroup, h *Handlers) {
	group := r.Group("/common")
	{
		group.POST("/email/send-code", h.Common.SendVerifyCode)
		group.POST("/refresh", h.Common.Refresh)
	}
}
