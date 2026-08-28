"""crawl4ai-service 统一启动入口。

同一个服务提供**两个版本的接口**，二者共用同一份爬取 / 提取核心逻辑
（``src/crawler.CrawlerService`` 单例）：

1. **HTTP（FastAPI）**：RESTful JSON，带 Swagger UI（``/docs``），
   适合浏览器调试、外部系统快速对接。监听 ``PORT``（默认 5003）。
2. **gRPC**：由 ``proto/crawl/crawl.proto`` 定义，``Health`` / ``Crawl`` / ``Extract``
   三个 RPC，适合 Go 网关 / 内部高性能调用。同样监听 ``PORT``（默认 5003）。

两个版本通过依赖分组解耦（见 ``pyproject.toml`` 的 ``[project.optional-dependencies]``）：

* 只打包 HTTP：``uv sync --extra http`` → 运行 ``python -m main`` 只起 HTTP
* 只打包 gRPC：``uv sync --extra grpc`` → 运行 ``python -m main`` 只起 gRPC
* 两个都打包：``uv sync --extra http --extra grpc`` → 同时起两个版本

本入口根据**实际安装的依赖**决定启动哪些版本（见 :func:`src.config.http_available`
/ :func:`src.config.grpc_available`），无需环境变量开关；若两个依赖组都没装则报错退出。

运行前，gRPC 版本需先生成桩代码（见 ``scripts/gen_proto.sh``）：

```bash
bash scripts/gen_proto.sh   # 仅 gRPC 版本需要
python -m main              # 按已安装依赖自动启动对应版本
```

也可直接运行某一版本的入口（调试更直观）：

```bash
python -m src.grpc_server   # 仅 gRPC
python -m src.api           # 仅 HTTP（src/api.py 的 __main__）
```
"""
from __future__ import annotations

import logging
import threading

from src.config import get_settings, grpc_available, http_available

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
)
logger = logging.getLogger("crawl4ai.main")


def _start_http(settings) -> None:
    """在子线程中启动 FastAPI（uvicorn）。"""
    import uvicorn

    from src.api import build_app

    app = build_app()
    logger.info("Crawl4AI HTTP 服务已启动，监听 %s:%d", settings.host, settings.port)
    uvicorn.run(
        app,
        host=settings.host,
        port=settings.port,
        log_level=settings.log_level.lower(),
    )


def main() -> None:
    settings = get_settings()
    want_http = http_available()
    want_grpc = grpc_available()

    logger.info(
        "crawl4ai-service v%s 启动中（http=%s, grpc=%s, port=%d）",
        settings.service_version,
        want_http,
        want_grpc,
        settings.port,
    )

    # HTTP 与 gRPC 共用同一个端口（PORT），两种传输协议不能在该端口上同时监听。
    # 因此若两个依赖组都装了，这里优先启动 gRPC，并跳过 HTTP 以避免端口冲突；
    # 需要 HTTP 时请只装 http 组（uv sync --extra http）。
    if want_http and want_grpc:
        logger.warning(
            "HTTP 与 gRPC 共用端口 %d，不能同时监听；本次仅启动 gRPC。"
            "如需 HTTP 版本，请用 `uv sync --extra http` 仅安装 http 组后运行。",
            settings.port,
        )
        want_http = False

    threads: list[threading.Thread] = []
    grpc_server = None

    if want_http:
        t = threading.Thread(target=_start_http, args=(settings,), daemon=True)
        t.start()
        threads.append(t)

    if want_grpc:
        from src.grpc_server import start_grpc

        grpc_server = start_grpc(settings)

    if not threads and grpc_server is None:
        raise SystemExit(
            "未安装任何版本接口的依赖：请使用 `uv sync --extra http` 或 "
            "`uv sync --extra grpc` 安装对应依赖组后再启动。"
        )

    # 主线程阻塞：gRPC 用 wait_for_termination；若未启用 gRPC，则 join HTTP 线程。
    try:
        if grpc_server is not None:
            grpc_server.wait_for_termination()
        else:
            for t in threads:
                t.join()
    except KeyboardInterrupt:
        logger.info("收到中断信号，正在关闭…")
        if grpc_server is not None:
            grpc_server.stop(grace=2)


if __name__ == "__main__":
    main()
