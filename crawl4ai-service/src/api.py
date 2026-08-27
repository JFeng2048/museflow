"""crawl4ai-service 的 HTTP API 路由注册。

使用 :class:`fastapi.APIRouter` 集中管理 3 个对外接口 + 1 个全局异常处理器，
并通过 router 级 :func:`verify_api_key` 依赖统一加 Bearer 认证。

认证基于 :class:`fastapi.security.HTTPBearer`（token 形式：``Authorization:
Bearer <token>``），会在 Swagger UI 顶部显示 **Authorize** 按钮，调用方只需
在弹窗里填一次 token 即可作用于全部接口。token 的值就是环境变量
``API_KEY`` 配置的密钥（这里把 ``API_KEY`` 当作一个 long-lived static token
使用，不做 JWT 签名校验——简单场景够用，生产建议替换为带签名的 JWT）。

所有响应统一用 :class:`~src.schema.APIResponse` 包装：

```
{ "code": 200, "msg": "成功", "data": { ... } }
```
"""
from __future__ import annotations

import logging
import time

from fastapi import APIRouter, Depends, FastAPI, HTTPException, Request, status
from fastapi.responses import JSONResponse
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer

from src.config import get_settings
from src.crawler import CrawlerService
from src.extractor import build_extractor
from src.schema import (
    APIResponse,
    CrawlData,
    CrawlRequest,
    ExtractData,
    ExtractRequest,
    HealthData,
    ok,
)

logger = logging.getLogger("crawl4ai-service")

# 进程级启动时间，/health 据此计算 uptime。
_STARTUP_TIME = time.time()

# Swagger UI 顶部的 "Authorize" 按钮靠它生成。
# ``bearerFormat="API Key"`` 让 Swagger 弹窗标题提示 "Value: API Key"。
# ``auto_error=False`` 让我们在依赖内部自定义 401 文案/包装。
_bearer_scheme = HTTPBearer(bearerFormat="API Key", auto_error=False)


# ============================================================
# 认证依赖
# ============================================================
async def verify_api_key(
    credentials: HTTPAuthorizationCredentials | None = Depends(_bearer_scheme),
) -> None:
    """简单的 Bearer token 认证。

    * 环境变量 ``API_KEY`` 为空时，**禁用认证**（仅供本地开发）。
    * 配置了 ``API_KEY`` 时，所有挂载该依赖的接口都必须携带匹配的 Bearer token：
      ``Authorization: Bearer <API_KEY>``。
    * 接入方式为 :class:`HTTPBearer`，
      会在 Swagger UI 顶部渲染 **Authorize** 按钮，
      调用方只需在该弹窗里填一次 token 即可作用于全部接口。
    """
    settings = get_settings()
    if not settings.api_key:
        return  # 未配置 → 跳过认证
    if credentials is None or credentials.scheme.lower() != "bearer":
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail={"code": "UNAUTHORIZED", "msg": "缺少 Authorization Bearer 头"},
            headers={"WWW-Authenticate": "Bearer"},
        )
    if credentials.credentials != settings.api_key:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail={"code": "UNAUTHORIZED", "msg": "API Key 无效"},
            headers={"WWW-Authenticate": "Bearer"},
        )


# 顶层 APIRouter；tags 决定 OpenAPI 文档中的分组；
# dependencies 让所有路由都强制走 X-API-Key 认证。
router = APIRouter(
    dependencies=[Depends(verify_api_key)],
)


def _get_crawler(request: Request) -> CrawlerService:
    """从 FastAPI 应用状态中获取 :class:`CrawlerService` 单例。"""
    service: CrawlerService | None = getattr(request.app.state, "crawler_service", None)
    if service is None:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="爬虫尚未初始化",
        )
    return service


