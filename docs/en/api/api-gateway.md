# api-gateway API

The unified HTTP entry-point gateway (Gin, `:5001`). Exposes RESTful JSON externally, proxies to user-service over gRPC internally, and owns auth, CORS, access logging, and request tracing. Swagger UI at `/swagger/index.html`.

## Purpose

- **Protocol translation**: turn external HTTP/JSON into backend gRPC calls and convert proto responses back to JSON.
- **Unified auth**: verify access token (shares `JWT_SECRET` with user-service); manage the refresh-token HttpOnly Cookie; compatible with 2FA ticket (`mfa_ticket`) responses.
- **Unified errors**: map gRPC status to HTTP status + bilingual `Response{code,message,data}`.
- **Cross-cutting**: CORS, access logging, request-id injection into log context for traceability.
- **Route grouping**: auth under `/api/v1/auth/*`, user profile under `/api/v1/user/*`; login/register/reset/email-code are public, the rest require `Authorization: Bearer`.

## Interface (HTTP routes)

Prefix `/api/v1`. Full fields and examples in the gateway Swagger (`/swagger/index.html`).

| Method | Path | Auth | Purpose |
| :--- | :--- | :--- | :--- |
| POST | `/api/v1/auth/register` | No | Register; body needs email code `code` (get it from `send-code` first); account marked verified on success |
| POST | `/api/v1/auth/login` | No | Password login; issues access (body) + refresh (HttpOnly Cookie). Returns `mfa_ticket` instead of tokens when 2FA is on |
| POST | `/api/v1/auth/refresh` | No (Cookie) | Exchange refresh Cookie for a new access token; refresh is not rotated |
| POST | `/api/v1/auth/logout` | No (Cookie) | Logout; revoke current device's refresh and add access to deny-list |
| POST | `/api/v1/auth/change-password` | Yes | Change password; requires old password |
| POST | `/api/v1/auth/email/send-code` | No | Send email code; `scene` is `register`/`verify`/`login`, with resend cooldown, avoids enumeration |
| POST | `/api/v1/auth/email/verify` | No | Verify email code (catch-up); marks `email_verified` on success |
| POST | `/api/v1/auth/login/code` | No | Passwordless email-code login; reuses dual-token issuance, compatible with 2FA ticket |
| POST | `/api/v1/auth/mfa/enable` | Yes | Enable 2FA; returns TOTP secret + QR URI, needs `verify` to confirm |
| POST | `/api/v1/auth/mfa/verify` | Yes | Verify TOTP or one-time recovery code; confirms enable/disable, or exchanges ticket during login |
| POST | `/api/v1/auth/mfa/disable` | Yes | Disable 2FA |
| GET  | `/api/v1/auth/mfa/recovery-codes` | Yes | Get one-time recovery codes (for login when TOTP device is lost) |
| POST | `/api/v1/auth/password/reset-code` | No | Send password-reset code to email |
| POST | `/api/v1/auth/password/reset` | No | Reset password with code; code is single-use |
| GET  | `/api/v1/auth/sessions` | Yes | Active session (device) list for current user |
| DELETE | `/api/v1/auth/sessions/:id` | Yes | Revoke a session (device) |
| GET  | `/api/v1/user/profile` | Yes | Get current user profile |
| GET  | `/health` | No | Health check |
| GET  | `/swagger/index.html` | No | Swagger UI |

> The gateway holds no user data itself; all business checks and token issuance happen in user-service. Interfaces marked `Yes` in the `Auth` column require `Authorization: Bearer <access_token>` in the request header.
