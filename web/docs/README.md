# 前端接口对接情况

本文档说明 `web/` 前端与后端 `api-gateway` 的接口对接现状：哪些已打通、哪些仍是本地 mock、
以及后端还缺哪些接口。对接时以本文档为准，避免重复排查。

> 后端路由以 `services/api-gateway/internal/router/` 为准，前端调用见 `src/api/`。

---

## 一、总览

| 模块 | 状态 | 说明 |
|---|---|---|
| 认证（登录/注册/登出/重置密码） | ✅ 已对接 | `/auth/*` |
| 邮箱验证码 | ✅ 已对接 | `/common/email/send-code` |
| 个人资料 / 改密 / 改邮箱 | ✅ 已对接 | `/user/*` |
| 两步验证（2FA / TOTP） | ✅ 已对接 | `/mfa/*` |
| 会话管理 | ✅ 已对接 | `/user/sessions` |
| 第三方账号（查看 / 解绑） | ✅ 已对接 | `/user/oauth`（绑定未开放） |
| 管理后台（用户/角色/权限/审计日志） | ✅ 已对接 | `/admin/*` |
| 作品（novel） | ⚠️ mock | 无后端服务 |
| 生成任务（generation） | ⚠️ mock | 无后端服务 |
| 素材（material） | ⚠️ mock | 接口已封装但后端无路由，请求必然失败后回落 |
| 发布渠道（publish） | ⚠️ mock | 同上 |
| 灵感 / 设定集 / 统计 / 积分 / 模型 | ⚠️ mock | 无后端服务 |

**已实现的服务**只有 `user-service`（Go）、`crawl4ai-service`（Python）和 `api-gateway`。
**没有** novel-service、generation-service，因此作品、生成、素材、发布等模块的 mock 属于"后端先做"，不是前端漏接。

---

## 二、传输层约定

所有请求经 `src/utils/request.ts` 统一处理，业务层只写相对路径。

| 项 | 约定 |
|---|---|
| 基础路径 | `/api/v1`（来自 `VITE_API_BASE_URL`，网关全局前缀） |
| 请求头 | `Content-Type: application/json`；有令牌时自动带 `Authorization: Bearer <token>` |
| 凭证 | `withCredentials: true`（刷新令牌为 HttpOnly Cookie） |
| 响应信封 | `{ code, message, data }`，由 `unwrap()` 拆包后**直接返回 `data`** |
| 成功判定 | `code` 落在 `[2000, 3000)`，即 `CodeSuccess=2000 / Created=2001 / Accepted=2002` |
| 失败 | 抛 `Error`，附带 `code` 与 HTTP `status`，消息取后端 `message` |
| 401 自动刷新 | 遇 401 先用 `/common/refresh` 换新令牌再重试一次；并发请求共享同一次刷新 |
| 不重试清单 | `/common/refresh`、`/auth/login`、`/auth/register`、`/auth/mfa/verify-login`、`/auth/password/reset`、`/auth/logout` —— 这些接口返回 401 是正常业务结果（密码错误等），不能被刷新掩盖 |
| 令牌存储 | `localStorage` 键 `mf.token`，统一在 `src/constants/auth.ts` 定义 |

**字段命名**：后端 DTO 用 `snake_case` 且时间为 Unix 秒；前端领域模型用 `camelCase` 与 ISO 字符串。
转换集中在 `src/api/` 各模块内（`mapUser`、`toISO` 等），视图与 store 不感知后端字段命名。

---

## 三、已对接接口

### 3.1 认证 `/auth`

| 前端函数 | 方法 | 路径 | 认证 |
|---|---|---|---|
| `login` | POST | `/auth/login` | 否 |
| `loginWithCode` | POST | `/auth/login/code` | 否 |
| `verifyMfaLogin` | POST | `/auth/mfa/verify-login` | 否（用 mfa_ticket） |
| `register` | POST | `/auth/register` | 否 |
| `logout` | POST | `/auth/logout` | 是 |
| `resetPassword` | POST | `/auth/password/reset` | 否 |

注意事项：

