# Repository Guidelines

Contributor guide for MuseFlow, an AI-powered novel generation platform. The backend is a Go microservices monorepo (gRPC + Gin), and the frontend is Vue 3 + TypeScript + Vite.

## Project Structure & Module Organization

Monorepo with a Go Workspace (`go.work`) at the repo root. Each service and each shared package has its own `go.mod` (referenced via `replace` directives in `go.work`). No root `go.mod`.

- `pkg/` — shared Go libraries, each an independent module:
  - `envloader/` — layered `.env` loading (system env > service `.env` > repo-root `.env` > defaults). Service-prefixed keys via `Get`, shared keys via `GetCommon` (e.g. `JWT_SECRET`, `REDIS_*`, `DB_*`).
  - `errcode/` — unified error codes (2000+) with bilingual (zh/en) messages driven by `Accept-Language`; `SuccessGin`/`ErrorGin` for HTTP, gRPC status mapping in handlers.
  - `logger/` — `slog` + `lumberjack` logger with `Config` (level/format/output/path/rotation), `Init`, context and field helpers (`logger.Err`, `logger.UserUUID`, `logger.WithTraceID`).
- `proto/user/` — shared gRPC contract (`user.proto` + generated `user.pb.go` / `user_grpc.pb.go`), independent module.
- `services/user-service/` — gRPC user service (`:5002`):
  - `internal/config` — loads `USER_` (service), `DB_*`/`REDIS_*`/`JWT_SECRET` (shared via `New("DB",...)`/`New("REDIS",...)`), `LOG_*` (via `GetCommon`).
  - `internal/handler` — gRPC handlers; convert proto ↔ service layer; map `auth.Err*` to gRPC status codes.
  - `internal/service` — business logic, split into subpackages: `auth` (`AuthService`), `token` (`TokenManager`/`claims`/`fingerprint`), `dto` (`Device`,`TokenPair`). No import cycle: `auth` → `token`+`dto`; `token`/`dto` depend on nothing internal.
  - `internal/repository` — GORM data access + Redis token store (`TokenStore`); `model.User` uses fixed table `user_svc.users`.
  - `internal/model` — GORM entity `User` (password nullable for SSO users).
- `services/api-gateway/` — HTTP gateway (`:5001`, Gin):
  - `internal/config` (`GATEWAY_` + shared), `router`, `middleware` (CORS/auth/access-log/request-id), `handler` (dto ↔ proto), `client` (user-service gRPC client), `dto` (HTTP request/response structs for Swagger).
- `database/user_svc.sql` — PostgreSQL DDL; creates schema `user_svc` and table `user_svc.users` (schema is fixed, not driven by config).
- `web/` — Vue 3 + TS + Vite frontend.
- `docs/cn/develop/双令牌认证系统设计文档.md` — dual-token auth design reference.
- `.env` (gitignored) + `.env.example` (committed) — global config; each service dir has its own `.env.example` for overrides.

## Build, Test, and Development Commands

Go uses a workspace; there is no root `go.mod`. From the repo root (Windows; `make` is not available, use `go` directly or Go module dirs):

- `go work sync` — (re)resolve the workspace modules. Run after adding modules.
- Build a service: `cd services/user-service && go build ./cmd/server` (or `api-gateway`). Use root `Makefile` `make build` on non-Windows.
- Run a service: `cd services/user-service && go run ./cmd/server` (starts gRPC on `:5002`). Gateway: `cd services/api-gateway && go run ./cmd/server`.
- Vet all: `go vet ./...` (run from a module dir so the workspace resolves, e.g. `cd services/user-service`).
- Test all: `go test ./...` inside a service dir. Single test: `go test ./internal/service/auth/ -run TestLoginIssuesUsableTokenPair -v`.
- Regenerate gRPC: `bash scripts/gen-proto.sh` (needs `protoc` 29.3 + `protoc-gen-go` + `protoc-gen-go-grpc`).
- Swagger (gateway): `cd services/api-gateway && swag init -g cmd/server/main.go -o docs`.
- Frontend (`web/`): `pnpm install`, `pnpm dev`, `pnpm build` (vue-tsc + vite), `pnpm preview`. Run `pnpm build` before commit.

Hot reload: install `air` (`go install github.com/air-verse/air@latest`), then `cd services/user-service && air` (or gateway).

## Coding Style & Naming Conventions

Go: standard `gofmt`/`go vet`; exported PascalCase, files `snake_case.go`. Errors: business errors defined as `errors.New` in the `auth` package and mapped to gRPC status in handlers; never return raw errors across the gRPC boundary. Config reads go through `envloader` (no direct `os.Getenv`). Logging through `pkg/logger`, prefer `logger.WithTraceID`/`logger.Err` field helpers over string formatting.

Frontend: Vue 3 `<script setup>` SFCs, strict TS (`@vue/tsconfig`), 2-space indent.

## Testing Guidelines

Backend: `*_test.go` with the standard `testing` package, placed beside the code in its package (e.g. `internal/service/auth/auth_service_test.go` uses in-memory `repository.UserRepository`/`TokenStore` fakes). `auth` package has the primary coverage; keep tests isolated via fakes. Frontend: Vitest `*.spec.ts` beside source. Always run `go test ./...` and `go vet ./...` for the changed module before committing.

## Commit & Pull Request Guidelines

Commit messages are short, imperative, Chinese-language summaries (e.g. `说明文档初始化`), ideally under 50 characters with a "why" body when needed.

PRs should link the related issue, describe the change and motivation, and include screenshots for UI changes. The README references a `CONTRIBUTING.md` that does not yet exist; until then, follow these guidelines and keep PRs focused.

## Security & Configuration Tips

Never commit secrets. `.env` files are gitignored; supply configuration through environment variables or `.env.example` copies. Shared secrets use unprefixed keys: `JWT_SECRET` (gateway + user-service sign/verify), `REDIS_*` (token whitelist/blacklist), `DB_*` (PostgreSQL). Per-service config uses prefixes `USER_`, `GATEWAY_`, `LOG_`. Build artifacts (`dist/`, `bin/`, `*.test`, `coverage.*`) are excluded from version control. `go.work` is gitignored by default — keep `go.work.use` listing all modules.
