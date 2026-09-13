# Service Architecture

**In one sentence**: the backend is a Go microservices monorepo (gRPC + Gin) where api-gateway is the only HTTP boundary, user-service owns the business logic, and the frontend is a Vue 3 + TS + Vite SPA.

## Overall architecture

```
                ┌─────────────┐
   Browser ───▶ │ api-gateway │ :5001  (Gin HTTP, Swagger, JWT verify, Cookie management)
                └──────┬──────┘
                       │ gRPC
                ┌──────┴──────┐
                │ user-service│ :5002  (auth / user / RBAC / 2FA / audit)
                └──────┬──────┘
        ┌──────────────┼──────────────┐
     PostgreSQL    Redis          (external SMTP)
    user_svc    token allow/deny list   email codes
```

Browsers talk to api-gateway only, through `/api/v1/**`; services talk to each other over gRPC and are not exposed.
Deployment shape, entry layering and network reachability: see [Deployment Architecture](deployment.md).

## Module breakdown

| Path | Type | Responsibility |
|------|------|----------------|
| `pkg/envloader` | shared lib | Layered `.env` loading; service-prefixed keys + shared unprefixed keys |
| `pkg/errcode` | shared lib | Unified error codes (2xxx success / 4xxx client / 5xxx server), bilingual zh/en |
| `pkg/logger` | shared lib | `slog` + `lumberjack` structured logging |
| `proto/user` | contract | gRPC proto and generated code |
| `services/user-service` | service | All user-domain business logic (including the async worker) |
| `services/api-gateway` | service | HTTP boundary, routing, middleware, gRPC client |
| `services/crawl4ai-service` | service | Crawling / content extraction (HTTP + gRPC dual interface) |
| `web` | frontend | Vue 3 + TS + Vite SPA; built assets served by Nginx |

## Layering and dependency rules

- **One-directional**: `handler → service → repository`, no reverse edges.
- user-service is split further: `auth` → `token` + `dto`; `token`/`dto` never depend back on `auth`.
- Config goes through `envloader`; never use `os.Getenv` directly.
- Business errors are `errors.New` values in the `auth` package, mapped to gRPC status by handlers; never passed raw.

## Data persistence

- PostgreSQL schema is fixed as `user_svc`; table `user_svc.users` is defined by `services/user-service/database/user_svc.sql`.
- Redis handles: refresh-token allow-list, access-token deny-list, device session list, permission cache.
- Passwords use bcrypt; SSO users may have a nullable password field.
- Table definitions: `services/user-service/database/`.

## Key designs

- **Dual tokens**: access (stateless JWT, 1h) + refresh (JWT + Redis allow-list, 30d, HttpOnly cookie) — see [Dual Token Auth](../develop/dual-token-auth.md).
- **2FA**: TOTP two-step login + recovery codes — see [2FA Design](../develop/2fa.md).
- **Authorization**: RBAC reference data (roles, permissions, mappings) is self-healed by startup seeding; `super_admin` bypasses per-code checks via a wildcard — see [RBAC Design](../develop/rbac.md).
- **Human verification**: Cloudflare Turnstile on login/registration; allowed hostnames come from the deployment domain — see [Turnstile Design](../develop/turnstile.md).
- **Async email**: codes are queued and sent by the user-service worker; progress is pushed to the frontend over SSE (`/api/v1/common/tasks/{task_id}/stream`).

## Related documents

| Kind | Documents |
|------|-----------|
| Public interfaces | [`../api/api-gateway.md`](../api/api-gateway.md), [`../api/user-service.md`](../api/user-service.md), [`../api/crawl4ai-service.md`](../api/crawl4ai-service.md) |
| Implementation details | [`../develop/`](../develop/) (development guide, module designs) |
| Deployment and entry | [Deployment Architecture](deployment.md) |