- **注册不返回令牌**。后端 `/auth/register` 只回 `UserInfo`，因此 `register()` 在注册成功后
  自动调一次 `login()` 换取令牌；若账号需先完成邮箱验证才能登录，则返回空令牌，
  由 `Register.vue` 引导用户去登录页。
- **登出是"先本地、后服务端"**：立即清空本地状态让界面马上响应，再用旧令牌后台通知服务端失效。
  所以 `logout(token?)` 支持显式传令牌。
- `login` 的 `username` 字段实际映射到后端 `email`（前端表单兼容邮箱/用户名两种输入习惯）。

### 3.2 通用 `/common`

| 前端函数 | 方法 | 路径 | 说明 |
|---|---|---|---|
| `sendCode` | POST | `/common/email/send-code` | 异步发送，返回 `task_id` + `expires_in`（HTTP 202） |
| — | POST | `/common/refresh` | 由 `request.ts` 在 401 时自动调用，业务层无需关心 |
| — | GET | `/common/tasks/:task_id/stream` | **未接入**，SSE 订阅邮件发送进度 |

`sendCode` 的 `scene` 取值：`register` | `login` | `reset_password` | `change_email`，
人机验证令牌字段为 `captcha_token`（Cloudflare Turnstile）。

### 3.3 用户 `/user`（需登录）

| 前端函数 | 方法 | 路径 |
|---|---|---|
| `fetchProfile` | GET | `/user/profile` |
| `updateProfile` | PUT | `/user/profile` |
| `changePassword` | PUT | `/user/password` |
| `changeEmail` | POST | `/user/email/change` |
| `fetchMyPermissions` | GET | `/user/permissions` |
| `listSessions` | GET | `/user/sessions` |
| `revokeSession` | DELETE | `/user/sessions/:token` |
| `listOAuthBindings` | GET | `/user/oauth` |
| `unbindProvider` | DELETE | `/user/oauth/:provider` |
| `bindProvider` | — | **后端未开放**，前端直接抛错提示 |

后端字段：`old_password` / `new_password`、`new_email`、`nickname` / `avatar_url` / `bio`。

### 3.4 两步验证 `/mfa`（需登录）

| 前端函数 | 方法 | 路径 |
|---|---|---|
| `setupMfa` | POST | `/mfa/setup` |
| `verifyMfa` | POST | `/mfa/verify` |
| `disableMfa` | POST | `/mfa/disable` |
| `regenerateRecoveryCodes` | POST | `/mfa/recovery-codes` |
| `getMfaStatus` | GET | `/mfa/status` |

### 3.5 管理后台 `/admin`（需登录 + `user:admin` 权限码）

权限由网关 `RequirePermission` 中间件校验（用 token 中的 `user_uuid` 调 user-service `CheckPermission`），
**前端不做权限判定**，无权限时后端返回 403。

| 前端函数 | 方法 | 路径 |
|---|---|---|
| `listUsers` | GET | `/admin/users` |
| `getUserDetail` | GET | `/admin/users/:uuid` |
| `updateUserStatus` | PUT | `/admin/users/:uuid/status` |
| `assignRole` | PUT | `/admin/users/:uuid/role` |
| `listRoles` | GET | `/admin/roles` |
| `createRole` | POST | `/admin/roles` |
| `updateRole` | PUT | `/admin/roles/:id` |
| `deleteRole` | DELETE | `/admin/roles/:id` |
| `setRolePermissions` | PUT | `/admin/roles/:id/permissions` |
| `listPermissions` | GET | `/admin/permissions` |
| `listAuditLogs` | GET | `/admin/audit-logs` |

查询参数：`page` / `page_size` / `keyword` / `status` / `order_by` / `desc`（用户列表），
`user_uuid` / `action` / `from` / `to`（审计日志，时间为 Unix 秒）。

后端字段：`role_code`、`permission_codes`、`status`。
用户状态取值：`1`=正常 `2`=冻结 `3`=已注销 `4`=待审核。

对应页面：

- `views/admin/Users.vue` —— 真实数据，服务端分页
- `views/admin/Roles.vue` —— 真实数据（角色增删改 + 权限分配）
- `views/admin/Logs.vue` —— 真实数据（审计日志）

