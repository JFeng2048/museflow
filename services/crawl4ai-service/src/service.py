"""进程级共享服务单例。

HTTP（FastAPI）与 gRPC 两个版本的接口都从这里获取同一个
:class:`~src.crawler.CrawlerService` 实例与启动时间，避免重复初始化浏览器、
也保证两个端口对外暴露的行为完全一致（同一份爬取 / 提取逻辑）。

设计要点：

* ``CrawlerService`` 内部维护了 Crawl4AI 的浏览器池，构造较昂贵，必须单例。
* ``_STARTUP_TIME`` 供 ``/health`` 与 gRPC ``Health`` 计算 uptime，
  两个接口应使用同一个基准时间。
* 两个入口（``main.py`` 统一启动、或单独 ``grpc_server.py`` / HTTP 启动）都通过
  :func:`get_crawler_service` 与 :data:`STARTUP_TIME` 访问，保持口径统一。
"""
from __future__ import annotations

import threading
import time

from src.config import get_settings
from src.crawler import CrawlerService

# 服务进程启动时间（首次导入本模块时记录），供健康检查计算 uptime。
STARTUP_TIME: float = time.time()

_lock = threading.Lock()
_service: CrawlerService | None = None


def get_crawler_service() -> CrawlerService:
    """返回进程级 :class:`CrawlerService` 单例（线程安全，惰性初始化）。"""
    global _service
    if _service is None:
        with _lock:
            if _service is None:
                _service = CrawlerService(get_settings())
    return _service


def uptime_seconds() -> float:
    """服务启动至今的秒数。"""
    return time.time() - STARTUP_TIME