# ============================================================
# /health 健康检查
# ============================================================
@router.get(
    "/health",
    response_model=APIResponse[HealthData],
    tags=["元信息"],
    summary="健康检查",
    description=(
        "存活 / 就绪探针接口。\n\n"
        "- **无需任何参数**。\n"
        "- 返回 ``uptime_seconds``（服务启动至当前的秒数）和 ``auth_enabled``\n"
        "  （``API_KEY`` 是否已配置，对应是否启用了 Bearer 认证）。\n"
        "- 常用于 k8s liveness / readiness 探针或负载均衡健康检查。\n\n"
        "**认证**：本接口也受 Bearer token 保护。\n"
        "当 ``API_KEY`` 已配置时，请求头必须携带 ``Authorization: Bearer <API_KEY>``，\n"
        "否则返回 ``401``。\n\n"
        "**响应字段**：\n\n"
        "| 字段              | 类型    | 说明                                  |\n"
        "| ----------------- | ------- | ------------------------------------- |\n"
        "| ``status``        | string  | 固定 ``ok``                           |\n"
        "| ``service``       | string  | 服务名（``SERVICE_NAME``）            |\n"
        "| ``version``       | string  | 服务版本（``SERVICE_VERSION``）        |\n"
        "| ``uptime_seconds``| number  | 启动至今的秒数                        |\n"
        "| ``auth_enabled``  | boolean | 是否已启用 Bearer token 认证          |"
    ),
    response_description="服务存活且返回当前运行信息。",
    responses={
        200: {
            "description": "服务存活。",
            "content": {
                "application/json": {
                    "example": {
                        "code": 200,
                        "msg": "成功",
                        "data": {
                            "status": "ok",
                            "service": "crawl4ai-service",
                            "version": "0.1.0",
                            "uptime_seconds": 12.345,
                            "auth_enabled": True,
                        },
                    }
                }
            },
        },
        401: {"description": "缺少或错误的 Bearer token。"},
    },
)
async def health() -> APIResponse[HealthData]:
    """存活探针接口。"""
    settings = get_settings()
    return ok(
        HealthData(
            status="ok",
            service=settings.service_name,
            version=settings.service_version,
            uptime_seconds=time.time() - _STARTUP_TIME,
            auth_enabled=bool(settings.api_key),
        )
    )


# ============================================================
# /crawl 基础爬取
# ============================================================
@router.post(
    "/crawl",
    response_model=APIResponse[CrawlData],
    tags=["爬取"],
    summary="爬取单个 URL，返回清洗后的 Markdown",
    description=(
        "使用 Crawl4AI 抓取单个目标 URL，**只走浏览器渲染 + Markdown 清洗**，\n"
        "**不调用任何 LLM**。\n\n"
        "如需结构化数据，请改用 `/extract`。\n\n"
        "**请求体字段**：\n\n"
        "| 字段            | 类型   | 必填 | 说明                                          |\n"
        "| --------------- | ------ | ---- | --------------------------------------------- |\n"
        "| ``url``         | string | ✅   | 目标 URL（必须是合法的 http/https URL）       |\n"
        "| ``wait_for``    | string | ❌   | 爬取前需要等待出现的 CSS 选择器              |\n"
        "| ``options``     | object | ❌   | 爬取行为选项（见下表）                        |\n\n"
        "**options 子字段**：\n\n"
        "| 字段                     | 类型    | 默认  | 说明                              |\n"
        "| ------------------------ | ------- | ----- | --------------------------------- |\n"
        "| ``timeout``              | int     | 60    | 单 URL 超时（秒，5-600）          |\n"
        "| ``user_agent``           | string  | —     | 覆盖默认 UA                       |\n"
        "| ``bypass_cache``         | boolean | false | 跳过缓存结果                      |\n"
        "| ``remove_overlay_elements`` | boolean | false | 移除弹窗/遮罩元素              |\n"
        "| ``simulate_user``        | boolean | false | 模拟人类操作行为                  |\n"
        "| ``magic``                | boolean | false | 启用 Crawl4AI 魔法模式（反爬）    |\n"
        "| ``locale``               | string  | —     | 浏览器语言区域，如 ``zh-CN``      |\n\n"
        "**响应字段（``data``）**：\n\n"
        "| 字段           | 类型    | 说明                                  |\n"
        "| -------------- | ------- | ------------------------------------- |\n"
        "| ``success``    | boolean | 是否爬取成功                          |\n"
        "| ``url``        | string  | 实际请求的 URL（已规范化）            |\n"
        "| ``markdown``   | string  | 清洗后的 Markdown 内容                |\n"
        "| ``status_code``| int     | HTTP 响应状态码（可能为 null）        |\n"
        "| ``error``      | object  | 失败时的错误描述（成功时为 null）     |\n"
        "| ``elapsed_ms`` | int     | 服务端处理耗时（毫秒）                |"
    ),
    response_description="返回清洗后的 Markdown 内容。",
    responses={
        200: {
            "description": "成功，返回 Markdown。",
            "content": {
                "application/json": {
                    "example": {
                        "code": 200,
                        "msg": "成功",
                        "data": {
                            "success": True,
                            "url": "https://example.com",
                            "markdown": "# Example Domain\n\nThis is an example...",
                            "status_code": 200,
                            "error": None,
                            "elapsed_ms": 1234,
                        },
                    }
                }
            },
        },
        401: {"description": "缺少或错误的 Bearer token。"},
        422: {"description": "请求体不合法（缺 url、URL 格式错等）。"},
        500: {"description": "服务端内部错误。"},
        503: {"description": "爬虫未初始化（lifespan 失败）。"},
    },
)
async def crawl(req: CrawlRequest, request: Request) -> APIResponse[CrawlData]:
    """爬取单 URL 并返回 Markdown。"""
    started = time.perf_counter()
    logger.info("crawl：%s（wait_for=%s）", req.url, req.wait_for)

    data = await _get_crawler(request).crawl(
        url=str(req.url),
        options=req.options,
        wait_for=req.wait_for,
    )
    elapsed_ms = int((time.perf_counter() - started) * 1000)
    logger.info(
        "crawl done：%s success=%s elapsed=%dms",
        req.url,
        data.success,
        elapsed_ms,
    )
    return ok(data)


