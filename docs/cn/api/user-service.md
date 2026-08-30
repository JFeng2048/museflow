# user-service 接口说明

用户与认证核心服务（gRPC，`:5002`）。账号体系、双令牌认证、邮箱验证码、两步验证（TOTP）、RBAC、审计、OAuth 关联均在此实现。HTTP 不直接暴露，由 api-gateway 转发。

## 作用

- **账号生命周期**：注册、资料修改、用户/角色后台查询。
- **双令牌认证**：登录签发 access（短效）+ refresh（长效，HttpOnly Cookie）；刷新、登出、按设备吊销。
- **邮箱验证码**：注册校验（`register`）、免密登录（`login`）、密码重置（`reset_password`）、修改邮箱（`change_email`）四类场景，验证码按场景隔离并设重发冷却。
- **异步邮件与进度订阅**：邮件发送经 asynq 队列异步化，`SendVerifyCode` 只生成验证码并入队（返回 `task_id`），由独立 Worker 进程消费；`WatchTask`（服务端流）用于订阅发送进度，供网关转 SSE。
- **人机验证**：`SendVerifyCode` 是易被脚本刷的接口，服务端在生成验证码前用 Cloudflare Turnstile 核验 `captcha_token`（`internal/pkg/turnstile`）。未通过则不生成验证码、不占冷却、不入队；未配置密钥时降级为跳过，校验服务故障时 fail-closed。
- **两步验证**：开启/校验/关闭 TOTP，提供一次性恢复码；登录启用后返回 `mfa_ticket`。
- **密码与权限**：改密、密码重置（邮箱验证码）、RBAC 角色与权限（带 Redis 缓存）。
- **审计**：关键操作（注册、登录、改密、邮箱验证、修改邮箱、2FA 变更等）落库审计。

## 接口（gRPC 方法）

完整契约见 `proto/user/user.proto`。主要方法：

- 认证：`Register` / `Login` / `RefreshToken` / `Logout` / `ChangePassword`
- 邮箱：`SendVerifyCode` / `WatchTask` / `LoginWithCode` / `ChangeEmail`（修改邮箱，需登录）
- 两步验证：`EnableMFA` / `VerifyMFA` / `DisableMFA` / `GenerateRecoveryCodes` / `RegenerateRecoveryCodes`
- 会话：`ListSessions` / `RevokeSession`
- 后台：`ListUsers` / `GetUser` / `AssignRole` / `ListRoles`

错误码统一由 `pkg/errcode` 维护（中英双语），未知邮箱在登录/验证码场景返回与错误凭证一致的错误以防枚举。

## 异步任务进度

`SendVerifyCode` 返回 `task_id` 与 `expires_in`，不等待邮件真正发出。调用方用 `WatchTask` 订阅：

| 字段 | 说明 |
| :--- | :--- |
| `status` | `pending`（已入队）/ `sending`（发送中）/ `retrying`（重试中）/ `success` / `failed` |
| `message` | 面向用户的友好提示，可直接展示 |
| `updated_at` | 状态更新时间（Unix 秒） |

`success` 与 `failed` 为终态，服务端在推送后主动结束流。任务状态在 Redis 中保留 `USER_QUEUE_STATUS_TTL_SECONDS`（默认 600 秒），过期后 `WatchTask` 返回 `NotFound`。
