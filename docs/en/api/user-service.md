# user-service API

The core user & authentication service (gRPC, `:5002`). Account management, dual-token auth, email verification codes, TOTP 2FA, RBAC, audit, and OAuth linking all live here. It is not exposed directly over HTTP; api-gateway proxies to it.

## Purpose

- **Account lifecycle**: register, profile update, user/role admin queries.
- **Dual-token auth**: login issues access (short-lived) + refresh (long-lived, HttpOnly Cookie); refresh, logout, per-device revocation.
- **Email codes**: four scenes — register verification, passwordless login, password reset, change email — isolated per scene with resend cooldown.
- **Async delivery & progress**: email sending is offloaded to an Asynq queue; `SendVerifyCode` only generates the code and enqueues a task (returning a `task_id`) that a separate worker process consumes. `WatchTask` (server stream) subscribes to delivery progress so the gateway can relay it as SSE.
- **2FA**: enable/verify/disable TOTP, plus one-time recovery codes; returns `mfa_ticket` when enabled at login.
- **Password & permissions**: change password, password reset (email code), RBAC roles & permissions (Redis-cached).
- **Audit**: key operations (register, login, change password, email verify, change email, 2FA changes, etc.) persisted to audit log.

## Interface (gRPC methods)

Full contract in `proto/user/user.proto`. Main methods:

- Auth: `Register` / `Login` / `RefreshToken` / `Logout` / `ChangePassword`
- Email: `SendVerifyCode` / `WatchTask` / `LoginWithCode` / `ChangeEmail` (change email, requires login)
- 2FA: `EnableMFA` / `VerifyMFA` / `DisableMFA` / `GenerateRecoveryCodes` / `RegenerateRecoveryCodes`
- Sessions: `ListSessions` / `RevokeSession`
- Admin: `ListUsers` / `GetUser` / `AssignRole` / `ListRoles`

Error codes are unified in `pkg/errcode` (bilingual zh/en); unknown emails return the same error as wrong credentials on login/code flows to prevent enumeration.

## Async task progress

`SendVerifyCode` returns a `task_id` and `expires_in` without waiting for actual delivery. Callers subscribe via `WatchTask`:

| Field | Description |
| :--- | :--- |
| `status` | `pending` (queued) / `sending` / `retrying` / `success` / `failed` |
| `message` | Display-ready user-facing text |
| `updated_at` | Status update time (Unix seconds) |

`success` and `failed` are terminal: the server closes the stream after emitting them. Task status is retained in Redis for `USER_QUEUE_STATUS_TTL_SECONDS` (default 600s); once expired, `WatchTask` returns `NotFound`.
