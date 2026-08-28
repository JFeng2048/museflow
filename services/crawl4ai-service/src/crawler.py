"""Crawl4AI 的封装层，提供单 URL 爬取 / 提取与重试能力。"""
from __future__ import annotations

import asyncio
import time
from typing import Any

from crawl4ai import AsyncWebCrawler, BrowserConfig, CrawlerRunConfig
from crawl4ai.extraction_strategy import ExtractionStrategy
from tenacity import (
    AsyncRetrying,
    RetryError,
    retry_if_exception_type,
    stop_after_attempt,
    wait_exponential,
)

from src.config import Settings
from src.extractor import LLMSmartExtractor, safe_json_loads
from src.schema import CrawlData, CrawlerOptions, ErrorInfo, ExtractData


# 安全可重试的异常类型（瞬时网络/服务器错误）。
RETRYABLE_EXCEPTIONS: tuple[type[BaseException], ...] = (
    asyncio.TimeoutError,
    ConnectionError,
)


def _build_browser_config(settings: Settings, options: CrawlerOptions) -> BrowserConfig:
    """根据服务配置 + 单次请求选项构建 :class:`BrowserConfig`。

    若 ``settings.browser_executable_path`` 非空（例如 ``/usr/bin/google-chrome``）
    或 ``settings.browser_type`` 非默认的 ``chromium``，
    通过 ``chrome_channel`` 字段让 Playwright 调用系统浏览器，
    从而绕过 ``playwright install chromium``，
    适合 Playwright 暂未支持 Chromium 的 Linux 发行版（如 ubuntu26.04）。

    注意：本版 Crawl4AI 的 :class:`BrowserConfig` 内部只读取 ``chrome_channel``
    （见 ``browser_manager.py:1115`` 的 ``if self.config.chrome_channel``），
    把它转发到 Playwright 的 ``channel`` 参数。**裸的 ``channel`` 字段会被忽略。**
    ``chrome_channel`` 接受 ``chrome`` / ``msedge`` / ``chrome-beta`` 等。
    """
    cfg_kwargs: dict[str, Any] = {
        "headless": settings.headless,
        "user_agent": options.user_agent or settings.user_agent,
        "verbose": False,
        "enable_stealth": settings.enable_stealth,
    }
    use_system_browser = bool(settings.browser_executable_path) or settings.browser_type != "chromium"
    if use_system_browser:
        cfg_kwargs["chrome_channel"] = settings.browser_type
    return BrowserConfig(**cfg_kwargs)


def _build_run_config(
    options: CrawlerOptions,
    extraction_strategy: ExtractionStrategy | None = None,
    timeout: int = 60,
    wait_for: str | None = None,
) -> CrawlerRunConfig:
    """构建 :class:`CrawlerRunConfig`。

    ``wait_for`` 在 :class:`CrawlRequest` 中是顶层字段（不在
    :class:`CrawlerOptions` 里），由调用方显式传入。
    """
    return CrawlerRunConfig(
        # Crawl4AI 内部使用毫秒为单位。
        page_timeout=timeout * 1000,
        wait_for=wait_for,
        bypass_cache=options.bypass_cache,
        remove_overlay_elements=options.remove_overlay_elements,
        simulate_user=options.simulate_user,
        magic=options.magic,
        locale=options.locale,
        extraction_strategy=extraction_strategy,
    )


