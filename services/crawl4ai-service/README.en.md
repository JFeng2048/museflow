# crawl4ai-service

A standalone **crawler microservice** wrapping the intelligent web-extraction capability of [Crawl4AI](https://docs.crawl4ai.com/).
**The same extraction/crawl core (`src/crawler.py` + `src/extractor.py`, depending on Crawl4AI's browser rendering)
is exposed through two transports, with dependencies split into two optional groups so each can be packaged and run independently:**

| Variant | Transport | Port (default) | Use case | Entry / definition | Dep group |
| --- | --- | --- | --- | --- | --- |
| **HTTP** | FastAPI / REST + JSON | `5003` | Browser debugging, Swagger (`/docs`), quick external integration | `src/api.py` (`python -m src.api`) | `[http]` |
| **gRPC** | gRPC | `5003` | Go gateway / high-performance internal calls | `src/grpc_server.py` (`python -m src.grpc_server`) | `[grpc]` |

> HTTP and gRPC **share the same port** `PORT` (default `5003`). The two transports cannot listen on that port simultaneously, so at deploy time enable only one variant via its dependency group (install `[http]` to run HTTP, install `[grpc]` to run gRPC).

> The two variants are **fully equivalent**: both HTTP routes and gRPC methods forward to the same `CrawlerService` singleton, with identical behavior, auth, and response shape. The only difference is which dependency group is installed at build time — you can install only `[http]` or only `[grpc]` to shrink the image/deploy footprint:
>
> ```bash
> uv sync --extra http   # HTTP deps only
> uv sync --extra grpc   # gRPC deps only
> uv sync --extra http --extra grpc   # both, single entrypoint runs both
> ```

> **This service only does "crawl + extract".**
> - LLM credentials (`api_key` / `base_url` / `model`) are passed explicitly by the caller on every `Extract` request — **never stored in env / `.env`**; `Extract` always uses the LLM, there is no `use_llm` switch.
> - The shared gRPC contract lives at repo-root `proto/crawl/crawl.proto`; methods: `Health` / `Crawl` / `Extract`.
> - The gateway/caller passes a Bearer token via `Authorization` (HTTP header / gRPC metadata); the `API_KEY` env var toggles auth on/off.

## Feature Overview

- **Two transports, dependency groups**: HTTP (FastAPI, with Swagger UI) and gRPC are split into `[http]` / `[grpc]` optional dependency groups, runnable separately.
- **Shared core logic**: both variants reuse the same `CrawlerService` singleton (`src/service.py`), behavior consistent.
- **3 capabilities**: `Health` probe, `Crawl` single-URL crawl (Markdown), `Extract` LLM structured extraction (each exposed over HTTP and gRPC).
- **Shared Proto contract**: `proto/crawl/crawl.proto` generates stubs for both the Go gateway and the Python service, language-agnostic.
- **Per-request LLM credentials**: `llm.api_key` / `base_url` / `model` go in the `Extract` request, keeping tenants/models isolated.
- **OpenAI-compatible LLM**: any OpenAI-protocol provider (DeepSeek, Together, Qwen, Volcengine Ark, self-hosted gateway…) plugs in directly.
- **Single-URL extraction + retry**: `tenacity` exponential backoff auto-retries transient network errors.
- **Config-driven**: `pydantic-settings` loads `.env`; all runtime params are hot-tunable.
- **uv-managed deps**: `uv.lock` is committed for reproducible CI/deploy.
- **One-shot Docker**: `astral/uv:python3.13-bookworm-slim` multi-stage build; choose `http` / `grpc` / `both` via the `VERSION` build arg; Chromium is preinstalled in the builder stage and copied into runtime, **host Chrome not required**.

## Directory Structure

```
services/crawl4ai-service/
├── main.py              # Unified entrypoint: starts HTTP + gRPC (by ENABLE_* / available deps)
├── test_api.py          # HTTP interface tests
├── pyproject.toml       # Metadata & deps (base + [http] / [grpc] optional groups)
├── uv.lock              # uv lockfile (commit this)
├── README.md
├── .env.example         # Env sample (no LLM creds; includes PORT)
├── scripts/
│   ├── gen_proto.sh     # Generate src/crawl_pb2.py, src/crawl_pb2_grpc.py via grpcio-tools (gRPC variant)
│   ├── run-http.sh      # Run HTTP variant only (after uv sync --extra http)
│   └── run-grpc.sh      # Run gRPC variant only (after uv sync --extra grpc)
│
├── src/                 # Business modules (src layout)
│   ├── config.py        # Config (pydantic-settings; API_KEY / BROWSER_* / dual-interface flags)
│   ├── schema.py        # Pydantic schemas + CrawlerOptions / ExtractSchema / LLMConfig
│   ├── crawler.py       # Service layer: single-URL crawl + retry + browser config
│   ├── extractor.py     # Service layer: LLM extraction strategy (Crawl4AI LLMExtractionStrategy)
│   ├── service.py       # Process-wide CrawlerService singleton + start time (shared by HTTP/gRPC)
│   ├── api.py           # HTTP (FastAPI) transport: build_app() factory + routes
│   └── grpc_server.py   # gRPC transport: CrawlService implementation
│
└── docker/              # Deploy artifacts
    ├── Dockerfile       # Multi-stage (builder + runtime, shared astral/uv base)
    └── docker-compose.yml  # HTTP / gRPC share a port (VERSION decides which actually starts)
```

> Modules live under `src/` to avoid accidentally importing the project root as a package during local tests. Import via `from src.xxx import ...`.

## Capability Map

| Capability | HTTP (FastAPI) | gRPC (proto/crawl) | Auth |
| --- | --- | --- | --- |
| Health | `GET /health` | `Health` | ✅ / ❌ |
| Crawl | `POST /crawl` | `Crawl` | ✅ / ❌ |
| Extract | `POST /extract` | `Extract` | ✅ / ❌ |

> When `API_KEY` is set in `.env`, auth = ✅; otherwise auth is off (local dev only). See [Auth](#auth-bearer-token).

## Quick Start (local, uv)

### 1. Prepare environment

Requires Python 3.13+ and [uv](https://docs.astral.sh/uv/).

```bash
# Install uv (if needed)
curl -LsSf https://astral.sh/uv/install.sh | sh
```

### 2. Install deps & sync lockfile

```bash
cd crawl4ai-service
uv sync --extra dev
```

`uv sync` reads `uv.lock`, creates `.venv`, and installs dependencies.

### 3. Configure environment

```bash
cp .env.example .env
# Edit as needed; API_KEY is optional (empty = auth disabled)
```

### 4. Install browser (first time required)

```bash
# Browser binary only (~150MB):
uv run playwright install chromium
# Or with system deps (recommended on Linux first time):
uv run playwright install --with-deps chromium
```

### 5. Install deps & pick a variant to run

Deps are split into two groups (see `pyproject.toml` `[project.optional-dependencies]`). Install the group first, then start:

```bash
# (A) HTTP variant only
uv sync --extra http
uv run -m src.api            # or bash scripts/run-http.sh

# (B) gRPC variant only (generate stubs first)
bash scripts/gen_proto.sh
uv sync --extra grpc
uv run -m src.grpc_server     # or bash scripts/run-grpc.sh

# (C) Both groups, unified entrypoint (same port → only one actually starts)
uv sync --extra http --extra grpc
uv run -m main
```

The unified entrypoint `main.py` **auto-detects the installed groups** (`http_available()` / `grpc_available()`): HTTP if only `[http]` is installed, gRPC if only `[grpc]` is installed; if both are installed it prefers gRPC (HTTP skipped) due to the port conflict. No env switch needed.

Port: HTTP and gRPC share `PORT` (default `5003`).

### 6. Run tests

```bash
uv run pytest
```

## Docker Deploy

> Base image: `astral/uv:python3.13-bookworm-slim` (shared multi-stage). Browser: Chromium is preinstalled to `/ms-playwright` in the builder stage and reused by runtime — **host Chrome not required**. Start command: `python -m main` (per Dockerfile `CMD`), starting HTTP + gRPC.

### One-liner

```bash
cd services/crawl4ai-service
docker compose -f docker/docker-compose.yml up -d

# In-container gRPC health check (grpcurl etc.):
# grpcurl -plaintext -H "authorization: Bearer ${API_KEY:-}" localhost:5003 crawl.CrawlService/Health

# In-container HTTP health check (Swagger UI at http://localhost:5003/docs):
# curl -H "Authorization: Bearer ${API_KEY:-}" http://localhost:5003/health
```

`docker-compose.yml` essentials:
- `ports: "5003:5003"`: HTTP / gRPC shared port (the `VERSION` build arg decides which dep group is actually installed)
- `environment:`: inlined (no `env_file`); `PORT` sets the listen port
- `shm_size: 2gb`: Chromium / Playwright needs a large `/dev/shm` in containers, else OOM on big pages
- Build stage: `grpc` / `both` run `bash scripts/gen_proto.sh` to generate gRPC stubs; `http` only places a placeholder

### Custom API Key

```bash
# Option 1: export to shell
export API_KEY=my-secret-key
docker compose -f docker/docker-compose.yml up -d

# Option 2: edit docker-compose.yml environment.API_KEY directly
```

### Rebuild image

```bash
# After changing src/ or main.py:
docker build -t jobinsight/crawl4ai-service:0.1.0 -f docker/Dockerfile .
docker compose -f docker/docker-compose.yml up -d
```

> If `apt-get update` is slow (connecting to `deb.debian.org` ~15kB/s), add before [Dockerfile#L28-L29](docker/Dockerfile):
> `RUN sed -i 's|deb.debian.org|mirrors.aliyun.com|g' /etc/apt/sources.list.d/debian.sources` to switch to Aliyun mirror.

## Auth (Bearer Token)

This service uses an `Authorization: Bearer <API_KEY>` header for auth — **not JWT, not OAuth**. The `.env` `API_KEY` is used directly as a long-lived static token; the caller echoes it back in the request header. Production should replace it with a signed, expiring JWT.

> Swagger UI (`/docs`) shows an **Authorize** button: click → paste the `API_KEY` value → Authorize. All subsequent requests in that browser session auto-carry `Authorization: Bearer <your-key>`.

### Enable auth

In `.env`:

```bash
API_KEY=your-strong-secret-key
```

After start, **all 3 endpoints** (`/health`, `/crawl`, `/extract`) require:

```bash
Authorization: Bearer your-strong-secret-key
```

Error codes:

| Scenario | HTTP | `code` | `msg` |
| --- | --- | --- | --- |
| Missing `Authorization` header | 401 | 401 | 缺少 Authorization Bearer 头 |
| `Authorization` not Bearer form | 401 | 401 | 缺少 Authorization Bearer 头 |
| Wrong token | 401 | 401 | API Key 无效 |
| Correct token | 200 | 200 | 成功 |

### Disable auth (local dev only)

Leave `API_KEY` unset (or empty) in `.env`. Startup prints:

```
WARNING  API_KEY 未配置，认证已禁用（仅供本地开发）
```

**Always configure it in production.**

## Call Examples

> Examples assume `API_KEY=crawl4ai-service-api-key` (the placeholder in `.env.example`); omit the `Authorization` header when auth is off locally.

### `/health`

```bash
curl -H "Authorization: Bearer crawl4ai-service-api-key" \
  http://localhost:5003/health
```

```json
{
  "code": 200,
  "msg": "成功",
  "data": {
    "status": "ok",
    "service": "crawl4ai-service",
    "version": "0.1.0",
    "uptime_seconds": 1.234,
    "auth_enabled": true
  }
}
```

### `/crawl` — pure crawl, returns Markdown

**Minimal body** (url only):

```bash
curl -X POST http://localhost:5003/crawl \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer crawl4ai-service-api-key" \
  -d '{"url": "https://example.com"}'
```

**Full body** (with `wait_for` and `options`):

```bash
curl -X POST http://localhost:5003/crawl \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer crawl4ai-service-api-key" \
  -d '{
    "url": "https://example.com",
    "wait_for": "body",
    "options": {
      "timeout": 60,
      "user_agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36",
      "bypass_cache": false,
      "remove_overlay_elements": true,
      "simulate_user": false,
      "magic": false,
      "locale": "zh-CN"
    }
  }'
```

**Success response**:

```json
{
  "code": 200,
  "msg": "成功",
  "data": {
    "success": true,
    "url": "https://example.com",
    "markdown": "# Example Domain\n\n...",
    "status_code": 200,
    "error": null,
    "elapsed_ms": 1234
  }
}
```

### `/extract` — LLM structured extraction

> `llm` is **required**; Pydantic rejects requests without it (422).

**Minimal body**:

```bash
curl -X POST http://localhost:5003/extract \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer crawl4ai-service-api-key" \
  -d '{
    "url": "https://example.com",
    "instruction": "提取页面主标题和一段描述",
    "llm": {"api_key": "sk-xxx", "base_url": "https://api.deepseek.com/v1", "model": "deepseek-chat"}
  }'
```

**Full body** (with `schema_fields` to constrain LLM output):

```bash
curl -X POST http://localhost:5003/extract \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer crawl4ai-service-api-key" \
  -d '{
    "url": "https://jobs.bytedance.com",
    "instruction": "提取所有岗位名称、薪资范围和工作地点",
    "schema_fields": [
      {"name": "title", "description": "岗位名称", "type": "string", "required": true},
      {"name": "salary", "description": "薪资范围", "type": "string", "required": false},
      {"name": "location", "description": "工作地点", "type": "string", "required": false}
    ],
    "llm": {
      "api_key": "sk-xxx", "base_url": "https://api.deepseek.com/v1", "model": "deepseek-chat",
      "temperature": 0.0, "max_tokens": 2048, "request_timeout": 120
    },
    "options": {"timeout": 60, "bypass_cache": true, "remove_overlay_elements": true, "locale": "zh-CN"},
    "extraction_timeout": 120
  }'
```

**Success response**:

```json
{
  "code": 200,
  "msg": "成功",
  "data": {
    "success": true,
    "url": "https://jobs.bytedance.com",
    "markdown": "## 职位列表\n- ...",
    "data": {"title": ["后端工程师", "算法工程师"], "salary": ["25-50K", "30-60K"], "location": ["北京", "上海"]},
    "elapsed_ms": 4567,
    "model": "deepseek-chat"
  }
}
```

> `data.data` is the LLM-extracted business fields (dict keyed by your `schema_fields`); the outer `data` is the wrapper `CrawlData` (`success` / `url` / `markdown` / `elapsed_ms` / `model`).

### CSS / XPath mode?

> `/extract` always uses the LLM; there is no `use_llm=false` / pure-CSS path. If you only want cleaned Markdown, use `/crawl`. For pure CSS/XPath extraction without an LLM, call the main backend's extraction capability (not replicated here to avoid duplication).

### `LLM` field (OpenAI-compatible)

| Field | Required | Default | Description |
| --- | --- | --- | --- |
| `api_key` | ✅ | — | OpenAI-compatible API key |
| `base_url` | ✅ | — | OpenAI-compatible base URL (e.g. `https://.../v1`) |
| `model` | ✅ | — | Model name |
| `temperature` | ❌ | `0.0` | Sampling temperature 0.0–2.0 |
| `max_tokens` | ❌ | `2048` | Max output tokens |
| `request_timeout` | ❌ | `120` | Single LLM request timeout (s) |

## Configuration Reference

All settings come from env vars (or `.env`), but **no LLM credentials**. Full list in [.env.example](.env.example).

| Variable | Default | Description |
| --- | --- | --- |
| `HOST` | `0.0.0.0` | Listen address |
| `PORT` | `5003` | HTTP / gRPC shared listen port |
| `API_KEY` | empty | Empty disables auth; set enables it |
| `SERVICE_NAME` | `crawl4ai-service` | Service id, in `/health` |
| `SERVICE_VERSION` | `0.1.0` | Version, in `/health` |
| `LOG_DIR` | `/app/logs` | Log dir (in container; lazy-created) |
| `LOG_FILE` | `service.log` | Log filename |
| `LOG_LEVEL` | `INFO` | `DEBUG`/`INFO`/`WARNING`/`ERROR` |
| `DEFAULT_TIMEOUT` | `60` | Single-URL timeout (s) |
| `MAX_RETRIES` | `3` | Transient retry count |
| `RETRY_BACKOFF_FACTOR` | `2.0` | Exponential backoff factor |
| `ENABLE_STEALTH` | `true` | Crawl4AI stealth/anti-bot mode |
| `HEADLESS` | `true` | Browser headless |
| `BROWSER_EXECUTABLE_PATH` | empty | Empty → Playwright Chromium; else system Chrome |
| `BROWSER_TYPE` | `chromium` | `chromium` / `chrome` / `msedge` |
| `PLAYWRIGHT_BROWSERS_PATH` | `/ms-playwright` | Playwright browser install location |

> In-container logs go to both **stderr** (`docker logs`) and **`$LOG_DIR/$LOG_FILE`**. The minimal compose mounts no volume, so logs are lost on container removal; add a `volumes` entry to persist.

## Troubleshooting

### 1. `ModuleNotFoundError: No module named 'config'`
`uv sync` not done, or CWD is not the project root:
```bash
cd crawl4ai-service   # the dir containing main.py and src/
uv sync
```

### 2. `playwright._impl._errors.Error: Executable doesn't exist at .../chromium-.../chrome`
Playwright's Chromium not installed:
```bash
uv run playwright install chromium
# or with system deps:
uv run playwright install --with-deps chromium
```

### 3. Playwright does not support the current Linux distro
e.g. `ERROR: Playwright does not support chromium on ubuntu26.04-x64`. Prebuilt Chromium isn't published for that distro yet. **If system `/usr/bin/google-chrome` exists**, set in `.env`:
```bash
BROWSER_EXECUTABLE_PATH=/usr/bin/google-chrome
BROWSER_TYPE=chrome
```
The service auto-uses system Chrome on start, no `playwright install` needed. `src/crawler.py`'s `_build_browser_config` forwards via `chrome_channel=settings.browser_type` to Crawl4AI.

### 4. Docker build stuck at `apt-get update` for minutes
`deb.debian.org` is slow domestically. Only touch the Dockerfile (not daemon.json): insert before `RUN apt-get update`:
```dockerfile
RUN sed -i 's|deb.debian.org|mirrors.aliyun.com|g' /etc/apt/sources.list.d/debian.sources
```

### 5. Docker build stuck pulling `docker/dockerfile:1.7`
The `# syntax=docker/dockerfile:1.7` line makes BuildKit pull a parser image that domestic mirrors don't cache. Workarounds:
- `DOCKER_BUILDKIT=0 docker build -t jobinsight/crawl4ai-service:0.1.0 -f docker/Dockerfile .`
- Or comment out the `# syntax=docker/dockerfile:1.7` line at the top of the Dockerfile.

### 6. Container won't start / browser not found
The Dockerfile already contains `python -m playwright install --with-deps chromium`. If that step fails at build, temporarily mount the host browser:
```yaml
# docker/docker-compose.yml
volumes:
  - /usr/bin/google-chrome:/usr/bin/google-chrome:ro
environment:
  - BROWSER_EXECUTABLE_PATH=/usr/bin/google-chrome
  - BROWSER_TYPE=chrome
```

### 7. `/extract` returns `422` with `llm` in `detail.loc`
The request body is missing the `llm` field. Add:
```json
"llm": {"api_key": "sk-xxx", "base_url": "https://api.deepseek.com/v1", "model": "deepseek-chat"}
```
`llm.api_key` / `base_url` / `model` are all required.

### 8. `/health` (or others) returns `401 code=401 msg="API Key 无效"` or `"缺少 Authorization Bearer 头"`
`API_KEY` is set but the request header is missing/wrong. Check:
1. Header is exactly `Authorization: Bearer <your-key>`.
2. A proxy isn't stripping the `Authorization` header.
3. The token matches the server `.env` `API_KEY` exactly.

### 9. Crawling some sites (e.g. `jobs.bytedance.com`) succeeds once then times out (`ACS-GOTO`) on the second try
ByteDance-like / large sites deploy **Akamai Bot Manager**. Explanation:
- **1st** (20–30s success): passed the ACS JS challenge.
- **2nd** (60s timeout): triggered "too many visits" rule, ACS blocks directly.

**Fixes (by priority):**
1. **Change UA**: use a real Chrome UA, not a self-signed one like `JobInsight/0.1`.
2. **Add interval** between requests to the same site: `sleep 30s+`.
3. **`wait_for` a concrete element**: wait for the ACS challenge to finish (e.g. `css:.list-item` or `networkidle`).
4. **Proxy pool**: pass `http://user:pass@host:port` via `options.proxy`.
5. **Avoid magic mode**: `magic: false` (default).

For bulk crawling of such sites, add a proxy layer + rate-limiting middleware rather than calling this service in a tight loop.

### 10. View in-container logs
```bash
docker logs -f crawl4ai-service
docker exec -it crawl4ai-service tail -f /app/logs/service.log
```
The minimal compose mounts no volume, so logs are lost on removal; add:
```yaml
volumes:
  - ./crawl4ai_logs:/app/logs
```

## Integration with the main backend

Callers (Go gateway / main backend) use the `proto/crawl`-generated client stub to call this service over gRPC:

```python
# Main backend (Python) pseudo-code
import grpc
import crawl_pb2, crawl_pb2_grpc

channel = grpc.insecure_channel("crawl4ai-service:5003")
stub = crawl_pb2_grpc.CrawlServiceStub(channel)
resp = stub.Extract(crawl_pb2.ExtractRequest(
    url=job_url,
    instruction="提取岗位名称、薪资、地点",
    llm=crawl_pb2.LLMConfig(api_key=..., base_url=..., model=...),  # injected per request
))
if resp.success:
    job_info = json.loads(resp.data_json)
else:
    raise BusinessError(resp.error_message)
```

The Go gateway can reuse `github.com/museflow/proto/crawl`'s `crawlpb.NewCrawlServiceClient`. Containers talk over the `museflow-net` bridge; the main backend reaches this service by name `crawl4ai-service:5003`.
