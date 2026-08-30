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
| POST | `/api/v1/auth/login/code` | No | 邮箱验证码免密登录；复用双令牌签发，兼容 2FA 票据返回 |
| POST | `/api/v1/auth/mfa/enable` | Yes | 开启两步验证；返回 TOTP 密钥与二维码 URI，需 `verify` 确认 |
| POST | `/api/v1/auth/mfa/verify` | Yes | 校验 TOTP 或一次性恢复码；登录态下用于确认开启/关闭，或登录流程中换发令牌 |
| POST | `/api/v1/auth/mfa/disable` | Yes | 关闭两步验证 |
| GET  | `/api/v1/auth/mfa/recovery-codes` | Yes | 获取一次性恢复码（用于丢失 TOTP 设备时登录） |
| POST | `/api/v1/auth/password/reset` | No | 用验证码重置密码；验证码一次性消费。验证码需先经 `send-code`（`scene=reset_password`）获取 |
| GET  | `/api/v1/auth/sessions` | Yes | 当前用户的活跃会话（设备）列表 |
| DELETE | `/api/v1/auth/sessions/:id` | Yes | 吊销指定会话（设备） |
| POST | `/api/v1/common/email/send-code` | No | 发送邮箱验证码；`scene` 取 `register`/`login`/`reset_password`/`change_email`，带重发冷却，避免账号枚举。需携带 `captcha_token`（Cloudflare Turnstile，一次性）。邮件异步发送，返回 `202` + `task_id`（`expires_in` 为验证码有效期秒数） |
| GET | `/api/v1/common/tasks/{task_id}/stream` | No | SSE 订阅发送进度；事件名为 `pending`/`sending`/`retrying`/`success`/`failed`，收到终态后服务端关闭连接 |
| POST | `/api/v1/common/refresh` | No (Cookie) | 用 refresh Cookie 换取新 access token；刷新后旧 refresh 轮转 |
| GET  | `/api/v1/user/profile` | Yes | 获取当前用户资料 |
| POST | `/api/v1/user/email/change` | Yes | 修改邮箱；先经 `send-code`（`scene=change_email`）向新邮箱取码，校验通过后更新邮箱并标记已验证；新邮箱不可被其他账号占用 |
| GET  | `/health` | No | 健康检查 |
| GET  | `/swagger/index.html` | No | Swagger 文档 UI |

> 网关本身不持有用户数据，所有业务校验与令牌签发均在 user-service 完成；`Auth` 列标注 `Yes` 的接口需在请求头携带 `Authorization: Bearer <access_token>`。

## 人机验证（Cloudflare Turnstile）

`POST /common/email/send-code` 受人机验证保护：请求体需携带 `captcha_token`，后端在生成验证码前调用 Cloudflare siteverify 核验。

```jsonc
{
  "email": "author@museflow.ai",
  "scene": "register",
  "captcha_token": "0.xxxxxxxxxxxxxxxxxxxxxxxx..."   // 一次性，来自前端 widget
}
```

行为要点：

| 场景 | HTTP | 说明 |
| :--- | :--- | :--- |
| 令牌缺失 | 403 | 前端需拉起 widget 让用户验证 |
| 令牌无效 / 已用过 | 403 | 令牌**一次性**，前端每次发送都必须重新获取并重置 widget |
| action 或 hostname 不匹配 | 403 | 令牌非本站本次场景产生 |
| 校验服务不可用 | 503 | fail-closed，可重试 |

未通过人机验证时不会生成验证码、不会占用重发冷却、不会投递发信任务。
服务端未配置密钥时降级为跳过（仅适用于本地开发，启动会输出告警）。

## 邮件发送进度（SSE）

发送验证码是异步的：接口投递 asynq 任务后立即返回 `202`，邮件由 user-service 的 Worker 并发发出。前端可凭返回的 `task_id` 订阅进度：

```js
const { task_id } = await sendCode({ email, scene })

const es = new EventSource(`/api/v1/common/tasks/${task_id}/stream`)
es.addEventListener('success', (e) => {
  const data = JSON.parse(e.data) // { task_id, status, message, updated_at }
  showTip(data.message)           // 验证码已发送，请查收邮件
  es.close()
})
es.addEventListener('failed', (e) => {
  showError(JSON.parse(e.data).message) // 邮件发送失败，请稍后重试
  es.close()
})
```

事件字段：`task_id`、`status`、`message`（可直接展示的中文提示）、`updated_at`。服务端每 15 秒发送一次注释行心跳保活，连接断开时浏览器会按 `retry` 字段（3 秒）自动重连。
