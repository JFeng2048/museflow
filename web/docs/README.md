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
| 模型配置（后台渠道/模型/系统配置 + 用户自定义） | ✅ 已对接 | `/admin/model-providers`、`/admin/models`、`/admin/settings`、`/user/models` 等 |
| 作品（novel） | ⚠️ mock | 无后端服务，页面读 `@/mock` |
| 生成任务（generation） | ⚠️ mock | 无后端服务，页面读 `@/mock` |
| 素材（material） | ⚠️ mock | 无后端服务，页面读 `@/mock/materials` |
| 发布渠道（publish） | ⚠️ mock | 无后端服务，页面读 `@/mock/publish` |
| 灵感 / 设定集 / 统计 / 积分 | ⚠️ mock | 无后端服务，页面读 `@/mock/*` |

**已实现的服务**包括 `user-service`（Go）、`config-service`（Go）、`crawl4ai-service`（Python）和 `api-gateway`。
**没有** novel-service、generation-service，因此作品、生成、素材、发布等模块的 mock 属于"后端先做"，不是前端漏接。

标 ⚠️ mock 的页面都会在首屏挂一条 `DemoNotice`，明确告诉用户这页是本地示例、操作不会同步到账号。

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

### 3.6 模型与系统配置 `/admin/model-*`、`/user/model-*`（需登录）

配置域由 `config-service`（gRPC `:5004`）承载，网关转发，前端统一封装在 `src/api/model/index.ts`。
设计取舍与表结构见[模型与系统配置设计文档](../../docs/cn/develop/模型与系统配置设计文档.md)，
分组的 key-value 配置让新增一项可调参数不必改表。

管理端一组要的是 `system:admin`，**不是** 3.5 的 `user:admin`：平台渠道持有 `api_key`，
是平台资产，该权限目前只授给 super_admin。用户端一组只需登录，且用户标识一律取 token 中的
`user_uuid`，不接受请求体或查询参数里的 user_uuid，否则改一个参数就能操作别人的渠道。

| 前端函数 | 方法 | 路径 |
|---|---|---|
| `listProviders` | GET | `/admin/model-providers` |
| `createProvider` | POST | `/admin/model-providers` |
| `updateProvider` | PUT | `/admin/model-providers/:id` |
| `setProviderActive` | PUT | `/admin/model-providers/:id/active` |
| `deleteProvider` | DELETE | `/admin/model-providers/:id` |
| `fetchAdminRemoteModels` | POST | `/admin/model-providers/remote-models` |
| `listModels` | GET | `/admin/models` |
| `createModel` | POST | `/admin/models` |
| `updateModel` | PUT | `/admin/models/:id` |
| `setModelActive` | PUT | `/admin/models/:id/active` |
| `deleteModel` | DELETE | `/admin/models/:id` |
| `listSettings` | GET | `/admin/settings` |
| `getSetting` | GET | `/admin/settings/:config_group/:key` |
| `upsertSetting` | PUT | `/admin/settings/:config_group/:key` |
| `deleteSetting` | DELETE | `/admin/settings/:config_group/:key` |
| `listUserProviders` | GET | `/user/model-providers` |
| `createUserProvider` | POST | `/user/model-providers` |
| `updateUserProvider` | PUT | `/user/model-providers/:id` |
| `deleteUserProvider` | DELETE | `/user/model-providers/:id` |
| `fetchUserRemoteModels` | POST | `/user/model-providers/remote-models` |
| `listUserModels` | GET | `/user/models` |
| `createUserModel` | POST | `/user/models` |
| `updateUserModel` | PUT | `/user/models/:id` |
| `deleteUserModel` | DELETE | `/user/models/:id` |
| `listAvailableModels` | GET | `/user/models/available` |
| `listPublicSettings` | GET | `/user/settings` |

密钥约定：`api_key` 与 `secret_value` 都是只写字段，写入后任何查询接口都不回明文，响应里只有
`api_key_hint`（末 4 位）与 `api_key_updated_at`；写入响应会额外回一次明文，仅用于确认填对了什么。
`base_url` 对管理端可见（排查连通性要用）。前端用 `KeySlot`（`components/model/KeySlot.vue`）
承载这个"只写不读"的录入交互。

