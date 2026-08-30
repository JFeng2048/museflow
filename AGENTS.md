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
- `services/user-service/database/user_svc.sql` — PostgreSQL DDL; creates schema `user_svc` and table `user_svc.users` (schema is fixed, not driven by config).
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

Hot reload: install `air` (`go install github.com/air-verse/air@latest`). Root entry points start every service at once — `dev.bat` (Windows, one window per service) or `./dev.sh` (Linux/macOS, tmux when available): `dev.bat`, `dev.bat gateway|user|worker|web|full|help`. Granular usage: `cd services/user-service && air` (or gateway); the worker uses its own config, `cd services/user-service && air -c .air.worker.toml`.

**Line endings**: `*.bat` files must stay CRLF — cmd.exe truncates lines and garbles output on LF. `*.sh` must stay LF. Check after generating scripts with editors/AI tools that default to LF.

## Coding Style & Naming Conventions

Go: standard `gofmt`/`go vet`; exported PascalCase, files `snake_case.go`. Errors: business errors defined as `errors.New` in the `auth` package and mapped to gRPC status in handlers; never return raw errors across the gRPC boundary. Config reads go through `envloader` (no direct `os.Getenv`). Logging through `pkg/logger`, prefer `logger.WithTraceID`/`logger.Err` field helpers over string formatting.

Frontend: Vue 3 `<script setup>` SFCs, strict TS (`@vue/tsconfig`), 2-space indent.

### Encoded Conventions (团队编码习惯)

These conventions are mandatory for contributions to this repo:

1. **提交信息中英双语**：Commit 主题行（subject）采用 Conventional Commits 规范（`type(scope): 中文主题 / English subject`），类型如 `feat`/`fix`/`refactor`/`docs`/`chore`/`build`/`test`；正文（body）可选，用中文说明「为什么改」。示例：
   ```
   feat(user-service): 拆分 auth/token/dto 子包 / split auth/token/dto subpackages
   ```
   复杂改动按功能原子分拆为多个提交，而非一次性大提交。
2. **配置分层与环境变量**：所有配置统一走 `envloader`，禁止直接使用 `os.Getenv`。共享键无前缀（`JWT_SECRET`、`REDIS_*`、`DB_*`），通过 `envloader.New("REDIS",...)` + `GetCommon` 读取；服务专属键使用前缀（`USER_`、`GATEWAY_`、`LOG_`），通过 `Get` 读取。分层优先级：系统环境变量 > 服务 `.env` > 仓库根 `.env` > 默认值。
3. **注释与文案用中文**：代码注释、文档（如 `*.md`）、日志文案以中文为主；变量名/函数名等标识符仍用英文，保持 `gofmt`/`go vet` 规范。
4. **无循环依赖分层**：`service` 内部按 `auth`/`token`/`dto` 子包拆分，依赖单向、禁止循环：`auth` → `token` + `dto`；`token`/`dto` 不反向依赖 `auth` 或任何内部包。跨层调用同样保持单向（handler → service → repository）。

## Testing Guidelines

Backend: `*_test.go` with the standard `testing` package, placed beside the code in its package (e.g. `internal/service/auth/auth_service_test.go` uses in-memory `repository.UserRepository`/`TokenStore` fakes). `auth` package has the primary coverage; keep tests isolated via fakes. Frontend: Vitest `*.spec.ts` beside source. Always run `go test ./...` and `go vet ./...` for the changed module before committing.

## Commit & Pull Request Guidelines

Commit messages follow Conventional Commits with bilingual subject lines (see 编码习惯 above): `type(scope): 中文主题 / English subject`. Subject lines stay concise (ideally under 50 chars per language); add a Chinese "why" body when the change is non-obvious. Split large changes into focused atomic commits per feature/module.

PRs should link the related issue, describe the change and motivation, and include screenshots for UI changes. The README references a `CONTRIBUTING.md` that does not yet exist; until then, follow these guidelines and keep PRs focused.

## Security & Configuration Tips

Never commit secrets. `.env` files are gitignored; supply configuration through environment variables or `.env.example` copies. Shared secrets use unprefixed keys: `JWT_SECRET` (gateway + user-service sign/verify), `REDIS_*` (token whitelist/blacklist), `DB_*` (PostgreSQL). Per-service config uses prefixes `USER_`, `GATEWAY_`, `LOG_`. Build artifacts (`dist/`, `bin/`, `*.test`, `coverage.*`) are excluded from version control. `go.work` is gitignored by default — keep `go.work.use` listing all modules.
