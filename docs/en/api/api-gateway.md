# api-gateway API

The unified HTTP entry-point gateway (Gin, `:5001`). Exposes RESTful JSON externally, proxies to user-service over gRPC internally, and owns auth, CORS, access logging, and request tracing. Swagger UI at `/swagger/index.html`.

## Purpose

- **Protocol translation**: turn external HTTP/JSON into backend gRPC calls and convert proto responses back to JSON.
- **Unified auth**: verify access token (shares `JWT_SECRET` with user-service); manage the refresh-token HttpOnly Cookie; compatible with 2FA ticket (`mfa_ticket`) responses.
- **Unified errors**: map gRPC status to HTTP status + bilingual `Response{code,message,data}`.
- **Cross-cutting**: CORS, access logging, request-id injection into log context for traceability.
- **Route grouping**: auth under `/api/v1/auth/*`, user profile under `/api/v1/user/*`, public common capabilities (send code, refresh token) under `/api/v1/common/*`; login/register/password-reset/email-code/refresh are public, the rest require `Authorization: Bearer`.
   Models and system settings come in two flavours: `/api/v1/admin/model-*`, `/api/v1/admin/models` and `/api/v1/admin/settings/*` need login plus the `system:admin` permission, while `/api/v1/user/model*`, `/api/v1/user/models/available` and `/api/v1/user/settings` need login only.

## Interface (HTTP routes)

Prefix `/api/v1`. Full fields and examples in the gateway Swagger (`/swagger/index.html`).

| Method | Path | Auth | Purpose |
| :--- | :--- | :--- | :--- |
| POST | `/api/v1/auth/register` | No | Register; body needs email code `code` (get it from `send-code` first); account marked verified on success |
| POST | `/api/v1/auth/login` | No | Password login; issues access (body) + refresh (HttpOnly Cookie). Returns `mfa_ticket` instead of tokens when 2FA is on |
| POST | `/api/v1/auth/logout` | No (Cookie) | Logout; revoke current device's refresh and add access to deny-list |
| POST | `/api/v1/auth/login/code` | No | Passwordless email-code login; reuses dual-token issuance, compatible with 2FA ticket |
| POST | `/api/v1/auth/mfa/enable` | Yes | Enable 2FA; returns TOTP secret + QR URI, needs `verify` to confirm |
| POST | `/api/v1/auth/mfa/verify` | Yes | Verify TOTP or one-time recovery code; confirms enable/disable, or exchanges ticket during login |
| POST | `/api/v1/auth/mfa/disable` | Yes | Disable 2FA |
| GET  | `/api/v1/auth/mfa/recovery-codes` | Yes | Get one-time recovery codes (for login when TOTP device is lost) |
| POST | `/api/v1/auth/password/reset` | No | Reset password with code; code is single-use. Get the code via `send-code` (`scene=reset_password`) first |
| GET  | `/api/v1/auth/sessions` | Yes | Active session (device) list for current user |
| DELETE | `/api/v1/auth/sessions/:id` | Yes | Revoke a session (device) |
| POST | `/api/v1/common/email/send-code` | No | Send email code; `scene` is `register`/`login`/`reset_password`/`change_email`, with resend cooldown, avoids enumeration. Requires `captcha_token` (Cloudflare Turnstile, single-use). Delivery is async: returns `202` + `task_id` (and `expires_in`, the code TTL in seconds) |
| GET | `/api/v1/common/tasks/{task_id}/stream` | No | SSE stream of delivery progress; event names are `pending`/`sending`/`retrying`/`success`/`failed`. The server closes the connection after a terminal event |
| POST | `/api/v1/common/refresh` | No (Cookie) | Exchange refresh Cookie for a new access token; refresh is rotated |
| GET  | `/api/v1/user/profile` | Yes | Get current user profile |
| POST | `/api/v1/user/email/change` | Yes | Change email; get a code via `send-code` (`scene=change_email`) to the new email first, then verify to update email and mark verified; new email must not be used by another account |
| GET  | `/health` | No | Health check |
| GET  | `/swagger/index.html` | No | Swagger UI |

> The gateway holds no user data itself; all business checks and token issuance happen in user-service. Interfaces marked `Yes` in the `Auth` column require `Authorization: Bearer <access_token>` in the request header.

## Models & system settings (proxied to config-service)

This group is proxied to config-service (gRPC `:5004`) by the gateway, still under the `/api/v1` prefix. Full request/response fields are in the gateway Swagger (`/swagger/index.html`), tag `model-模型与系统配置`; key handling, tables and error codes are in [config-service API](config-service.md).

The two sides differ in auth strength: the admin group holds the platform `api_key`, so it needs `system:admin` on top of login; the user group only touches the caller's own providers and models, so login is enough.

### Platform providers (admin)

