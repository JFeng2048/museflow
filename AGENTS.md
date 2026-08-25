# Repository Guidelines

Contributor guide for MuseFlow, an AI-powered novel generation platform built on Go microservices with a Vue 3 frontend.

## Project Structure & Module Organization

The repository is in early stage. Only the web frontend exists today; the Go backend described in `README.md` is not yet implemented.

- `web/` — Vue 3 + TypeScript + Vite frontend (the only source module currently).
  - `web/src/` — application source (`App.vue`, `main.ts`, `style.css`, plus `assets/` and `components/`).
  - `web/public/` — static assets copied as-is (`favicon.svg`, `icons.svg`).
  - `web/*.tsconfig*.json`, `web/vite.config.ts` — TypeScript and Vite configuration.
- `README.md` / `READMD.cn.md` — English and Chinese project overviews.
- `LICENSE` — MIT license.

When the Go services land, follow standard Go layout (`cmd/`, `internal/`, `pkg/`) per the README architecture (crawling, RAG generation, scheduling, publishing).

## Build, Test, and Development Commands

The frontend uses pnpm (see `web/pnpm-lock.yaml`). From `web/`:

- `pnpm install` — install dependencies.
- `pnpm dev` — start the Vite dev server with hot reload.
- `pnpm build` — type-check with `vue-tsc -b`, then build the production bundle with Vite.
- `pnpm preview` — serve the built bundle locally for verification.

For the Go backend (once added): `go build ./...`, `go run ./cmd/...`, and `go test ./...` from the repo root.

## Coding Style & Naming Conventions

Frontend: Vue 3 `<script setup>` SFCs, TypeScript in strict mode (`@vue/tsconfig`), 2-space indentation. `vue-tsc` enforces types and Vite handles bundling; run `pnpm build` before committing.

Backend (planned): standard Go style via `gofmt`/`go vet`; exported names use Go PascalCase and files use `snake_case.go`.

## Testing Guidelines

No test suite exists yet. Add frontend tests with Vitest (`*.spec.ts` beside source) and backend tests as `*_test.go` using the standard `testing` package. Run `pnpm build` and `go test ./...` to confirm no regressions.

## Commit & Pull Request Guidelines

Commit messages are short, imperative, Chinese-language summaries (e.g., `说明文档初始化`), ideally under 50 characters with a "why" body when needed.

PRs should link the related issue, describe the change and motivation, and include screenshots for UI changes. The README references a `CONTRIBUTING.md` that does not yet exist; until then, follow these guidelines and keep PRs focused.

## Security & Configuration Tips

Never commit secrets. `.env` files are gitignored; supply configuration through environment variables. Build artifacts (`dist/`, `*.test`, `coverage.*`) are already excluded from version control.