# ============================================================
# /extract 智能提取（始终走 LLM）
# ============================================================
@router.post(
    "/extract",
    response_model=APIResponse[ExtractData],
    tags=["提取"],
    summary="从单个 URL 用 LLM 智能提取结构化数据",
    description=(
        "对单个目标 URL 抓取页面 → 走 LLM → 按 ``instruction`` 抽取结构化 JSON。\n\n"
        "**LLM 凭证不会从环境变量读取**；调用方必须在请求体的 ``llm`` 字段里\n"
        "显式提供 ``api_key`` / ``base_url`` / ``model``，本服务再原样转发到任意\n"
        "OpenAI 兼容服务（DeepSeek、Together、Qwen、火山方舟、自建网关…）。\n\n"
        "**请求体字段**：\n\n"
        "| 字段                | 类型   | 必填 | 说明                                          |\n"
        "| ------------------- | ------ | ---- | --------------------------------------------- |\n"
        "| ``url``             | string | ✅   | 目标 URL                                      |\n"
        "| ``instruction``     | string | ✅   | 提取规则的自然语言描述（驱动 LLM）            |\n"
        "| ``schema_fields``   | array  | ❌   | 可选 JSON Schema 字段定义，约束 LLM 输出      |\n"
        "| ``llm``             | object | ✅   | LLM 配置（见下表）                            |\n"
        "| ``options``         | object | ❌   | 爬取行为选项（与 ``/crawl`` 一致）            |\n"
        "| ``extraction_timeout`` | int | ❌   | 单次 LLM 调用超时（秒，10-600，默认 120）     |\n\n"
        "**llm 子字段（OpenAI 兼容）**：\n\n"
        "| 字段              | 必填 | 默认  | 说明                                          |\n"
        "| ----------------- | ---- | ----- | --------------------------------------------- |\n"
        "| ``api_key``       | ✅   | —     | LLM API Key                                   |\n"
        "| ``base_url``      | ✅   | —     | LLM base URL（如 ``https://api.deepseek.com/v1``） |\n"
        "| ``model``         | ✅   | —     | 模型名                                        |\n"
        "| ``temperature``   | ❌   | 0.0   | 采样温度（0.0-2.0）                           |\n"
        "| ``max_tokens``    | ❌   | 2048  | 最大输出 token（64-32768）                    |\n"
        "| ``request_timeout`` | ❌ | 120   | LLM 请求超时（秒，10-600）                    |\n\n"
        "**schema_fields 元素**：\n\n"
        "| 字段          | 类型    | 必填 | 说明                                 |\n"
        "| ------------- | ------- | ---- | ------------------------------------ |\n"
        "| ``name``      | string  | ✅   | 字段名                               |\n"
        "| ``description`` | string | ✅ | 字段描述（喂给 LLM）                 |\n"
        "| ``type``      | string  | ❌   | ``string`` / ``number`` / ``integer`` / ``boolean`` / ``array`` / ``object`` |\n"
        "| ``required``  | boolean | ❌   | 是否必填                             |\n"
        "| ``items``     | object  | ❌   | 当 ``type=array`` 时的元素 schema    |\n\n"
        "**响应字段（``data``）**：\n\n"
        "| 字段           | 类型    | 说明                                          |\n"
        "| -------------- | ------- | --------------------------------------------- |\n"
        "| ``success``    | boolean | 是否提取成功                                  |\n"
        "| ``url``        | string  | 实际请求的 URL                                |\n"
        "| ``markdown``   | string  | 抓取后清洗的 Markdown（喂给 LLM 的原文）      |\n"
        "| ``data``       | object  | LLM 抽取出的结构化 JSON（任意形状）           |\n"
        "| ``error``      | object  | 失败时的错误描述（成功时为 null）             |\n"
        "| ``elapsed_ms`` | int     | 服务端处理耗时（毫秒）                        |\n"
        "| ``model``      | string  | 实际使用的模型名（成功时回显，便于审计）      |"
    ),
    response_description="成功时返回 LLM 抽取出的结构化数据 ``data.data``。",
    responses={
        200: {
            "description": "成功，返回 LLM 抽取结果。",
            "content": {
                "application/json": {
                    "example": {
                        "code": 200,
                        "msg": "成功",
                        "data": {
                            "success": True,
                            "url": "https://jobs.bytedance.com",
                            "markdown": "## 职位列表\n- 后端工程师 / 北京 ...",
                            "data": {
                                "jobs": [
                                    {"name": "后端工程师", "location": "北京"},
                                    {"name": "算法工程师", "location": "上海"},
                                ]
                            },
                            "error": None,
                            "elapsed_ms": 4567,
                            "model": "qwen3.6-35b-a3b",
                        },
                    }
                }
            },
        },
        401: {"description": "缺少或错误的 Bearer token。"},
        422: {
            "description": (
                "请求体不合法。常见原因：缺 ``url``、缺 ``instruction``、"
                "缺 ``llm`` 字段，或 ``llm.api_key`` / ``base_url`` / ``model`` 之一为空。"
            )
        },
        500: {"description": "服务端内部错误。"},
        503: {"description": "爬虫未初始化（lifespan 失败）。"},
    },
)
async def extract(req: ExtractRequest, request: Request) -> APIResponse[ExtractData]:
    """对单 URL 运行 LLM 智能提取。

    调用方必须在请求体里提供 :class:`src.schema.LLMConfig`（``llm`` 字段），
    否则 Pydantic 会以 ``422 Unprocessable Entity`` 提前拦截。
    """
    started = time.perf_counter()
    logger.info(
        "extract：%s model=%s",
        req.url,
        req.llm.model,
    )

    extractor = build_extractor(
        instruction=req.instruction,
        schema_fields=req.schema_fields,
        llm_config=req.llm,
        extraction_timeout=req.extraction_timeout,
    )

    data = await _get_crawler(request).extract(
        url=str(req.url),
        options=req.options,
        extractor=extractor,
        model=req.llm.model,
    )
    elapsed_ms = int((time.perf_counter() - started) * 1000)
    logger.info(
        "extract done：%s success=%s elapsed=%dms",
        req.url,
        data.success,
        elapsed_ms,
    )
    return ok(data)


def register_exception_handlers(app: FastAPI) -> None:
    """注册全局异常处理器，避免堆栈泄露给客户端。"""

    @app.exception_handler(Exception)
    async def _unhandled(_: Request, exc: Exception) -> JSONResponse:
        logger.exception("未处理异常：%s", exc)
        return JSONResponse(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            content={"code": 500, "msg": "服务器内部错误", "data": None},
        )

    @app.exception_handler(HTTPException)
    async def _http_exc(_: Request, exc: HTTPException) -> JSONResponse:
        """把 :class:`HTTPException` 也包成 APIResponse 形状。"""
        detail = exc.detail
        msg: str
        code: int = exc.status_code
        if isinstance(detail, dict):
            msg = str(detail.get("msg") or detail.get("detail") or "请求错误")
        else:
            msg = str(detail)
        return JSONResponse(
            status_code=exc.status_code,
            content={"code": code, "msg": msg, "data": None},
            headers=exc.headers,
        )
