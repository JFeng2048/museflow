# user-service

The core user & authentication microservice of MuseFlow, built with **gRPC + Go**, listening on `:5002`. It owns the account lifecycle, dual-token auth, email verification codes, TOTP two-factor auth, RBAC, audit logging, and OAuth linking.

## Tech Stack

| Area | Choice | Notes |
| --- | --- | --- |
| Language | Go 1.26 | Multi-module workspace via `go.work` |
| Transport | gRPC + Protocol Buffers | Contract defined in `proto/user/user.proto` |
| Web framework | — | Pure gRPC server; HTTP is handled by api-gateway |
| ORM | GORM | `gorm.io/driver/postgres`, pool 50/10 |
| Database | PostgreSQL | Fixed schema `user_svc`, DDL in `database/user_svc.sql` |
| Cache | Redis | refresh-token allow/deny lists, email verification codes |
| Auth | JWT + bcrypt | `token` subpackage signs/verifies; bcrypt for passwords |
| Logging | slog + lumberjack | Unified via `pkg/logger` (JSON + rotation) |
| Error codes | `pkg/errcode` | Unified bilingual (zh/en) codes mapped to gRPC status |

## Architecture

Strict one-directional layering, no cycles: `handler → service → repository`. The `service` layer is further split into `auth/token/dto/rbac/oauth/audit/notify/admin`; `token`/`dto` never depend back on `auth` or any internal package.

```
cmd/server/main.go          Entrypoint: wiring, gRPC Server, health check, graceful shutdown
internal/
  config/                    Config loading (USER_ prefix + shared DB_/REDIS_/JWT_SECRET/LOG_)
  handler/                   gRPC handlers; proto ↔ service mapping; error-code mapping
  service/
    auth/                    register/login/refresh/logout/change-password/2FA/email codes (core)
    token/                   TokenManager, claims, device fingerprint
    dto/                     Device, TokenPair and other cross-layer structs
    rbac/                    Roles & permissions, Redis-cached (TTL = Refresh TTL)
    oauth/                   Third-party account linking
    audit/                   Operation audit persistence
    notify/                  Email sending (SMTP, falls back to logging if unconfigured)
    admin/                   Admin operations (user/role queries)
  repository/                GORM access + Redis stores (TokenStore/VerifyCodeStore)
  model/                     GORM entities (User, etc.); fixed table `user_svc.users`
proto/user/                  Shared contract across services (user.pb.go / user_grpc.pb.go)
database/                    PostgreSQL DDL and migration scripts
```

### Key Designs

- **Dual tokens**: access token (default 1h, returned in body) + refresh token (default 30d, HttpOnly Cookie). Refresh tokens live in a Redis allow-list; logout revokes them and access tokens go to a deny-list.
- **TOTP 2FA**: RFC 6238. After enabling, login step 1 returns an `mfa_ticket`; step 2 exchanges it for tokens via TOTP/`VerifyEmailCode`. One-time recovery codes are provided.
- **Email verification codes**: scene-aware (register/verify/login/reset) with Redis key prefixes + `SetNX` resend cooldown; register marks `email_verified` on success.
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
| `USER_SMTP_*` | — | SMTP (logs if unconfigured) |
| `LOG_*` | — | Log level/format/output (shared) |

## Build & Run

```bash
# Build
cd services/user-service && go build ./cmd/server

# Run (requires PostgreSQL/Redis up and repo-root .env configured)
go run ./cmd/server

# Hot reload (requires air)
air

# Test
go test ./...

# Vet
go vet ./...
```

Database schema is maintained by `database/user_svc.sql` (**not** GORM AutoMigrate); column changes live in `database/migrations/`.

## API Overview (gRPC)

`Register`, `Login`, `RefreshToken`, `Logout`, `ChangePassword`, `UpdateProfile`, 2FA (`EnableMFA`/`VerifyMFA`/`DisableMFA`/`GenerateRecoveryCodes`/`RegenerateRecoveryCodes`), email (`SendVerifyCode`/`VerifyEmail`/`LoginWithCode`), sessions (`ListSessions`/`RevokeSession`), admin (`ListUsers`/`AssignRole`), etc. Full contract in `proto/user/user.proto`.

## Testing

`internal/service/auth` is the primary coverage package, using in-memory fakes for `UserRepository`/`TokenStore`/`VerifyCodeStore`; `oauth` has its own fakes. Run `go test ./...` and `go vet ./...` before committing.
