"""
gRPC 服务入口：把原 FastAPI 版的 /crawl、/extract、/health 三个接口
映射为 CrawlService 的 Crawl / Extract / Health 三个 RPC。

底层抓取与抽取逻辑完全复用 src.crawler.CrawlerService 与 src.extractor，
因此依然依赖 Crawl4AI（浏览器渲染 + Markdown 清洗 + LLM 抽取），
仅把传输层从 HTTP/JSON 替换为 gRPC。

运行前需先生成 gRPC 桩代码（见 scripts/gen_proto.sh）：
    python -m grpc_tools.protoc -I../../proto/crawl \\
        --python_out=src --grpc_python_out=src \\
        ../../proto/crawl/crawl.proto

随后：python -m src.grpc_server
"""
from __future__ import annotations

import json
import logging
import os
import time
from concurrent import futures

import grpc

# 由 scripts/gen_proto.sh 生成的桩代码
import crawl_pb2
import crawl_pb2_grpc

from src.config import get_settings
from src.crawler import CrawlerService
from src.extractor import LLMSmartExtractor
from src.schema import CrawlerOptions, ExtractSchema, LLMConfig
from src.service import get_crawler_service, uptime_seconds

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")
logger = logging.getLogger("crawl4ai.grpc")

# 服务名 / 版本直接取自配置，保持与 HTTP 接口一致。
SERVICE_NAME = "crawl4ai-service"


def _to_crawler_options(pb_options) -> CrawlerOptions:
    """把 proto CrawlerOptions 转为 src.schema.CrawlerOptions。"""
    if pb_options is None:
        return CrawlerOptions()
    kwargs = {}
    if pb_options.timeout:
        kwargs["timeout"] = pb_options.timeout
    if pb_options.user_agent:
        kwargs["user_agent"] = pb_options.user_agent
    if pb_options.bypass_cache:
        kwargs["bypass_cache"] = True
    if pb_options.remove_overlay_elements:
        kwargs["remove_overlay_elements"] = True
    if pb_options.simulate_user:
        kwargs["simulate_user"] = True
    if pb_options.magic:
        kwargs["magic"] = True
    if pb_options.locale:
        kwargs["locale"] = pb_options.locale
    return CrawlerOptions(**kwargs)


def _build_llm_config(pb_llm) -> LLMConfig:
    """把 proto LLMConfig 转为 src.schema.LLMConfig。"""
    return LLMConfig(
        api_key=pb_llm.api_key or os.getenv("OPENAI_API_KEY", ""),
        base_url=pb_llm.base_url or os.getenv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
        model=pb_llm.model or os.getenv("OPENAI_MODEL", "gpt-4o-mini"),
        temperature=pb_llm.temperature or 0.0,
        max_tokens=pb_llm.max_tokens or 2048,
        request_timeout=pb_llm.request_timeout or 120,
    )


def _build_schema_fields(fields):
    """把 proto schema_fields 列表转为 list[ExtractSchema]。"""
    out = []
    for f in fields:
        item = None
        if f.items_json:
            try:
                item = json.loads(f.items_json)
            except json.JSONDecodeError:
                logger.warning("schema field %s 的 items_json 不是合法 JSON，已忽略", f.name)
        out.append(
            ExtractSchema(
                name=f.name,
                description=f.description,
                type=f.type or "string",
                required=f.required,
                items=item,
            )
        )
    return out


