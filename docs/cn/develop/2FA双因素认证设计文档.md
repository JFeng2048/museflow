# 2FA 双因素认证设计文档

## 概述

在双令牌认证基础上增加基于 TOTP（RFC 6238）的双因素认证，提升账号安全性。
开启 2FA 后，登录流程拆分为两步：第一步校验密码，第二步校验 TOTP 验证码。

实现载体：

| 组件 | 职责 |
|------|------|
| `services/api-gateway` | 暴露 `/auth/mfa/*` 与 `/auth/mfa/verify-login` HTTP 接口，转发 gRPC |
| `services/user-service` | `mfa` 子包生成密钥/校验验证码、`auth` 子包编排登录二次验证、repository 持久化密钥 |
| `proto/user/user.proto` | `SetupMFA` / `VerifyMFA` / `DisableMFA` / `RegenerateRecoveryCodes` / `GetMFAStatus` / `VerifyMFALogin` 六个 RPC |
| PostgreSQL `user_svc.users` | 存储 `mfa_secret`、`mfa_enabled`、`mfa_recovery_codes` |

## 核心流程

### 登录两步验证

```
1. POST /auth/login {email, password}
   ├─ 未开启 2FA → 直接返回 access/refresh 双令牌
   └─ 已开启 2FA  → 返回 mfa_ticket（不含令牌），requires_mfa=true

2. POST /auth/mfa/verify-login {mfa_ticket, code}
   ├─ code 校验失败 → 401，并记录审计
   └─ code 校验成功 → 签发 access/refresh 双令牌，写入设备会话
```

`mfa_ticket` 是一种短期 JWT（类型 `mfa_ticket`，默认 5 分钟过期），仅用于串联登录两步，
无法换取任何业务资源。票据中携带 `user_uuid`，服务端校验后据此生成正式令牌。

### 启用 2FA

```
1. POST /auth/mfa/setup      → 返回 secret + otpauth_url（此时尚未启用）
2. 用户在验证器 App 扫码绑定
3. POST /auth/mfa/verify {code} → 校验通过后启用，返回 8 个恢复码
```

启用前必须完成一次 TOTP 校验，防止用户填错密钥导致永久锁死。

### 恢复码

开启 2FA 时生成 8 个单次使用的恢复码（默认长度 10）。任一恢复码可在
`/auth/mfa/verify-login` 中代替 TOTP 验证码使用，使用后立即作废并补充新码，
始终保持 8 个可用。

## 配置项

| 配置键 | 默认值 | 说明 |
|--------|--------|------|
| `MFA_TICKET_TTL_SECONDS` | 300 | 登录中间票据有效期（秒） |
| `MFA_ISSUER` | MuseFlow | 验证器 App 中展示的发行方 |
| `MFA_SKEW` | 1 | 允许的时钟偏移步数（每步 30 秒） |
| `MFA_RECOVERY_CODE_COUNT` | 8 | 恢复码数量 |
| `MFA_RECOVERY_CODE_LENGTH` | 10 | 单个恢复码长度 |

## 安全要点

- `mfa_secret` 以明文存于数据库（加密方案可后续增强），仅服务端可见，绝不返回前端。
- `otpauth_url` 仅在 `setup` 阶段返回一次，用于扫码；不持久化。
- 关闭 2FA 同样需要输入当前 TOTP 验证码，防止攻击者趁会话未失效时关闭保护。
- 验证失败计入审计（`mfa_verify_fail`），便于安全追溯。
