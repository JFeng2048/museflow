# user-service API

The core user & authentication service (gRPC, `:5002`). Account management, dual-token auth, email verification codes, TOTP 2FA, RBAC, audit, and OAuth linking all live here. It is not exposed directly over HTTP; api-gateway proxies to it.

## Purpose

- **Account lifecycle**: register, profile update, user/role admin queries.
- **Dual-token auth**: login issues access (short-lived) + refresh (long-lived, HttpOnly Cookie); refresh, logout, per-device revocation.
- **Email codes**: three scenes — register verification, catch-up verify, passwordless login — isolated per scene with resend cooldown.
- **2FA**: enable/verify/disable TOTP, plus one-time recovery codes; returns `mfa_ticket` when enabled at login.
- **Password & permissions**: change password, password reset (email code), RBAC roles & permissions (Redis-cached).
- **Audit**: key operations (register, login, change password, email verify, 2FA changes, etc.) persisted to audit log.

## Interface (gRPC methods)

Full contract in `proto/user/user.proto`. Main methods:

- Auth: `Register` / `Login` / `RefreshToken` / `Logout` / `ChangePassword`
- Email: `SendVerifyCode` / `VerifyEmail` / `LoginWithCode`
- 2FA: `EnableMFA` / `VerifyMFA` / `DisableMFA` / `GenerateRecoveryCodes` / `RegenerateRecoveryCodes`
- Sessions: `ListSessions` / `RevokeSession`
- Admin: `ListUsers` / `GetUser` / `AssignRole` / `ListRoles`

Error codes are unified in `pkg/errcode` (bilingual zh/en); unknown emails return the same error as wrong credentials on login/code flows to prevent enumeration.