class CrawlServicer(crawl_pb2_grpc.CrawlServiceServicer):
    """CrawlService 的 gRPC 实现。"""

    def __init__(self, crawler: CrawlerService):
        self._crawler = crawler

    def Health(self, request, context):
        settings = get_settings()
        return crawl_pb2.HealthResponse(
            healthy=True,
            service=SERVICE_NAME,
            version=settings.service_version,
            uptime_seconds=round(uptime_seconds(), 2),
            auth_enabled=bool(settings.api_key),
        )

    def Crawl(self, request, context):
        start = time.time()
        if not request.url:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("url 不能为空")
            return crawl_pb2.CrawlResponse(
                success=False, error_code="INVALID_URL", error_message="url 不能为空", elapsed_ms=0
            )
        options = _to_crawler_options(request.options)
        try:
            result = self._crawler.crawl(
                url=str(request.url),
                options=options,
                wait_for=request.wait_for or None,
            )
        except Exception as exc:  # noqa: BLE001 - 兜底，避免 gRPC 内部异常泄露
            logger.exception("Crawl 未预期错误: %s", exc)
            return crawl_pb2.CrawlResponse(
                success=False,
                url=request.url,
                error_code="UNEXPECTED_ERROR",
                error_message=str(exc),
                error_retryable=False,
                elapsed_ms=int((time.time() - start) * 1000),
            )

        return self._crawl_result_to_proto(result, start)

    @staticmethod
    def _crawl_result_to_proto(result, start) -> crawl_pb2.CrawlResponse:
        elapsed = int((time.time() - start) * 1000)
        if not result.success:
            err = result.error
            return crawl_pb2.CrawlResponse(
                success=False,
                url=result.url,
                status_code=result.status_code or 0,
                error_code=err.code if err else "CRAWL_FAILED",
                error_message=err.message if err else "抓取失败",
                error_retryable=err.retryable if err else False,
                elapsed_ms=result.elapsed_ms or elapsed,
            )
        return crawl_pb2.CrawlResponse(
            success=True,
            url=result.url,
            markdown=result.markdown or "",
            status_code=result.status_code or 0,
            elapsed_ms=result.elapsed_ms or elapsed,
        )

    def Extract(self, request, context):
        start = time.time()
        if not request.url:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("url 不能为空")
            return crawl_pb2.ExtractResponse(
                success=False, error_code="INVALID_URL", error_message="url 不能为空", elapsed_ms=0
            )
        if not request.instruction and not request.schema_fields:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("instruction 与 schema_fields 至少提供其一")
            return crawl_pb2.ExtractResponse(
                success=False,
                error_code="INVALID_SCHEMA",
                error_message="instruction 与 schema_fields 至少提供其一",
                elapsed_ms=0,
            )

        llm_config = _build_llm_config(request.llm)
        extractor = LLMSmartExtractor(
            llm_config=llm_config,
            instruction=request.instruction or "",
            schema_fields=_build_schema_fields(request.schema_fields) or None,
            extraction_timeout=request.extraction_timeout or 120,
        )
        options = _to_crawler_options(request.options)

        try:
            result = self._crawler.extract(
                url=str(request.url),
                options=options,
                extractor=extractor,
            )
        except Exception as exc:  # noqa: BLE001
            logger.exception("Extract 未预期错误: %s", exc)
            return crawl_pb2.ExtractResponse(
                success=False,
                url=request.url,
                error_code="UNEXPECTED_ERROR",
                error_message=str(exc),
                error_retryable=False,
                elapsed_ms=int((time.time() - start) * 1000),
            )

        return self._extract_result_to_proto(result, start)

    @staticmethod
    def _extract_result_to_proto(result, start) -> crawl_pb2.ExtractResponse:
        elapsed = int((time.time() - start) * 1000)
        if not result.success:
            err = result.error
            return crawl_pb2.ExtractResponse(
                success=False,
                url=result.url,
                error_code=err.code if err else "EXTRACT_FAILED",
                error_message=err.message if err else "抽取失败",
                error_retryable=err.retryable if err else False,
                elapsed_ms=result.elapsed_ms or elapsed,
            )
        data_json = json.dumps(result.data or {}, ensure_ascii=False)
        return crawl_pb2.ExtractResponse(
            success=True,
            url=result.url,
            markdown=result.markdown or "",
            data_json=data_json,
            model=result.model or "",
            elapsed_ms=result.elapsed_ms or elapsed,
        )


def start_grpc(settings=None) -> grpc.Server:
    """构造并启动 gRPC 服务，返回 server 对象（供统一入口 main.py 调用）。

    使用 :func:`src.service.get_crawler_service` 复用与 HTTP 接口相同的
    :class:`CrawlerService` 单例，保证两个版本的爬取 / 提取逻辑完全一致。
    """
    settings = settings or get_settings()
    crawler = get_crawler_service()
    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=settings.default_max_concurrent or 5),
        options=[
            ("grpc.max_send_message_length", 100 * 1024 * 1024),
            ("grpc.max_receive_message_length", 100 * 1024 * 1024),
        ],
    )
    crawl_pb2_grpc.add_CrawlServiceServicer_to_server(CrawlServicer(crawler), server)
    listen_addr = f"[::]:{settings.port}"
    server.add_insecure_port(listen_addr)
    server.start()
    logger.info("Crawl4AI gRPC 服务已启动，监听 %s", listen_addr)
    return server


def serve() -> None:
    """单独运行 gRPC 版本（``python -m src.grpc_server``）。"""
    server = start_grpc()
    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        server.stop(grace=2)


if __name__ == "__main__":
    serve()
