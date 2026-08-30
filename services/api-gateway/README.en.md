# api-gateway

The unified HTTP entry point of MuseFlow, built with **Gin + Go**, listening on `:5001`. It exposes RESTful JSON endpoints and proxies to backend gRPC microservices (currently user-service), owning auth, CORS, access logging, and request tracing.

## Tech Stack

| Area | Choice | Notes |
| --- | --- | --- |
| Language | Go 1.26 | `go.work` multi-module workspace |
| HTTP framework | Gin | Routing, binding/validation, middleware |
| Upstream | gRPC | `google.golang.org/grpc` client to user-service |
| Docs | Swagger (swaggo) | `swag init`; UI at `/swagger/index.html` |
| Logging | `pkg/logger` (slog + lumberjack) | JSON format, rotation |
| Error codes | `pkg/errcode` | Unified bilingual `Response{code,message,data}` |
| Config | `pkg/envloader` | Layered; `GATEWAY_` prefix + shared keys |

## Architecture

```
cmd/server/main.go        Entrypoint: config, logging, user-service gRPC client, HTTP Server, graceful shutdown
internal/
  config/                  Config (GATEWAY_ prefix + JWT_SECRET / LOG_ shared keys)
  client/                  user-service gRPC client wrapper (lazy connect)
  router/                  Route registration & middleware assembly
    v1/                    Domain-split routers (auth / user ...)
  middleware/              CORS, JWT auth, access log, request id
  handler/                 HTTP handlers: dto ↔ proto, gRPC error mapping, Cookie writes
  dto/                     Request/response structs (with Swagger annotations)
proto/user/                Contract shared with user-service
```

### Key Designs

- **Dual-token pass-through**: access token is returned in the body and sent as `Authorization: Bearer`; refresh token is written to an HttpOnly Cookie (controlled by `CookieSecure`/`SameSite`/`Domain`). Login/refresh/logout manage the Cookie at the gateway.
- **Auth middleware**: verifies access-token signature locally (shares `JWT_SECRET` with user-service). Under `/auth/*` only login/register/reset/email-code are public; everything else requires a valid Bearer.
- **2FA compatibility**: when login returns an `mfa_ticket`, the gateway issues no tokens and only echoes the ticket; the client exchanges it via `/auth/mfa/verify-login`.
- **Async progress over SSE**: sending a code is handled asynchronously by user-service; the gateway converts the `WatchTask` server stream into SSE for the client. It closes the connection after a terminal event (`success`/`failed`) and emits periodic heartbeats so intermediaries don't drop it. SSE requires response buffering to be off (`X-Accel-Buffering: no`); Nginx must also disable `proxy_buffering` in deployment.
- **Unified error mapping**: gRPC status is converted via `errcode` into HTTP status + bilingual message (`writeGRPCError`).
- **Request tracing**: every request gets a request-id injected into the log context for cross-service tracing.

## API Overview (HTTP, prefix `/api/v1`)

| Method | Path | Description | Auth |
| --- | --- | --- | --- |
| POST | `/auth/register` | Register (requires email code) | No |
| POST | `/auth/login` | Password login (may trigger 2FA) | No |
| POST | `/auth/logout` | Logout and revoke tokens | No |
| POST | `/auth/mfa/*` | 2FA enable/verify/recovery | Yes |
| POST | `/auth/password/reset` | Reset password with code (code via `/common/email/send-code` scene=reset_password) | No |
| POST | `/auth/login/code` | Passwordless email-code login | No |
| GET | `/auth/sessions` | List sessions | Yes |
| DELETE | `/auth/sessions/:id` | Revoke a session | Yes |
| POST | `/common/email/send-code` | Send email code (register/login/reset_password/change_email); async, returns `202` + `task_id` | No |
| GET | `/common/tasks/{task_id}/stream` | SSE stream of email delivery progress (closed after terminal event) | No |
| POST | `/common/refresh` | Refresh access via refresh Cookie | No |
| POST | `/user/email/change` | Change email (login required, get change_email code first) | Yes |
| GET | `/user/profile` | Current user profile | Yes |

Full fields and examples are in Swagger (`/swagger/index.html`).

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `GATEWAY_PORT` | `5001` | HTTP port |
| `GATEWAY_USER_SERVICE_URL` | `localhost:5002` | user-service gRPC address |
| `JWT_SECRET` | — | Shared signing/verify key with user-service (required) |
| `GATEWAY_ALLOW_ORIGINS` | `http://localhost:5173` | CORS origins (comma-separated) |
| `GATEWAY_COOKIE_SECURE` | `false` | Cookie HTTPS-only (set true in prod) |
| `GATEWAY_COOKIE_SAMESITE` | `lax` | SameSite policy |
| `GATEWAY_COOKIE_DOMAIN` | — | Cookie domain |
| `LOG_*` | — | Logging (shared) |

## Build & Run

```bash
# Build
cd services/api-gateway && go build ./cmd/server

# Run
go run ./cmd/server

# Hot reload (air)
air

# Regenerate Swagger (requires swag)
swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

# Vet
go vet ./...
```

## Testing & Quality

The gateway has no standalone unit-test dir; business logic is covered by user-service tests. The gateway focuses on contract and routing. Run `go vet ./...` and validate Swagger annotations before committing.
