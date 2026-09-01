# user-service

The core user & authentication microservice of MuseFlow, built with **gRPC + Go**, listening on `:5002`. It owns the account lifecycle, dual-token auth, email verification codes, TOTP two-factor auth, RBAC, audit logging, and OAuth linking.

## Tech Stack

| Area | Choice | Notes |
| --- | --- | --- |
| Language | Go 1.26 | Multi-module workspace via `go.work` |
| Transport | gRPC + Protocol Buffers | Contract defined in `proto/user/user.proto` |
| Web framework | — | Pure gRPC server; HTTP is handled by api-gateway |
| ORM | GORM | `gorm.io/driver/postgres`, pool 50/10 |
| Database | PostgreSQL | Fixed schema `user_svc`, DDL in `services/user-service/database/user_svc.sql` |
| Cache | Redis | refresh-token allow/deny lists, email verification codes |
| Async queue | Asynq (Redis-backed) | Offloads slow external deps (email) to a separate worker process |
| Auth | JWT + bcrypt | `token` subpackage signs/verifies; bcrypt for passwords |
| Logging | slog + lumberjack | Unified via `pkg/logger` (JSON + rotation) |
| Error codes | `pkg/errcode` | Unified bilingual (zh/en) codes mapped to gRPC status |

## Architecture

Strict one-directional layering, no cycles: `handler → service → repository`. The `service` layer is further split into `auth/token/dto/rbac/oauth/audit/task/admin`; `token`/`dto` never depend back on `auth` or any internal package. `internal/pkg/*` holds business-agnostic infrastructure (email, queue) shared by the worker and services without depending back on upper layers.

```
cmd/server/main.go          gRPC entrypoint: wiring, health check, graceful shutdown, enqueues async tasks
cmd/worker/main.go          Async worker entrypoint: consumes the asynq queue (email delivery)
internal/
  config/                    Config loading (USER_ prefix + shared DB_/REDIS_/JWT_SECRET/LOG_)
  handler/                   gRPC handlers; proto ↔ service mapping; error-code mapping (incl. WatchTask server stream)
  pkg/
    email/                   Email: SMTP client + embedded templates (HTML/plain text)
    queue/                   Asynq wrapper: task defs, producer client, task status model
    turnstile/               Cloudflare Turnstile captcha (siteverify client)
  service/
    auth/                    register/login/refresh/logout/change-password/2FA/email codes (core)
    token/                   TokenManager, claims, device fingerprint
    dto/                     Device, TokenPair and other cross-layer structs
    task/                    Async task progress lookup & subscription (used by WatchTask)
    rbac/                    Roles & permissions, Redis-cached (TTL = Refresh TTL)
    oauth/                   Third-party account linking
    audit/                   Operation audit persistence
    admin/                   Admin operations (user/role queries)
  worker/handlers/           Asynq task handlers (email delivery, progress reporting)
  repository/                GORM access + Redis stores (TokenStore/VerifyCodeStore/TaskStore)
  model/                     GORM entities (User, etc.); fixed table `user_svc.users`
proto/user/                  Shared contract across services (user.pb.go / user_grpc.pb.go)
database/                    PostgreSQL DDL and migration scripts
Dockerfile                   gRPC service image
Dockerfile.worker            Worker image
```

### Key Designs

- **Dual tokens**: access token (default 1h, returned in body) + refresh token (default 30d, HttpOnly Cookie). Refresh tokens live in a Redis allow-list; logout revokes them and access tokens go to a deny-list.
- **TOTP 2FA**: RFC 6238. After enabling, login step 1 returns an `mfa_ticket`; step 2 exchanges it for tokens via TOTP. One-time recovery codes are provided.
- **Email verification codes**: scene-aware (register/login/reset_password/change_email) with Redis key prefixes + `SetNX` resend cooldown; register marks `email_verified` on success; change email uses the `change_email` scene.
- **Captcha (Cloudflare Turnstile)**: sending a code is a prime abuse target, so the server verifies a single-use token via siteverify before doing anything (`internal/pkg/turnstile`). Verification runs first — on failure no code is generated, no resend cooldown is consumed and nothing is enqueued, so bots cannot exhaust the cooldown and lock out real users. Unset secret degrades to skip (local dev only, warned at startup); verification outages are **fail-closed**.
- **Async email delivery with observable progress**: SMTP is a slow external dependency, so keeping it inside the gRPC request path would stall the API. `SendVerifyCode` now only generates the code and enqueues an Asynq task (returning a `task_id`); a separate worker process consumes the queue concurrently. The worker writes `pending → sending → retrying → success/failed` to Redis and broadcasts it over Pub/Sub; the gateway relays it via the `WatchTask` streaming RPC as SSE.
- **Compensation on enqueue failure**: if enqueueing fails, both the stored code and the resend cooldown lock are rolled back so the user does not wait out a useless cooldown.
- **Passwordless email login**: `LoginWithCode` reuses dual-token issuance and is compatible with 2FA.
- **RBAC cache**: permission lookups are cached in Redis with a TTL equal to the refresh-token lifetime (7 days) to cut DB load.
- **Anti-enumeration**: password/code login returns a unified error for unknown emails identical to wrong credentials.

