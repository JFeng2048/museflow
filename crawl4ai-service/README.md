# crawl4ai-service

一个独立的 FastAPI 微服务，封装 [Crawl4AI](https://docs.crawl4ai.com/) 的智能网页提取能力。
主后端（`backed/`）通过 HTTP 调用本服务完成需要的网页抓取 / 结构化提取任务。

> **本服务只做"爬取 + 提取"两件事。**
> - LLM 凭证（`api_key` / `base_url` / `model`）由调用方在每次 `/extract` 请求的 `llm` 字段中显式传入，**不在环境变量 / `.env` 中保存**；`/extract` 始终走 LLM，没有 `use_llm` 开关。
> - 通过 `Authorization: Bearer <API_KEY>` 头做认证，`API_KEY` 环境变量控制开关；Swagger UI 顶部有 **Authorize** 按钮，填一次作用于全部接口。

## 特性一览

- **3 个 HTTP 接口**：`/health` 健康探针、`/crawl` 单 URL 爬取（Markdown）、`/extract` LLM 智能结构化提取。
- **统一响应包装**：所有响应都是 `{ code, msg, data }` 形状（与主后端 `backed/schemas/response.py` 一致）。
- **LLM 凭证按请求传入**：`llm.api_key` / `base_url` / `model` 等放进 `POST /extract` 请求体，跨租户 / 跨模型互不污染。
- **OpenAI 兼容 LLM**：任意 OpenAI 协议服务（DeepSeek、Together、Qwen、火山方舟、自建网关……）都可以直接对接。
- **单 URL 提取 + 重试**：`tenacity` 指数退避，自动重试瞬时网络错误。
- **Bearer Token 认证**：`Authorization: Bearer <API_KEY>` 头，`API_KEY` 环境变量控制（留空关闭，仅供本地开发）；Swagger UI 顶部 **Authorize** 按钮一次性填好。
- **配置驱动**：`pydantic-settings` 加载 `.env`，所有运行参数可热改。
- **uv 管理依赖**：锁文件 `uv.lock` 入库，CI/部署可复现。
- **Docker 一键部署**：`astral/uv:python3.13-bookworm-slim` 多阶段构建，`docker compose up` 即起；Chromium 在 builder 阶段预装并拷贝进 runtime 镜像，**宿主机不需要 Chrome**。

## 目录结构

```
crawl4ai-service/
├── main.py              # FastAPI 入口（lifespan + 挂载 APIRouter + 异常处理 + 日志初始化）
├── test_api.py          # API + 认证 + 响应包装 + LLM 兼容性的测试
├── pyproject.toml       # 项目元数据与依赖
├── uv.lock              # uv 锁文件（应入库）
├── README.md
├── .env.example         # 环境变量样例（不含 LLM 凭证）
│
├── src/                 # 业务模块（采用 src layout）
│   ├── config.py        # 配置管理（pydantic-settings；含 API_KEY / LOG_* / BROWSER_*）
│   ├── schema.py        # Pydantic schema + APIResponse[T] 包装 + LLMConfig
│   ├── crawler.py       # 服务层：单 URL 异步爬取 + 重试 + 浏览器配置
│   ├── extractor.py     # 服务层：LLM 智能提取策略（基于 Crawl4AI LLMExtractionStrategy）
│   └── api.py           # APIRouter：3 个接口 + verify_api_key 依赖 + 异常处理
│
└── docker/              # 部署相关文件
    ├── Dockerfile       # 多阶段构建（builder + runtime，共用 astral/uv base）
    └── docker-compose.yml  # 极简 compose：固定端口 + 内联环境变量
```

> 业务模块放在 `src/` 下是为了避免本地测试时把项目根目录误当包导入。
> 导入时统一使用 `from src.xxx import ...` 前缀。

## 接口

| 方法 | 路径       | 摘要                              | 鉴权      |
| ---- | ---------- | --------------------------------- | --------- |
| GET  | `/health`  | 健康检查（返回 `auth_enabled`）   | ✅ / ❌   |
| POST | `/crawl`   | 爬取单 URL → Markdown             | ✅ / ❌   |
| POST | `/extract` | 从单 URL 用 LLM 智能提取结构化数据 | ✅ / ❌   |

> 当 `.env` 中设置了 `API_KEY` 时，鉴权 = ✅；未设置则鉴权关闭（仅供本地开发）。
> 详见 [认证](#认证bearer-token)。

OpenAPI 文档自动生成于 `/docs` 与 `/redoc`（`/docs` 顶部有 **Authorize** 按钮）。

## 响应格式

所有接口（包括错误）都返回统一的 `APIResponse` 包装：

```json
{
  "code": 200,
  "msg": "成功",
  "data": { ... }
}
```

`code` 含义：

| 取值 | 含义                                          |
| ---- | --------------------------------------------- |
| 200  | 成功                                          |
| 400  | 请求参数错误（extract 构造失败等）            |
| 401  | 缺少 / 错误的 `Authorization: Bearer <token>` |
| 422  | 请求体不合法（Pydantic 校验失败）             |
| 500  | 服务器内部错误                                |
| 503  | 爬虫未初始化（lifespan 失败）                 |

## 快速开始（本地 uv 运行）

### 1. 准备环境

需要 Python 3.13+ 与 [uv](https://docs.astral.sh/uv/)。

```bash
# 安装 uv（如尚未安装）
curl -LsSf https://astral.sh/uv/install.sh | sh
```

### 2. 安装依赖与同步锁文件

```bash
cd crawl4ai-service
uv sync --extra dev
```

`uv sync` 会读取 `uv.lock` 创建 `.venv` 并安装所有依赖。

### 3. 配置环境变量

```bash
cp .env.example .env
# 按需修改；其中 API_KEY 为可选（留空即关闭认证）
```

### 4. 安装浏览器（首次必须）

```bash
# 仅装浏览器二进制（约 150MB）：
uv run playwright install chromium
# 或者连同系统依赖一起装（Linux 首次推荐）：
uv run playwright install --with-deps chromium
```

> 如果你看到 `Playwright does not support chromium on <发行版>-x64` 的报错，
> 说明当前 Playwright 版本尚未为该 Linux 发行版发布预编译 Chromium。
> 解决办法见 [故障排查](#3-playwright-不支持当前-linux-发行版)。

### 5. 启动服务

```bash
uv run main.py
```

打开 `http://localhost:11235/docs` 即可看到 OpenAPI 文档并直接试调。
`main.py` 的 `__main__` 块内部调用 `uvicorn.run("main:app", host=settings.host, port=settings.port)`，
`host` / `port` 由 `HOST` / `PORT` 环境变量覆盖（默认 `0.0.0.0:11235`）。

### 6. 运行测试

```bash
uv run pytest
```

## Docker 部署

> 镜像基座：`astral/uv:python3.13-bookworm-slim`（多阶段共用）
> 浏览器：Chromium 在 builder 阶段预装到 `/ms-playwright`，runtime 阶段直接复用，**宿主机不需要 Chrome**。
> 启动命令：`uv run main.py`（由 Dockerfile 内的 `CMD` 指定）

### 一行启动

```bash
cd crawl4ai-service
docker compose -f docker/docker-compose.yml up -d

# 看健康
sleep 30 && curl -fsS http://localhost:11235/health | python3 -m json.tool
```

`docker-compose.yml` 的极简结构：

- `ports: "11235:11235"`（端口固定，不再走 env var 间接映射）
- `environment:` 内联注入（**不再用 env_file**）；shell 里 `export` 即可
- `shm_size: 2gb`（Chromium / Playwright 在容器里需要大块 `/dev/shm`，否则多 tab / 大页面会 OOM）
- `healthcheck` 携带 `Authorization: Bearer ${API_KEY:-}`（`/health` 走鉴权，healthcheck 也得带头）

### 自定义 API Key

```bash
# 方式 1：export 到 shell
export API_KEY=my-secret-key
docker compose -f docker/docker-compose.yml up -d

# 方式 2：直接修改 docker-compose.yml 中 environment.API_KEY
```

### 重新构建镜像

```bash
# 改完 src/ 或 main.py 后：
docker build \
  -t jobinsight/crawl4ai-service:0.1.0 \
  -f docker/Dockerfile \
  .

# 重启容器
docker compose -f docker/docker-compose.yml up -d
```

> 构建时若 `apt-get update` 太慢（连 `deb.debian.org` ~15kB/s），把 [Dockerfile#L28-L29](file:///home/jfeng/code/python/project/JobInsight/crawl4ai-service/docker/Dockerfile#L28-L29) 之前
> 加一行 `RUN sed -i 's|deb.debian.org|mirrors.aliyun.com|g' /etc/apt/sources.list.d/debian.sources` 切阿里源。

## 认证（Bearer Token）

本服务使用 `Authorization: Bearer <API_KEY>` 头做认证，**不是 JWT，更不是 OAuth**。
直接把 `.env` 里的 `API_KEY` 当作 long-lived static token 用，调用方在请求头里
原样回填；生产环境建议替换为带签名 + 过期的 JWT。

> Swagger UI（`/docs`）顶部会渲染一个 **Authorize** 按钮：
> 点击 → 弹窗里填入 `API_KEY` 的值 → 点 Authorize。
> 之后该浏览器会话内的所有请求都会自动带上 `Authorization: Bearer <your-key>`。

### 开启认证

在 `.env` 中设置：

```bash
API_KEY=your-strong-secret-key
```

启动服务后，**所有 3 个接口**（`/health`、`/crawl`、`/extract`）都会强制要求请求头携带：

```bash
Authorization: Bearer your-strong-secret-key
```

错误码：

| 场景                            | HTTP 状态码 | `code` | `msg`                              |
| ------------------------------- | ----------- | ------ | ---------------------------------- |
| 缺少 `Authorization` 头         | 401         | 401    | 缺少 Authorization Bearer 头       |
| `Authorization` 不是 Bearer 形式 | 401         | 401    | 缺少 Authorization Bearer 头       |
| token 错误                      | 401         | 401    | API Key 无效                       |
| token 正确                      | 200         | 200    | 成功                               |

### 关闭认证（仅限本地开发）

`.env` 中不设置 `API_KEY`（或留空）即可。启动时会打印：

```
WARNING  API_KEY 未配置，认证已禁用（仅供本地开发）
```

**生产环境务必配置。**

## 调用示例

> 以下示例默认 `API_KEY=crawl4ai-service-api-key`（即 `.env.example` 里的占位值）；
> 本地未配置认证时省略 `Authorization` 头即可。

### `/health`

```bash
curl -H "Authorization: Bearer crawl4ai-service-api-key" \
  http://localhost:11235/health
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

### `/crawl` —— 纯爬取，返回 Markdown

**最小 body**（只传 `url`）：

```bash
curl -X POST http://localhost:11235/crawl \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer crawl4ai-service-api-key" \
  -d '{
    "url": "https://example.com"
  }'
```

**完整 body**（带 `wait_for` 和 `options`）：

```bash
curl -X POST http://localhost:11235/crawl \
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

**成功响应**：

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

### `/extract` —— LLM 智能提取

> `llm` 字段是**必填**的，Pydantic 会以 422 拦截缺 `llm` 的请求。

**最小 body**：

```bash
curl -X POST http://localhost:11235/extract \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer crawl4ai-service-api-key" \
  -d '{
    "url": "https://example.com",
    "instruction": "提取页面主标题和一段描述",
    "llm": {
      "api_key": "sk-xxx",
      "base_url": "https://api.deepseek.com/v1",
      "model": "deepseek-chat"
    }
  }'
```

**完整 body**（带 `schema_fields` 约束 LLM 输出结构）：

```bash
curl -X POST http://localhost:11235/extract \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer crawl4ai-service-api-key" \
  -d '{
    "url": "https://jobs.bytedance.com",
    "instruction": "提取所有岗位名称、薪资范围和工作地点",
    "schema_fields": [
      {"name": "title",    "description": "岗位名称", "type": "string", "required": true},
      {"name": "salary",   "description": "薪资范围", "type": "string", "required": false},
      {"name": "location", "description": "工作地点", "type": "string", "required": false}
    ],
    "llm": {
      "api_key": "sk-xxx",
      "base_url": "https://api.deepseek.com/v1",
      "model": "deepseek-chat",
      "temperature": 0.0,
      "max_tokens": 2048,
      "request_timeout": 120
    },
    "options": {
      "timeout": 60,
      "bypass_cache": true,
      "remove_overlay_elements": true,
      "locale": "zh-CN"
    },
    "extraction_timeout": 120
  }'
```

**成功响应**：

```json
{
  "code": 200,
  "msg": "成功",
  "data": {
    "success": true,
    "url": "https://jobs.bytedance.com",
    "markdown": "## 职位列表\n- ...",
    "data": {
      "title": ["后端工程师", "算法工程师"],
      "salary": ["25-50K", "30-60K"],
      "location": ["北京", "上海"]
    },
    "elapsed_ms": 4567,
    "model": "deepseek-chat"
  }
}
```

> `data.data` 是 LLM 提取的**业务字段**（按你 `schema_fields` 定义的 key 返回 dict）；
> 外层 `data` 是包装层的 `CrawlData`（`success` / `url` / `markdown` / `elapsed_ms` / `model` 等元信息）。

### CSS / XPath 模式？

> 本服务 `/extract` 始终走 LLM，不再提供 `use_llm=false` / 纯 CSS 路径。
> 如果你只是想要清洗后的 Markdown，请改用 `/crawl`；
> 如果你想要纯 CSS / XPath 提取且不调 LLM，建议直接调用主后端已有的提取能力
> （本服务不复刻，避免功能重复）。

### `LLM` 字段（OpenAI 兼容）

| 字段              | 必填 | 默认   | 说明                                   |
| ----------------- | ---- | ------ | -------------------------------------- |
| `api_key`         | ✅   | —      | OpenAI 兼容 API Key                    |
| `base_url`        | ✅   | —      | OpenAI 兼容 base URL（如 `https://.../v1`） |
| `model`           | ✅   | —      | 使用的模型名                           |
| `temperature`     | ❌   | `0.0`  | 采样温度，范围 0.0-2.0                 |
| `max_tokens`      | ❌   | `2048` | 最大输出 token                         |
| `request_timeout` | ❌   | `120`  | 单次 LLM 请求超时（秒）                |

## 配置参考

所有设置来自环境变量（也可写入 `.env`），但**不包含任何 LLM 凭证**。
完整列表见 [.env.example](file:///home/jfeng/code/python/project/JobInsight/crawl4ai-service/.env.example)。

| 变量                       | 默认值              | 说明                                          |
| -------------------------- | ------------------- | --------------------------------------------- |
| `HOST` / `PORT`            | `0.0.0.0` / `11235` | 监听地址 / 端口                               |
| `API_KEY`                  | 空                  | 留空禁用认证；设置后启用                       |
| `SERVICE_NAME`             | `crawl4ai-service`  | 服务标识，写入 `/health` 响应                 |
| `SERVICE_VERSION`          | `0.1.0`             | 版本号，写入 `/health` 响应                   |
| `LOG_DIR`                  | `/app/logs`         | 日志文件目录（容器内；按需懒创建）            |
| `LOG_FILE`                 | `service.log`       | 日志文件名                                    |
| `LOG_LEVEL`                | `INFO`              | `DEBUG` / `INFO` / `WARNING` / `ERROR`        |
| `DEFAULT_TIMEOUT`          | `60`                | 单 URL 超时（秒）                             |
| `MAX_RETRIES`              | `3`                 | 瞬时错误重试次数                              |
| `RETRY_BACKOFF_FACTOR`     | `2.0`               | 指数退避因子                                  |
| `ENABLE_STEALTH`           | `true`              | 启用 Crawl4AI 的反爬隐身模式                  |
| `HEADLESS`                 | `true`              | 浏览器无头运行                                |
| `BROWSER_EXECUTABLE_PATH`  | 空                  | 留空用 Playwright Chromium，否则填系统 Chrome |
| `BROWSER_TYPE`             | `chromium`          | 与上面配合：`chromium` / `chrome` / `msedge`  |
| `PLAYWRIGHT_BROWSERS_PATH` | `/ms-playwright`    | Playwright 浏览器安装位置                     |

> 容器内日志同时输出到 **stderr**（`docker logs` 能看到）和 **`$LOG_DIR/$LOG_FILE`**（容器内文件）。
> 极简 compose 不挂载卷，容器销毁日志丢失；想持久化就在 `docker-compose.yml` 里加 `volumes`。

## 故障排查

### 1. `ModuleNotFoundError: No module named 'config'`

`uv sync` 没装好，或当前目录不在项目根目录：

```bash
cd crawl4ai-service   # 进入项目根（包含 main.py 和 src/ 的目录）
uv sync
```

### 2. `playwright._impl._errors.Error: Executable doesn't exist at .../chromium-.../chrome`

Playwright 的 Chromium 没装。执行：

```bash
uv run playwright install chromium
# 或带系统依赖：
uv run playwright install --with-deps chromium
```

### 3. Playwright 不支持当前 Linux 发行版

类似报错：
```
Failed to install browsers
Error: ERROR: Playwright does not support chromium on ubuntu26.04-x64
```

Playwright 还没为该 Linux 发行版发布预编译 Chromium。**系统已有 `/usr/bin/google-chrome` 时的解决办法：**

在 `.env` 中加入：

```bash
BROWSER_EXECUTABLE_PATH=/usr/bin/google-chrome
BROWSER_TYPE=chrome
```

服务启动时会自动调用系统 Chrome，**无需 `playwright install`**。
`src/crawler.py` 中的 `_build_browser_config` 通过 `chrome_channel=settings.browser_type` 转发给 Crawl4AI
（注意：本版 Crawl4AI 的 `BrowserConfig` 只读 `chrome_channel`，裸的 `channel` 字段会被忽略）。

### 4. Docker 镜像构建卡在 `apt-get update` 十几分钟

`deb.debian.org` 国内直连 ~15kB/s。**只动 Dockerfile**，不动 daemon.json：

在 `RUN apt-get update \` 之前插入一行：

```dockerfile
RUN sed -i 's|deb.debian.org|mirrors.aliyun.com|g' /etc/apt/sources.list.d/debian.sources
```

切到阿里云后 ~30s-1min 完成。

### 5. Docker 构建卡在 `docker/dockerfile:1.7` 拉不下来

`# syntax=docker/dockerfile:1.7` 这行会让 BuildKit 拉语法解析器镜像，国内 mirror 对这个**非应用类元镜像**普遍没有缓存。

**两种绕开方法**：

- 临时用老 builder：`DOCKER_BUILDKIT=0 docker build -t jobinsight/crawl4ai-service:0.1.0 -f docker/Dockerfile .`
- 或者把 Dockerfile 顶上 `# syntax=docker/dockerfile:1.7` 注释掉再 build。

### 6. Docker 容器起不来 / 报浏览器找不到

[docker/Dockerfile](file:///home/jfeng/code/python/project/JobInsight/crawl4ai-service/docker/Dockerfile) 中已经包含 `python -m playwright install --with-deps chromium`。
如果构建过程此步失败，可临时改为挂载宿主机的浏览器：

```yaml
# docker/docker-compose.yml 中追加
volumes:
  - /usr/bin/google-chrome:/usr/bin/google-chrome:ro
environment:
  - BROWSER_EXECUTABLE_PATH=/usr/bin/google-chrome
  - BROWSER_TYPE=chrome
```

### 7. `/extract` 返回 `422` 且 `detail.loc` 中含 `llm`

表示请求体里没有 `llm` 字段。在请求体里加上：

```json
"llm": {
  "api_key": "sk-xxx",
  "base_url": "https://api.deepseek.com/v1",
  "model": "deepseek-chat"
}
```

`llm.api_key` / `base_url` / `model` 三者必填。

### 8. `/health`（或其它接口）返回 `401 code=401 msg="API Key 无效"` 或 `"缺少 Authorization Bearer 头"`

表示 `API_KEY` 已配置，但请求头里没带 / 带错了 Bearer token。检查：

1. 请求头是否正确：`Authorization: Bearer <your-key>`（注意拼写）。
2. 客户端是否被代理剥掉了 `Authorization` 头。
3. token 是否和服务端 `.env` 中的 `API_KEY` 完全一致。

### 9. 爬取某些站点（如 `jobs.bytedance.com`）第一次成功、第二次 `ACS-GOTO` 超时

字节系 / 部分大型站点部署了 **Akamai Bot Manager**。两次请求的现象解释了为什么：
- **第 1 次**（20-30s 成功）：通过了 ACS 的 JS challenge
- **第 2 次**（60s 超时）：触发"短时间内多次访问"规则，ACS 直接挡掉

**修法**（按推荐顺序）：

1. **改 UA**：别用 `JobInsight/0.1` 这种自签名 UA，用真实 Chrome UA：
   ```json
   "options": {
     "user_agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36"
   }
   ```
2. **请求间加间隔**：同一站点两次请求之间 `sleep 30s+`。
3. **`wait_for` 改具体元素**：等 ACS challenge 跑完。例如 `css:.list-item` 或 `networkidle`。
4. **上代理池**：在 `options.proxy` 字段里传 `http://user:pass@host:port`。
5. **避开 magic 模式**：`magic: false`（默认）。

如果需要批量爬取这些站点，建议加代理层和频率控制中间件，不要直接在主循环里连续调本服务。

### 10. 容器内日志查看

```bash
# 实时看容器 stdout（stderr 也走这里）
docker logs -f crawl4ai-service

# 进容器看日志文件
docker exec -it crawl4ai-service tail -f /app/logs/service.log
```

> 极简 compose 没有 volumes，容器销毁日志就丢。想持久化就加一行：
> ```yaml
> volumes:
>   - ./crawl4ai_logs:/app/logs
> ```

## 与主后端的集成

`backed/schemas/response.py` 已经定义了同款 `APIResponse` 包装，所以主后端拿到本服务
返回后**不需要重新解包**：

```python
# 主后端伪代码
resp = await client.post(
    "http://crawl4ai-service:11235/extract",
    json={
        "url": job_url,
        "instruction": "提取岗位名称、薪资、地点",
        "llm": llm_config_for_request,   # ← 按请求注入
    },
    headers={"Authorization": f"Bearer {CRAWL4AI_API_KEY}"},
)
payload = resp.json()
if payload["code"] == 200:
    job_info = payload["data"]["data"]   # ← 注意嵌套 data.data
else:
    raise BusinessError(payload["msg"])
```

容器间走 `jobinsight-net` bridge 网络，主后端可以用服务名 `crawl4ai-service:11235` 直接访问。