---

## 四、仍是 mock 的模块

### 4.1 页面直接读本地 mock

| 页面 | 数据来源 |
|---|---|
| `views/admin/Dashboard.vue` | `@/mock/admin`（`adminMetrics`、`adminServices`） |
| `views/admin/Models.vue` | `@/mock/admin`（`adminModels`） |
| `views/admin/Announcements.vue` | `@/mock/admin`（`adminAnnouncements`） |
| `views/admin/Services.vue` | `@/mock/admin`（`adminServices`） |
| `views/lorebook/index.vue` | `@/mock`（`characters`、`worlds`、`foreshadows`） |
| `views/inspiration/index.vue` | `@/mock/materials`、`@/mock/trending` |

### 4.2 Store 直接读本地 mock

| Store | 数据来源 |
|---|---|
| `stores/novel.ts` | `@/mock`（`novels`） |
| `stores/generation.ts` | `@/mock`（`tasks`） |
| `stores/credit.ts` | `@/mock/credits` |
| `stores/model.ts` | `@/mock/models` |

因此 `views/novel`、`views/dashboard`、`views/task`、`views/statistics` 走的都是 store → mock。

### 4.3 已封装接口但后端无路由（请求必然失败后回落）

| 模块 | 接口 | 调用的后端路径 |
|---|---|---|
| `api/material` | `fetchMaterials` / `importMaterial` | `/materials` |
| `api/publish` | `fetchChannels` | `/publish/channels` |
| `api/novel` | `fetchNovels` / `fetchNovel` / `createNovel` | `/novels`、`/novels/:id` |
| `api/generation` | `fetchTasks` / `createTask` | `/tasks` |

这几个模块形如 `request.get(...).catch(() => mockData)`：网关没有对应路由，**请求必定失败并回落到 mock**。
注意 `api/novel` 与 `api/generation` 目前是**死代码**——`stores/novel.ts` 与 `stores/generation.ts`
直接 import `@/mock`，根本没有调用这两个模块。

> 这类 `.catch` 回落是历史遗留，会掩盖真实错误。已对接的模块（认证、用户、管理后台）已全部移除，
> 剩余模块等后端路由就绪后也应一并删除。

---

## 五、已知缺口

1. **第三方账号绑定未开放**：后端只有绑定列表与解绑，绑定需走服务端 OAuth 重定向且未暴露给前端。
   `bindProvider()` 目前直接抛错提示。
2. **无法查询角色已拥有权限**：后端只提供 `PUT /admin/roles/:id/permissions`（覆盖式保存），
   没有查询接口。因此 `Roles.vue` 权限弹窗每次打开都是空勾选，保存即覆盖。
   建议后端补一个 `GET /admin/roles/:id/permissions`。
3. **邮件发送进度未可视化**：`sendCode` 返回 `task_id`，后端提供
   `GET /common/tasks/:task_id/stream`（SSE）可订阅发送进度，前端尚未接入。
4. **管理员建号无接口**：后端没有管理员创建用户的路由，`Users.vue` 的"创建用户"入口已移除。

---

## 六、本地开发

```bash
cd web
pnpm install
pnpm dev
```

- dev 代理：`vite.config.ts` 将 `/api` 转发到 `http://localhost:5001`（网关端口），
  可用 `VITE_PROXY_TARGET` 覆盖。**不做 rewrite**，`/api` 前缀原样透传。
- `VITE_ENABLE_MOCK` 目前只影响两处（认证/用户/管理后台的**数据** mock 已删除，与它无关）：
  1. 顶栏是否显示 "MOCK" 角标（纯展示）；
  2. `TurnstileWidget` 的 `allow-fallback` —— **开启时登录/注册的人机验证允许失败降级，等于跳过校验**。

  因此生产环境务必保持 `false`，否则人机验证形同虚设。
- 后端：`cd services/api-gateway && go run ./cmd/server`（网关 `:5001`），
  另需启动 `user-service`（gRPC `:5002`）、PostgreSQL、Redis。

改动前端后请跑 `pnpm build`（含 `vue-tsc` 类型检查）再提交。