class CrawlerService:
    """对 :class:`AsyncWebCrawler` 的高级异步封装（单 URL 入口）。"""

    def __init__(self, settings: Settings) -> None:
        self._settings = settings
        self._crawler: AsyncWebCrawler | None = None
        self._lock = asyncio.Lock()

    async def start(self) -> None:
        """初始化底层的浏览器爬虫。"""
        async with self._lock:
            if self._crawler is None:
                self._crawler = AsyncWebCrawler(
                    config=_build_browser_config(self._settings, CrawlerOptions())
                )
                await self._crawler.start()

    async def stop(self) -> None:
        """关闭底层的浏览器爬虫。"""
        async with self._lock:
            if self._crawler is not None:
                await self._crawler.close()
                self._crawler = None

    async def __aenter__(self) -> "CrawlerService":
        await self.start()
        return self

    async def __aexit__(self, exc_type, exc, tb) -> None:
        await self.stop()

    # ------------------------------------------------------------
    # /crawl
    # ------------------------------------------------------------
    async def crawl(
        self,
        url: str,
        options: CrawlerOptions,
        wait_for: str | None = None,
    ) -> CrawlData:
        """爬取单 URL 并返回清洗后的 Markdown。"""
        if self._crawler is None:
            await self.start()
        assert self._crawler is not None  # noqa: S101

        timeout = options.timeout or self._settings.default_timeout
        run_cfg = _build_run_config(
            options, extraction_strategy=None, timeout=timeout, wait_for=wait_for
        )
        return await self._crawl_with_retry(url=url, run_cfg=run_cfg)

    async def _crawl_with_retry(self, url: str, run_cfg: CrawlerRunConfig) -> CrawlData:
        """爬取单个 URL，遇到瞬时错误时按指数退避进行重试。"""
        last_error: ErrorInfo | None = None
        assert self._crawler is not None  # noqa: S101

        try:
            async for attempt in AsyncRetrying(
                stop=stop_after_attempt(self._settings.max_retries + 1),
                wait=wait_exponential(
                    multiplier=self._settings.retry_backoff_factor,
                    min=1,
                    max=10,
                ),
                retry=retry_if_exception_type(RETRYABLE_EXCEPTIONS),
                reraise=True,
            ):
                with attempt:
                    start = time.perf_counter()
                    try:
                        result = await self._crawler.arun(url=url, config=run_cfg)
                    except RETRYABLE_EXCEPTIONS as exc:
                        last_error = ErrorInfo(
                            code=type(exc).__name__,
                            message=str(exc),
                            retryable=True,
                        )
                        raise
                    elapsed_ms = int((time.perf_counter() - start) * 1000)
                    return self._build_crawl_data(url=url, result=result, elapsed_ms=elapsed_ms)
        except RetryError:
            pass
        except RETRYABLE_EXCEPTIONS as exc:
            last_error = ErrorInfo(
                code=type(exc).__name__,
                message=str(exc),
                retryable=True,
            )

        return CrawlData(
            url=url,
            success=False,
            error=last_error or ErrorInfo(code="未知错误", message="未知错误", retryable=False),
        )

    @staticmethod
    def _build_crawl_data(
        url: str,
        result: Any,
        elapsed_ms: int,
    ) -> CrawlData:
        """把 Crawl4AI 的 CrawlResult 归一化为 :class:`CrawlData`。"""
        if not getattr(result, "success", False):
            return CrawlData(
                url=url,
                success=False,
                status_code=getattr(result, "status_code", None),
                error=ErrorInfo(
                    code="CRAWL_FAILED",
                    message=getattr(result, "error_message", "爬取失败"),
                    retryable=False,
                ),
                elapsed_ms=elapsed_ms,
            )

        markdown_obj = getattr(result, "markdown", None)
        markdown_str: str | None = None
        if markdown_obj is not None:
            # ``markdown`` 可能是 MarkdownGenerationResult 对象或字符串。
            markdown_str = getattr(markdown_obj, "raw_markdown", None) or (
                markdown_obj if isinstance(markdown_obj, str) else None
            )

        return CrawlData(
            url=url,
            success=True,
            status_code=getattr(result, "status_code", None),
            markdown=markdown_str,
            elapsed_ms=elapsed_ms,
        )

    # ------------------------------------------------------------
    # /extract
    # ------------------------------------------------------------
    async def extract(
        self,
        url: str,
        options: CrawlerOptions,
        extractor: LLMSmartExtractor,
        model: str | None = None,
    ) -> ExtractData:
        """对单 URL 抓取并提取结构化数据。"""
        if self._crawler is None:
            await self.start()
        assert self._crawler is not None  # noqa: S101

        timeout = options.timeout or self._settings.default_timeout
        strategy = extractor.build()
        run_cfg = _build_run_config(options, extraction_strategy=strategy, timeout=timeout)
        result_data = await self._extract_with_retry(url=url, run_cfg=run_cfg)
        # 注入本次请求所用的模型名（仅在成功时）。
        if model and result_data.success and result_data.model is None:
            result_data = result_data.model_copy(update={"model": model})
        return result_data

    async def _extract_with_retry(self, url: str, run_cfg: CrawlerRunConfig) -> ExtractData:
        """对单 URL 提取，遇到瞬时错误时进行重试。"""
        last_error: ErrorInfo | None = None
        assert self._crawler is not None  # noqa: S101

        try:
            async for attempt in AsyncRetrying(
                stop=stop_after_attempt(self._settings.max_retries + 1),
                wait=wait_exponential(
                    multiplier=self._settings.retry_backoff_factor,
                    min=1,
                    max=10,
                ),
                retry=retry_if_exception_type(RETRYABLE_EXCEPTIONS),
                reraise=True,
            ):
                with attempt:
                    start = time.perf_counter()
                    try:
                        result = await self._crawler.arun(url=url, config=run_cfg)
                    except RETRYABLE_EXCEPTIONS as exc:
                        last_error = ErrorInfo(
                            code=type(exc).__name__,
                            message=str(exc),
                            retryable=True,
                        )
                        raise
                    elapsed_ms = int((time.perf_counter() - start) * 1000)
                    return self._build_extract_data(url, result, elapsed_ms)
        except RetryError:
            pass
        except RETRYABLE_EXCEPTIONS as exc:
            last_error = ErrorInfo(
                code=type(exc).__name__,
                message=str(exc),
                retryable=True,
            )

        return ExtractData(
            url=url,
            success=False,
            error=last_error or ErrorInfo(code="未知错误", message="未知错误", retryable=False),
        )

    @staticmethod
    def _build_extract_data(
        url: str,
        result: Any,
        elapsed_ms: int,
    ) -> ExtractData:
        """把 Crawl4AI 的 CrawlResult 归一化为 :class:`ExtractData`。"""
        if not getattr(result, "success", False):
            return ExtractData(
                url=url,
                success=False,
                error=ErrorInfo(
                    code="CRAWL_FAILED",
                    message=getattr(result, "error_message", "爬取失败"),
                    retryable=False,
                ),
                elapsed_ms=elapsed_ms,
            )

        extracted = getattr(result, "extracted_content", None)
        data: dict[str, Any] = safe_json_loads(extracted) if extracted else {}

        markdown_obj = getattr(result, "markdown", None)
        raw_markdown = getattr(markdown_obj, "raw_markdown", None) or (
            markdown_obj if isinstance(markdown_obj, str) else None
        )

        return ExtractData(
            url=url,
            success=True,
            markdown=raw_markdown,
            data=data,
            elapsed_ms=elapsed_ms,
        )