对应页面：`views/admin/Models.vue`（平台渠道 / 平台模型）、`views/admin/Settings.vue`（系统配置）、
`views/settings/ModelSettings.vue`（用户自定义渠道与模型，挂在设置的"模型" tab 下）。

---

## 四、仍是 mock 的模块

这些模块的数据全部来自本地 `@/mock`，因为 novel-service、generation-service 等服务还不存在。
页面不再发注定失败的请求，而是直接读 mock，并在首屏挂 `DemoNotice`（`components/common/DemoNotice.vue`）
提示用户这页是示例数据。

### 4.1 页面直接读本地 mock

| 页面 | 数据来源 | 演示标注 |
|---|---|---|
| `views/novel/index.vue`、`views/novel/NovelDetail.vue` | `@/mock`（经 `stores/novel.ts`） | ✅ |
| `views/dashboard/index.vue`、`views/task/index.vue`、`views/statistics/index.vue` | `@/mock`、`@/mock/credits`（经 store） | ✅ |
| `views/inspiration/index.vue`、`components/layout/InspirationDrawer.vue` | `@/mock/materials`、`@/mock/trending` | ✅（抽屉用 `demo.drawerNote`） |
| `views/lorebook/index.vue` | `@/mock`（`characters`、`worlds`、`foreshadows`） | ✅ |
| `views/material/index.vue` | `@/mock/materials`（`materialStore` 可变副本） | ✅ |
| `views/publish/index.vue`、`views/settings/index.vue` 的"发布渠道" tab | `@/mock/publish`（`channels`） | ✅ |
| `views/admin/Dashboard.vue`、`views/admin/Announcements.vue`、`views/admin/Services.vue` | `@/mock/admin` | ✅ |

`views/settings/index.vue` 只有 credits 与 publish 两个 tab 是 mock，其余 tab（资料、安全、模型）走真实接口，
因此标注只加在这两个 tab 内，不是整页。

### 4.2 Store 直接读本地 mock

| Store | 数据来源 |
|---|---|
| `stores/novel.ts` | `@/mock`（`novels`） |
| `stores/generation.ts` | `@/mock`（`tasks`） |
| `stores/credit.ts` | `@/mock/credits` |

### 4.3 已无"伪请求"回落层

`src/api/` 下曾有 `novel`、`generation`、`material`、`publish` 四个模块，写法是
`request.get(...).catch(() => mockData)`。网关没有对应路由，因此请求**必定失败**再静默回落到 mock：
404、429 和服务没接通看起来一模一样，真实错误被兜底吞掉。这层已全部删除，页面改为直接 import `@/mock`。
现在 `src/api/` 只剩三个模块，各自按域划分：`system/auth`（登录注册、验证码、个人资料、2FA、
会话、第三方账号）、`admin`（用户、角色、权限、审计日志）、`model`（渠道、模型、系统配置）。
业务侧都从这三个路径直接 import，`api/index.ts` 的 barrel 只转出 `system/auth` 与 `model`。

后续接入对应后端服务时，把 store 与页面里的 `@/mock` 换成真实接口即可，**不要**再把 `.catch` 回落捡回来；
宁可让请求报错，也别让示例数据冒充真实响应。

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
- `VITE_ALLOW_CAPTCHA_FALLBACK`：人机验证降级开关，仅本地调试用。置 `true` 时，`TurnstileWidget`
  在脚本加载失败或站点密钥为空的情况下仍允许登录/注册提交，**等于跳过人机校验**。
  生产环境务必保持 `false`，否则人机验证形同虚设。它与页面上的「演示数据」标注无关——
  后者只说明该页数据来自本地 `@/mock`，由页面自己挂 `DemoNotice`，没有全局开关。
- 后端：`cd services/api-gateway && go run ./cmd/server`（网关 `:5001`），
  另需启动 `user-service`（gRPC `:5002`）、PostgreSQL、Redis。

改动前端后请跑 `pnpm build`（含 `vue-tsc` 类型检查）再提交。
