# api-gateway 接口说明

统一 HTTP 入口网关（Gin，`:5001`）。对外暴露 RESTful JSON，内部以 gRPC 转发到 user-service，并内聚认证、CORS、访问日志与请求链路标识。Swagger 文档在 `/swagger/index.html`。

## 作用

- **协议转换**：把外部 HTTP/JSON 请求转为后端 gRPC 调用，并把 proto 响应转回 JSON。
- **统一认证**：校验 access token（与 user-service 共享 `JWT_SECRET`）；管理 refresh token 的 HttpOnly Cookie；兼容 2FA 票据（`mfa_ticket`）返回。
- **统一错误**：把 gRPC status 映射为 HTTP 状态码 + 中英双语 `Response{code,message,data}`。
- **横切能力**：CORS、访问日志、请求 ID 注入日志上下文，便于跨服务追踪。
- **路由分组**：认证相关集中在 `/api/v1/auth/*`，用户资料在 `/api/v1/user/*`；其中登录、注册、重置、邮箱验证码等免认证，其余需 `Authorization: Bearer`。

## 接口（HTTP 路由）

前缀 `/api/v1`，详见根 README 的「API Reference」与 Swagger。核心分组：

- 认证：`/auth/register`、`/auth/login`、`/auth/refresh`、`/auth/logout`、`/auth/change-password`
- 邮箱：`/auth/email/send-code`、`/auth/email/verify`、`/auth/login/code`
- 两步验证：`/auth/mfa/enable`、`/auth/mfa/verify`、`/auth/mfa/disable`、`/auth/mfa/recovery-codes`
- 密码重置：`/auth/password/reset-code`、`/auth/password/reset`
- 会话：`/auth/sessions`、`/auth/sessions/:id`
- 用户：`/user/profile`

> 网关本身不持有用户数据，所有业务校验与令牌签发均在 user-service 完成。