| Method | Path | Auth | Purpose |
| :--- | :--- | :--- | :--- |
| GET | `/api/v1/admin/model-providers` | Yes + `system:admin` | Provider list, `keyword` / `only_active` / `page` / `page_size` |
| POST | `/api/v1/admin/model-providers` | Yes + `system:admin` | Create provider; the `api_key` is encrypted at rest and only its last 4 characters are echoed |
| PUT | `/api/v1/admin/model-providers/:id` | Yes + `system:admin` | Edit provider; `code` is immutable, an empty `api_key` keeps the existing key |
| PUT | `/api/v1/admin/model-providers/:id/active` | Yes + `system:admin` | Enable / disable |
| DELETE | `/api/v1/admin/model-providers/:id` | Yes + `system:admin` | Delete provider; fails while models still hang under it |
| POST | `/api/v1/admin/model-providers/remote-models` | Yes + `system:admin` | Probe the provider's upstream catalog; body is one of `provider_id` (key stays server-side) or `base_url` + `api_key` |

### Platform models (admin)

| Method | Path | Auth | Purpose |
| :--- | :--- | :--- | :--- |
| GET | `/api/v1/admin/models` | Yes + `system:admin` | Model list, `provider_id` / `model_type` / `keyword` / `only_active` / pagination |
| POST | `/api/v1/admin/models` | Yes + `system:admin` | Create model; `provider_id` is required |
| PUT | `/api/v1/admin/models/:id` | Yes + `system:admin` | Edit model; `code` and the owning provider are immutable |
| PUT | `/api/v1/admin/models/:id/active` | Yes + `system:admin` | Publish / unpublish |
| DELETE | `/api/v1/admin/models/:id` | Yes + `system:admin` | Delete model |

### System settings (admin)

| Method | Path | Auth | Purpose |
| :--- | :--- | :--- | :--- |
| GET | `/api/v1/admin/settings` | Yes + `system:admin` | Setting list, `config_group` / `only_public` |
| GET | `/api/v1/admin/settings/:config_group/:key` | Yes + `system:admin` | Single setting |
| PUT | `/api/v1/admin/settings/:config_group/:key` | Yes + `system:admin` | Write setting; either `value` or `secret_value`, secret values are write-only |
| DELETE | `/api/v1/admin/settings/:config_group/:key` | Yes + `system:admin` | Delete setting |

### My providers & models (user)

| Method | Path | Auth | Purpose |
| :--- | :--- | :--- | :--- |
| GET | `/api/v1/user/model-providers` | Yes | My custom provider list |
| POST | `/api/v1/user/model-providers` | Yes | Create a custom provider (own `base_url` + `api_key`) |
| PUT | `/api/v1/user/model-providers/:id` | Yes | Edit custom provider; an empty `api_key` keeps the existing key |
| DELETE | `/api/v1/user/model-providers/:id` | Yes | Delete custom provider |
| POST | `/api/v1/user/model-providers/remote-models` | Yes | Probe my provider catalog |
| GET | `/api/v1/user/models` | Yes | My model list |
| POST | `/api/v1/user/models` | Yes | Create a custom model, only under my own providers |
| PUT | `/api/v1/user/models/:id` | Yes | Edit custom model |
| DELETE | `/api/v1/user/models/:id` | Yes | Delete custom model |
| GET | `/api/v1/user/models/available` | Yes | Available models merged view (platform + custom), `model_type` optional |
| GET | `/api/v1/user/settings` | Yes | Public system settings, only `is_public=true` entries |

## Captcha (Cloudflare Turnstile)

`POST /common/email/send-code` is protected by captcha: the body must carry `captcha_token`, and the backend calls Cloudflare siteverify before generating any code.

```jsonc
{
  "email": "author@museflow.ai",
  "scene": "register",
  "captcha_token": "0.xxxxxxxxxxxxxxxxxxxxxxxx..."   // single-use, from the frontend widget
}
```

Behaviour:

| Case | HTTP | Notes |
| :--- | :--- | :--- |
| Missing token | 403 | Frontend must render the widget for the user |
| Invalid / already-used token | 403 | Tokens are **single-use**: re-fetch and reset the widget for every send |
| Action or hostname mismatch | 403 | Token was not minted for this site/scene |
| Verification service down | 503 | Fail-closed, safe to retry |

A failed captcha means no code is generated, no resend cooldown is consumed and nothing is enqueued.
With no secret configured the check is skipped (local dev only; warned at startup).

## Email delivery progress (SSE)

Sending a code is asynchronous: the endpoint enqueues an Asynq task and returns `202` immediately, while the user-service worker delivers the mail concurrently. The frontend can subscribe to progress with the returned `task_id`:

```js
const { task_id } = await sendCode({ email, scene })

const es = new EventSource(`/api/v1/common/tasks/${task_id}/stream`)
es.addEventListener('success', (e) => {
  const data = JSON.parse(e.data) // { task_id, status, message, updated_at }
  showTip(data.message)           // Code sent, check your inbox
  es.close()
})
es.addEventListener('failed', (e) => {
  showError(JSON.parse(e.data).message) // Delivery failed, please retry
  es.close()
})
```

Event fields: `task_id`, `status`, `message` (display-ready), `updated_at`. The server sends a comment heartbeat every 15s to keep intermediaries from closing the connection, and browsers auto-reconnect per the `retry` field (3s).
