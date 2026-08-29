# Architecture Design

## Overall Architecture

MuseFlow's backend is a Go microservices monorepo (gRPC + Gin); the frontend is Vue 3 + TS + Vite.

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

## Module Breakdown

| Path | Type | Responsibility |
|------|------|------|
| `pkg/envloader` | shared lib | Layered `.env` loading; service-prefixed keys + shared unprefixed keys |
| `pkg/errcode` | shared lib | Unified error codes (2xxx success / 4xxx client / 5xxx server), bilingual zh/en |
| `pkg/logger` | shared lib | `slog` + `lumberjack` structured logging |
| `proto/user` | contract | gRPC proto and generated code |
| `services/user-service` | service | All user-domain business logic |
| `services/api-gateway` | service | HTTP boundary, routing, middleware, gRPC client |
| `services/crawl4ai-service` | service | Crawling / content extraction (HTTP + gRPC dual interface) |

## Layering & Dependency Constraints

- **One-directional dependency**: handler → service → repository; no reverse edges.
- user-service is further split: `auth` → `token` + `dto`; `token`/`dto` never depend back on `auth`.
- Config goes through `envloader`; never use `os.Getenv` directly.
- Business errors are defined as `errors.New` in the `auth` package and mapped to gRPC status by handlers; never pass them raw.

## Data Persistence

- PostgreSQL schema is fixed as `user_svc`; the table `user_svc.users` is defined by `services/user-service/database/user_svc.sql`.
- Redis handles: refresh-token allow-list, access-token deny-list, device session list, permission cache.
- Passwords use bcrypt; SSO users may have a nullable password field.

## Key Designs

- **Dual tokens**: access (stateless JWT, 1h) + refresh (JWT + Redis allow-list, 30d, HttpOnly Cookie).
- **2FA**: TOTP two-step login + recovery codes (see [2FA Design](2fa.md)).
- **Authorization**: built-in three roles + permission-code comparison (see [RBAC Design](rbac.md)).
