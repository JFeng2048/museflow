# crawl4ai-service 接口说明

数据采集服务（Python，HTTP + gRPC 双接口，共用端口 `:5003`）。封装 Crawl4AI 的浏览器渲染与智能提取能力，供主后端抓取网页/结构化抽取信息。

## 作用

- **网页抓取**：单 URL 抓取并返回 Markdown（`Crawl`）。
- **结构化抽取**：调用 LLM（OpenAI 兼容协议）按指令/字段 schema 抽取页面结构化数据（`Extract`）；LLM 凭证（api_key / base_url / model）由调用方每次请求传入，不落库。
- **双传输**：同一套 `CrawlerService` 核心逻辑，分别经 FastAPI（含 Swagger `/docs`）与 gRPC 暴露，依赖按 `[http]`/`[grpc]` 分组可独立部署。
- **重试与可观测**：单 URL 失败指数退避重试；`/health` 暴露服务状态、版本、运行时长与认证开关。

## 接口

共享契约见 `proto/crawl/crawl.proto`（gRPC），HTTP 路由一一对应：

| 能力 | gRPC | HTTP | 说明 |
| :--- | :--- | :--- | :--- |
| 健康检查 | `Health` | `GET /health` | 服务存活与版本 |
| 抓取 | `Crawl` | `POST /crawl` | 返回页面 Markdown |
| 抽取 | `Extract` | `POST /extract` | LLM 结构化抽取 |

认证：使用 `Authorization: Bearer <API_KEY>` 静态令牌（`API_KEY` 未配置时关闭，仅本地）。详细字段与调用示例见服务内 `README.md`。
