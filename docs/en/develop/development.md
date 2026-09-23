# Development Guide

## Environment Setup

The repo root uses a Go Workspace (`go.work`), with no root `go.mod`. On Windows there is no `make`,
so run `go` commands directly inside each module directory.

```bash
# Resolve workspace (run after adding modules)
go work sync

# Install tools
go install github.com/swaggo/swag/cmd/swag@latest
go install github.com/air-verse/air@latest   # hot reload (optional)
```

## Local Run

```bash
# user-service (gRPC :5002)
cd services/user-service && go run ./cmd/server

# config-service (gRPC :5004)
cd services/config-service && go run ./cmd/server

# api-gateway (HTTP :5001)
cd services/api-gateway && go run ./cmd/server
```

Copy `.env.example` to `.env` and fill in config. user-service MFA config: see the [2FA Design](2fa.md) configuration section.

Extra requirements for config-service:

- Tables: `psql -f services/config-service/database/config_svc.sql` (a `DROP` + `CREATE` script, for empty databases only).
- Key: `MODEL_SECRET_KEY` must be 32 bytes (generate with `openssl rand -hex 16`); the service fails to start without it.
- Gateway: `GATEWAY_MODEL_SERVICE_URL=localhost:5004`.

Table structure, the key scheme and the frontend wiring for models and system settings are in the model & system configuration design doc (Chinese, `docs/cn/develop/模型与系统配置设计文档.md`).

## Build & Verify

```bash
# Build
cd services/user-service && go build ./cmd/server
cd services/config-service && go build ./cmd/server
cd services/api-gateway && go build ./cmd/server

# Static check
go vet ./...

# Test
go test ./...

# Single test example
go test ./internal/service/auth/ -run TestLoginIssuesUsableTokenPair -v
```

## Code Generation

- **gRPC**: `bash scripts/gen-proto.sh` (needs protoc 29.3 + plugins; falls back to `tools/protogen` when protoc is absent).
- **Swagger**: `cd services/api-gateway && swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal`
  (`--parseDependency --parseInternal` parses cross-package DTOs and the `errcode.Response` generic response).

## Coding Conventions

- Commits: Conventional Commits + bilingual subject, e.g. `feat(user-service): add 2FA / add 2FA`.
- Config: unified via `envloader`; shared keys have no prefix (`JWT_SECRET`/`REDIS_*`/`DB_*`), service keys have a prefix (`USER_`/`GATEWAY_`/`LOG_`).
- Text: comments, docs, and logs are mainly Chinese; identifiers are English, keeping `gofmt`/`go vet`.
- Layering: handler → service → repository one-directional; within service `auth`→`token`+`dto` acyclic.

## Frontend

```bash
cd web && pnpm install && pnpm dev   # dev
cd web && pnpm build                 # must build before commit (vue-tsc + vite)
```
