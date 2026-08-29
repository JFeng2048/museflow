package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/museflow/api-gateway/internal/client"
	"github.com/museflow/pkg/errcode"
	userpb "github.com/museflow/proto/user"
)

// RequirePermission 权限校验中间件。
//
// 校验流程与需求一致：
//  1. 从 gin 上下文取出当前用户 uuid（由 Auth 中间件从 token 解析后写入）
//  2. 调用 user-service 的 CheckPermission，传入路由声明的权限码
//  3. user-service 内部优先读 Redis 缓存（perm:user:{uuid}），
//     未命中则查库回填；具备权限返回 true
//  4. 无权限返回 403
//
// 用法（在需要权限的路由上声明）：
//
//	admin := v1.Group("/admin", middleware.Auth(userClient))
//	admin.GET("/users", middleware.RequirePermission(userClient, "user:read"), handler.ListUsers)
func RequirePermission(userClient *client.UserClient, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userUUID := CurrentUserUUID(c)
		if userUUID == "" {
			// 未经过 Auth 中间件，无法确定身份
			c.AbortWithStatusJSON(http.StatusUnauthorized, errcode.ErrorGin(c, errcode.CodeMissingToken))
			return
		}

		resp, err := userClient.Service().CheckPermission(c.Request.Context(), &userpb.CheckPermissionRequest{
			Uuid:       userUUID,
			Permission: permission,
		})
		if err != nil {
			// 用户服务不可用，按服务端错误处理
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, errcode.ErrorGin(c, errcode.CodeServiceUnavail))
			return
		}

		if !resp.GetAllowed() {
			c.AbortWithStatusJSON(http.StatusForbidden, errcode.ErrorGin(c, errcode.CodeForbidden))
			return
		}

		c.Next()
	}
}

// RequireAnyPermission 满足任意一个权限即可通过（用于「读或写都放行」的场景）。
func RequireAnyPermission(userClient *client.UserClient, permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userUUID := CurrentUserUUID(c)
		if userUUID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errcode.ErrorGin(c, errcode.CodeMissingToken))
			return
		}

		for _, p := range permissions {
			resp, err := userClient.Service().CheckPermission(c.Request.Context(), &userpb.CheckPermissionRequest{
				Uuid:       userUUID,
				Permission: p,
			})
			if err != nil {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, errcode.ErrorGin(c, errcode.CodeServiceUnavail))
				return
			}
			if resp.GetAllowed() {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, errcode.ErrorGin(c, errcode.CodeForbidden))
	}
}