## Configuration

Loaded via `pkg/envloader` with layered priority: system env > service `.env` > repo-root `.env` > defaults. See `.env.example`.

| Variable | Default | Description |
| --- | --- | --- |
| `USER_PORT` | `5002` | gRPC port |
| `DB_HOST`/`DB_PORT`/`DB_USER`/`DB_PASSWORD`/`DB_NAME` | — | PostgreSQL (shared) |
| `REDIS_ADDR`/`REDIS_PASSWORD`/`REDIS_DB` | `localhost:6379` | Redis (shared) |
| `JWT_SECRET` | — | JWT signing key (shared with gateway, required) |
| `USER_ACCESS_TTL_SECONDS` | `3600` | access token TTL |
| `USER_REFRESH_TTL_SECONDS` | `2592000` | refresh token TTL |
| `USER_BCRYPT_COST` | `10` | bcrypt cost |
| `USER_MFA_*` | — | 2FA issuer/skew/recovery code count & length |
| `USER_CODE_TTL_SECONDS`/`CODE_LENGTH`/`CODE_RESEND_COOLDOWN_SECONDS` | `600`/`6`/`60` | code params |
| `USER_TURNSTILE_SECRET` | — | Turnstile **secret key** (server-side only); unset degrades to skip |
| `USER_TURNSTILE_ENDPOINT` | Cloudflare default | siteverify endpoint |
| `USER_TURNSTILE_TIMEOUT_SECONDS` | `15` | Per-verification timeout in seconds; 15–30 recommended if Cloudflare is slow |
| `USER_TURNSTILE_ALLOWED_HOSTNAMES` | — | Hostname allow-list (comma separated); empty disables the check |
| `USER_SMTP_*` | — | SMTP (logs if unconfigured) |
| `USER_QUEUE_NAME` | `email` | Async queue name |
| `USER_QUEUE_MAX_RETRY` | `3` | Max retries per task |
| `USER_QUEUE_TIMEOUT_SECONDS` | `30` | Per-task execution timeout |
| `USER_QUEUE_STATUS_TTL_SECONDS` | `600` | Task progress retention (client subscription window) |
| `USER_WORKER_CONCURRENCY` | `20` | Worker concurrency, drives email throughput |
| `LOG_*` | — | Log level/format/output (shared) |

## Build & Run

The service and the worker are two separate processes and must both be started (two terminals locally):

```bash
# ---- gRPC service ----
cd services/user-service && go build ./cmd/server
go run ./cmd/server

# ---- Async worker (without it, no email is actually delivered) ----
go build ./cmd/worker
go run ./cmd/worker

# Hot reload (requires air; the worker uses its own config)
air                          # gRPC service
air -c .air.worker.toml      # worker

# On Windows, use the repo-root scripts
scripts\run-user.bat         scripts\run-user-worker.bat
scripts\watch-user.bat       scripts\watch-user-worker.bat

# Test
go test ./...

# Vet
go vet ./...
```

Container images are split as well (build context is the repo root):

```bash
docker build -f services/user-service/Dockerfile -t museflow/user-service .
docker build -f services/user-service/Dockerfile.worker -t museflow/user-service-worker .
```

Database schema is maintained by `services/user-service/database/user_svc.sql` (**not** GORM AutoMigrate); column changes live in `services/user-service/database/migrations/`.

## API Overview (gRPC)

`Register`, `Login`, `RefreshToken`, `Logout`, `ChangePassword`, `UpdateProfile`, 2FA (`EnableMFA`/`VerifyMFA`/`DisableMFA`/`GenerateRecoveryCodes`/`RegenerateRecoveryCodes`), email (`SendVerifyCode`/`WatchTask`/`LoginWithCode`/`ChangeEmail`), sessions (`ListSessions`/`RevokeSession`), admin (`ListUsers`/`AssignRole`), etc. Full contract in `proto/user/user.proto`.

`WatchTask` is the only **server-streaming** RPC: it subscribes to async task progress so the gateway can relay it as SSE.

## Testing

`internal/service/auth` is the primary coverage package, using in-memory fakes for `UserRepository`/`TokenStore`/`VerifyCodeStore`; `oauth` has its own fakes. The email and worker sides are covered with fakes too: `internal/pkg/email` (template rendering and MIME assembly), `internal/worker/handlers` (status transitions and retry policy), `internal/service/task` (progress subscription ordering), `internal/pkg/turnstile` (siteverify mocked with `httptest`, covering action/hostname checks and fail-closed behaviour). Run `go test ./...` and `go vet ./...` before committing.
