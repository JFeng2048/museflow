# crawl4ai-service API

The data-crawling service (Python, HTTP + gRPC dual interface, shared port `:5003`). Wraps Crawl4AI's browser rendering and intelligent extraction, for the main backend to crawl pages / extract structured data.

## Purpose

- **Web crawl**: single-URL crawl returning Markdown (`Crawl`).
- **Structured extraction**: call an LLM (OpenAI-compatible protocol) to extract structured data by instruction/field schema (`Extract`); LLM credentials (api_key / base_url / model) are passed per request by the caller, never stored.
- **Dual transport**: the same `CrawlerService` core is exposed via both FastAPI (with Swagger `/docs`) and gRPC; deps are split into `[http]`/`[grpc]` groups for independent deployment.
- **Retry & observability**: single-URL failures retry with exponential backoff; `/health` exposes service status, version, uptime, and auth flag.

## Interface

Shared contract in `proto/crawl/crawl.proto` (gRPC); HTTP routes map one-to-one:

| Capability | gRPC | HTTP | Description |
| :--- | :--- | :--- | :--- |
| Health | `Health` | `GET /health` | Service liveness & version |
| Crawl | `Crawl` | `POST /crawl` | Returns page Markdown |
| Extract | `Extract` | `POST /extract` | LLM structured extraction |

Auth: static `Authorization: Bearer <API_KEY>` token (`API_KEY` unset disables auth, local only). Detailed fields and examples in the service `README.md`.
