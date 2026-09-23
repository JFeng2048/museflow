# Service Architecture

**In one sentence**: the backend is a Go microservices monorepo (gRPC + Gin) where api-gateway is the only HTTP boundary, business logic is split by domain into user-service and config-service, and the frontend is a Vue 3 + TS + Vite SPA.

## Overall architecture

```
                ┌─────────────┐
   Browser ───▶ │ api-gateway │ :5001  (Gin HTTP, Swagger, JWT verify, Cookie management)
                └──────┬──────┘
                       │ gRPC
              ┌────────┴────────┐
        ┌─────┴─────┐      ┌─────┴─────┐
        │ user-svc  │      │ config-svc│
        │   :5002   │      │   :5004   │
        └─────┬─────┘      └─────┬─────┘
     auth / user / RBAC / 2FA  model catalog / credentials / settings
              │                     │
     ┌────────┼────────┐     PostgreSQL
     │        │        │     config_svc
 PostgreSQL  Redis   (external SMTP)
 user_svc   token allow/deny list   email codes
```

Browsers talk to api-gateway only, through `/api/v1/**`; services talk to each other over gRPC and are not exposed.
Deployment shape, entry layering and network reachability: see [Deployment Architecture](deployment.md).

## Module breakdown

| Path | Type | Responsibility |
|------|------|----------------|
| `pkg/envloader` | shared lib | Layered `.env` loading; service-prefixed keys + shared unprefixed keys |
| `pkg/errcode` | shared lib | Unified error codes (2xxx success / 4xxx client / 5xxx server), bilingual zh/en |
| `pkg/logger` | shared lib | `slog` + `lumberjack` structured logging |
| `proto/user` | contract | gRPC proto and generated code (user domain) |
| `proto/model` | contract | gRPC proto and generated code (model catalog and system settings) |
| `services/user-service` | service | All user-domain business logic (including the async worker) |
| `services/config-service` | service | Model providers and catalog, user-defined models, general system settings |
| `services/api-gateway` | service | HTTP boundary, routing, middleware, gRPC client |
| `services/crawl4ai-service` | service | Crawling / content extraction (HTTP + gRPC dual interface) |
| `web` | frontend | Vue 3 + TS + Vite SPA; built assets served by Nginx |

## Layering and dependency rules

- **One-directional**: `handler → service → repository`, no reverse edges.
- user-service is split further: `auth` → `token` + `dto`; `token`/`dto` never depend back on `auth`.
- Config goes through `envloader`; never use `os.Getenv` directly.
- Business errors are `errors.New` values in the `auth` package, mapped to gRPC status by handlers; never passed raw.

## Data persistence

- PostgreSQL schemas are fixed per domain: `user_svc` tables come from `services/user-service/database/user_svc.sql`, `config_svc` tables from `services/config-service/database/config_svc.sql`.
- Redis handles: refresh-token allow-list, access-token deny-list, device session list, permission cache.
- Passwords use bcrypt; SSO users may have a nullable password field.
- Model credentials (`api_key` of platform and user providers, secret system-setting values) are stored AES-256-GCM encrypted; only the last 4 characters are ever exposed.
- Table definitions: `services/user-service/database/` and `services/config-service/database/`.

## Key designs

- **Dual tokens**: access (stateless JWT, 1h) + refresh (JWT + Redis allow-list, 30d, HttpOnly cookie) — see [Dual Token Auth](../develop/dual-token-auth.md).
- **2FA**: TOTP two-step login + recovery codes — see [2FA Design](../develop/2fa.md).
- **Authorization**: RBAC reference data (roles, permissions, mappings) is self-healed by startup seeding; `super_admin` bypasses per-code checks via a wildcard — see [RBAC Design](../develop/rbac.md).
- **Human verification**: Cloudflare Turnstile on login/registration; allowed hostnames come from the deployment domain — see [Turnstile Design](../develop/turnstile.md).
- **Async email**: codes are queued and sent by the user-service worker; progress is pushed to the frontend over SSE (`/api/v1/common/tasks/{task_id}/stream`).
- **Model and system settings**: platform providers and models are maintained by admins, users may bring their own providers and models, credentials are write-only and the sole decryption path is upstream catalog discovery. Design doc (Chinese): `docs/cn/develop/模型与系统配置设计文档.md`; API contract (Chinese): `docs/cn/api/config-service.md`.

## Related documents

| Kind | Documents |
|------|-----------|
| Public interfaces | [`../api/api-gateway.md`](../api/api-gateway.md), [`../api/config-service.md`](../api/config-service.md), [`../api/user-service.md`](../api/user-service.md), [`../api/crawl4ai-service.md`](../api/crawl4ai-service.md) |
| Implementation details | [`../develop/`](../develop/) (development guide, module designs) |
| Deployment and entry | [Deployment Architecture](deployment.md) |
