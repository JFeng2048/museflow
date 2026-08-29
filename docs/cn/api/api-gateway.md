# api-gateway 接口说明

统一 HTTP 入口网关（Gin，`:5001`）。对外暴露 RESTful JSON，内部以 gRPC 转发到 user-service，并内聚认证、CORS、访问日志与请求链路标识。Swagger 文档在 `/swagger/index.html`。

## 作用

- **协议转换**：把外部 HTTP/JSON 请求转为后端 gRPC 调用，并把 proto 响应转回 JSON。
- **统一认证**：校验 access token（与 user-service 共享 `JWT_SECRET`）；管理 refresh token 的 HttpOnly Cookie；兼容 2FA 票据（`mfa_ticket`）返回。
- **统一错误**：把 gRPC status 映射为 HTTP 状态码 + 中英双语 `Response{code,message,data}`。
- **横切能力**：CORS、访问日志、请求 ID 注入日志上下文，便于跨服务追踪。
- **路由分组**：认证相关集中在 `/api/v1/auth/*`，用户资料在 `/api/v1/user/*`，公开通用能力（发送验证码、刷新令牌）归入 `/api/v1/common/*`；其中登录、注册、密码重置、邮箱验证码、刷新令牌等免认证，其余需 `Authorization: Bearer`。

## 接口（HTTP 路由）

前缀 `/api/v1`。完整字段与示例见网关 Swagger（`/swagger/index.html`）。

| Method | Path | Auth | 作用 |
| :--- | :--- | :--- | :--- |
| POST | `/api/v1/auth/register` | No | 注册；请求体需带邮箱验证码 `code`（先调 `send-code` 取码），成功后账号标记已验证 |
| POST | `/api/v1/auth/login` | No | 密码登录；成功签发 access（响应体）+ refresh（HttpOnly Cookie）。开启 2FA 时返回 `mfa_ticket` 而非令牌 |
| POST | `/api/v1/auth/logout` | No (Cookie) | 登出；吊销当前设备的 refresh 并加入 access 黑名单 |
| POST | `/api/v1/auth/email/send-code` | No | 发送邮箱验证码；`scene` 取 `register`/`login`/`reset_password`/`change_email`，带重发冷却，避免账号枚举 |
| POST | `/api/v1/auth/login/code` | No | 邮箱验证码免密登录；复用双令牌签发，兼容 2FA 票据返回 |
| POST | `/api/v1/auth/mfa/enable` | Yes | 开启两步验证；返回 TOTP 密钥与二维码 URI，需 `verify` 确认 |
| POST | `/api/v1/auth/mfa/verify` | Yes | 校验 TOTP 或一次性恢复码；登录态下用于确认开启/关闭，或登录流程中换发令牌 |
| POST | `/api/v1/auth/mfa/disable` | Yes | 关闭两步验证 |
| GET  | `/api/v1/auth/mfa/recovery-codes` | Yes | 获取一次性恢复码（用于丢失 TOTP 设备时登录） |
| POST | `/api/v1/auth/password/reset` | No | 用验证码重置密码；验证码一次性消费。验证码需先经 `send-code`（`scene=reset_password`）获取 |
| GET  | `/api/v1/auth/sessions` | Yes | 当前用户的活跃会话（设备）列表 |
| DELETE | `/api/v1/auth/sessions/:id` | Yes | 吊销指定会话（设备） |
| POST | `/api/v1/common/email/send-code` | No | 公开发送邮箱验证码（与 `/auth/email/send-code` 同源，统一归入 `/common` 便于调用） |
| POST | `/api/v1/common/refresh` | No (Cookie) | 用 refresh Cookie 换取新 access token；刷新后旧 refresh 轮转 |
| GET  | `/api/v1/user/profile` | Yes | 获取当前用户资料 |
| POST | `/api/v1/user/email/change` | Yes | 修改邮箱；先经 `send-code`（`scene=change_email`）向新邮箱取码，校验通过后更新邮箱并标记已验证；新邮箱不可被其他账号占用 |
| GET  | `/health` | No | 健康检查 |
| GET  | `/swagger/index.html` | No | Swagger 文档 UI |

> 网关本身不持有用户数据，所有业务校验与令牌签发均在 user-service 完成；`Auth` 列标注 `Yes` 的接口需在请求头携带 `Authorization: Bearer <access_token>`。
