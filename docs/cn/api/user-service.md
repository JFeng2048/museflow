# user-service 接口说明

用户与认证核心服务（gRPC，`:5002`）。账号体系、双令牌认证、邮箱验证码、两步验证（TOTP）、RBAC、审计、OAuth 关联均在此实现。HTTP 不直接暴露，由 api-gateway 转发。

## 作用

- **账号生命周期**：注册、资料修改、用户/角色后台查询。
- **双令牌认证**：登录签发 access（短效）+ refresh（长效，HttpOnly Cookie）；刷新、登出、按设备吊销。
- **邮箱验证码**：注册校验（`register`）、补验证（`verify`）、免密登录（`login`）三类场景，验证码按场景隔离并设重发冷却。
- **两步验证**：开启/校验/关闭 TOTP，提供一次性恢复码；登录启用后返回 `mfa_ticket`。
- **密码与权限**：改密、密码重置（邮箱验证码）、RBAC 角色与权限（带 Redis 缓存）。
- **审计**：关键操作（注册、登录、改密、邮箱验证、2FA 变更等）落库审计。

## 接口（gRPC 方法）

完整契约见 `proto/user/user.proto`。主要方法：

- 认证：`Register` / `Login` / `RefreshToken` / `Logout` / `ChangePassword`
- 邮箱：`SendVerifyCode` / `VerifyEmail` / `LoginWithCode`
- 两步验证：`EnableMFA` / `VerifyMFA` / `DisableMFA` / `GenerateRecoveryCodes` / `RegenerateRecoveryCodes`
- 会话：`ListSessions` / `RevokeSession`
- 后台：`ListUsers` / `GetUser` / `AssignRole` / `ListRoles`

错误码统一由 `pkg/errcode` 维护（中英双语），未知邮箱在登录/验证码场景返回与错误凭证一致的错误以防枚举。
