"""crawl4ai-service 的 FastAPI 入口。

仅负责：

* 构造 :class:`FastAPI` 实例与元信息；
* 启动 / 关闭跨应用生命周期的 :class:`~src.crawler.CrawlerService`；
* 挂载 :mod:`src.api` 提供的 APIRouter 与异常处理器。

所有 3 个对外接口（``/health``、``/crawl``、``/extract``）都在
:mod:`src.api` 中定义。
"""
from __future__ import annotations

import logging
import sys
from collections.abc import AsyncGenerator
from contextlib import asynccontextmanager
from pathlib import Path

from fastapi import FastAPI

from src.api import register_exception_handlers, router as api_router
from src.config import get_settings
from src.crawler import CrawlerService


# ---- 日志 ----
def _setup_logging() -> logging.Logger:
    """配置日志：stderr + 可挂载的文件处理器。

    文件路径由 ``LOG_DIR`` / ``LOG_FILE`` 控制（默认 ``/app/logs/service.log``），
    容器内 ``/app/logs`` 挂载到宿主机后可直接 ``tail -f`` 排查。
    若目录不可写，自动降级为只输出到 stderr。
    """
    s = get_settings()
    level = getattr(logging, s.log_level.upper(), logging.INFO)
    formatter = logging.Formatter(
        fmt="%(asctime)s [%(levelname)s] %(name)s :: %(message)s",
    )

    root = logging.getLogger()
    root.setLevel(level)
    # 清掉可能存在的旧 handler，避免与 uvicorn 重复输出
    for h in list(root.handlers):
        root.removeHandler(h)

    # 1) stderr：保留 ``docker logs`` / ``journalctl`` 可读性
    stream_handler = logging.StreamHandler(stream=sys.stderr)
    stream_handler.setFormatter(formatter)
    root.addHandler(stream_handler)

    # 2) 文件：挂载到宿主机，便于持久化与排查
    try:
        log_path = Path(s.log_dir) / s.log_file
        log_path.parent.mkdir(parents=True, exist_ok=True)
        file_handler = logging.FileHandler(log_path, encoding="utf-8")
        file_handler.setFormatter(formatter)
        root.addHandler(file_handler)
    except OSError as exc:
        sys.stderr.write(
            f"[warn] 日志文件初始化失败（{exc}），仅保留 stderr 输出\n"
        )

    return logging.getLogger("crawl4ai-service")


logger = _setup_logging()
settings = get_settings()


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncGenerator[None]:
    """管理跨应用生命周期的长连接 :class:`CrawlerService`。"""
    logger.info("正在启动 %s v%s", settings.service_name, settings.service_version)
    if settings.api_key:
        logger.info("已启用 API Key 认证")
    else:
        logger.warning("API_KEY 未配置，认证已禁用（仅供本地开发）")
    service = CrawlerService(settings)
    app.state.crawler_service = service
    try:
        await service.start()
        logger.info("爬虫已就绪")
        yield
    finally:
        logger.info("正在关闭爬虫")
        await service.stop()


app = FastAPI(
    title="Crawl4AI Service",
    version=settings.service_version,
    description="封装 Crawl4AI 的智能网页提取独立微服务。",
    lifespan=lifespan,
)

# 注册路由与全局异常处理器。
app.include_router(api_router)
register_exception_handlers(app)


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "main:app",
        host=settings.host,
        port=settings.port,
        log_level="info",
    )