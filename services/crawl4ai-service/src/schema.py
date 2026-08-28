"""crawl4ai-service API 的请求 / 响应 schema。

所有对外响应统一用 :class:`APIResponse` 包装（与主后端
``backed/schemas/response.py`` 的形状一致）：

```
{ "code": 200, "msg": "成功", "data": { ... } }
```
"""
from __future__ import annotations

from typing import Any, Generic, Literal, TypeVar

from pydantic import BaseModel, ConfigDict, Field, HttpUrl

T = TypeVar("T")


# ============================================================
# 统一响应包装
# ============================================================


class APIResponse(BaseModel, Generic[T]):
    """统一 API 响应包装。

    - ``code``：业务响应码，200 表示成功；其它取值与 HTTP 状态码对齐
      （400/401/500/502 ...）。
    - ``msg``：人类可读的提示信息。
    - ``data``：实际业务数据载荷；失败时为 ``None``。
    """

    code: int = Field(default=200, description="业务响应码，200 表示成功")
    msg: str = Field(default="成功", description="提示信息")
    data: T | None = Field(default=None, description="业务数据")

    model_config = ConfigDict(from_attributes=True)


def ok(data: T | None = None, msg: str = "成功") -> APIResponse[T]:
    """构造成功的 :class:`APIResponse`。"""
    return APIResponse[T](code=200, msg=msg, data=data)


def fail(code: int, msg: str) -> APIResponse[None]:
    """构造失败的 :class:`APIResponse`。"""
    return APIResponse[None](code=code, msg=msg, data=None)


# ============================================================
# 通用
# ============================================================


class CrawlerOptions(BaseModel):
    """单次爬虫调用的可选行为。"""

    timeout: int | None = Field(default=None, ge=5, le=600, description="单 URL 超时（秒）")
    user_agent: str | None = Field(default=None, description="覆盖默认 User-Agent")
    bypass_cache: bool = Field(default=False, description="跳过缓存结果")
    remove_overlay_elements: bool = Field(default=False, description="移除弹窗/遮罩元素")
    simulate_user: bool = Field(default=False, description="模拟人类操作行为")
    magic: bool = Field(default=False, description="启用 Crawl4AI 的魔法模式（反爬）")
    locale: str | None = Field(default=None, description="浏览器语言区域，如 zh-CN")


class ErrorInfo(BaseModel):
    """失败时的错误描述。"""

    code: str
    message: str
    retryable: bool = False


# ============================================================
# LLM（OpenAI 兼容）配置 —— 每次请求独立传入
# ============================================================


class LLMConfig(BaseModel):
    """LLM 提取器所需的配置。

    三个核心字段 ``api_key`` / ``base_url`` / ``model`` 必填，
    其余字段均有合理默认值。本服务**不在环境变量中保存**任何 LLM 凭证，
    所有配置由调用方在每次 ``/extract`` 请求里显式传入。
    """

    api_key: str = Field(min_length=1, description="OpenAI 兼容 API Key")
    base_url: str = Field(min_length=1, description="OpenAI 兼容 base URL")
    model: str = Field(min_length=1, description="模型名")
    temperature: float = Field(default=0.0, ge=0.0, le=2.0)
    max_tokens: int = Field(default=2048, ge=64, le=32768)
    request_timeout: int = Field(default=120, ge=10, le=600)

    def to_provider_dict(self) -> dict[str, Any]:
        """转换为 Crawl4AI ``LLMConfig`` 接受的字典。"""
        return {
            "provider": f"openai/{self.model}",
            "api_token": self.api_key,
            "base_url": self.base_url,
            "temperature": self.temperature,
            "max_tokens": self.max_tokens,
        }


# ============================================================
# /crawl —— 单 URL 爬取
# ============================================================


class CrawlRequest(BaseModel):
    """/crawl 请求体（单 URL）。"""

    url: HttpUrl = Field(description="待爬取的目标 URL")
    wait_for: str | None = Field(
        default=None,
        description="爬取前需要等待出现的 CSS 选择器（如 ``.job-list``）",
    )
    options: CrawlerOptions = Field(default_factory=CrawlerOptions)


class CrawlData(BaseModel):
    """/crawl 响应的 ``data`` 载荷。"""

    success: bool
    url: str
    markdown: str | None = None
    status_code: int | None = None
    error: ErrorInfo | None = None
    elapsed_ms: int | None = None


# ============================================================
# /extract —— 单 URL 智能提取
# ============================================================


class ExtractSchema(BaseModel):
    """提取 schema 中单个字段的定义。"""

    name: str = Field(min_length=1)
    description: str = Field(min_length=1)
    type: Literal["string", "number", "integer", "boolean", "array", "object"] = "string"
    required: bool = False
    items: dict[str, Any] | None = None


class ExtractRequest(BaseModel):
    """/extract 请求体（单 URL，基于 LLM 的结构化提取）。

    本接口**始终走 LLM**，调用方必须在请求体中显式传入
    :class:`LLMConfig`（``llm`` 字段）；不再提供 ``use_llm`` 开关。
    """

    url: HttpUrl = Field(description="待提取的目标 URL")
    instruction: str = Field(
        min_length=1,
        max_length=2000,
        description="提取规则的自然语言描述（用于驱动 LLM 提取）",
    )
    schema_fields: list[ExtractSchema] | None = Field(
        default=None,
        description="可选：JSON Schema 字段定义，用于约束 LLM 输出的字段与类型",
    )
    llm: LLMConfig = Field(
        description="LLM 配置（OpenAI 兼容）：api_key / base_url / model 等必填",
    )
    options: CrawlerOptions = Field(default_factory=CrawlerOptions)
    extraction_timeout: int = Field(default=120, ge=10, le=600)


class ExtractData(BaseModel):
    """/extract 响应的 ``data`` 载荷。

    ``data`` 字段是 LLM / CSS 提取器返回的任意 JSON 对象；
    若提取失败则 ``data`` 为 ``None`` 并填充 ``error``。
    """

    success: bool
    url: str
    markdown: str | None = None
    data: dict[str, Any] | None = None
    error: ErrorInfo | None = None
    elapsed_ms: int | None = None
    model: str | None = None


# ============================================================
# /health
# ============================================================


class HealthData(BaseModel):
    """/health 响应的 ``data`` 载荷。"""

    status: Literal["ok", "degraded", "down"] = "ok"
    service: str
    version: str
    uptime_seconds: float
    auth_enabled: bool = Field(
        description="是否启用了 API Key 认证（``API_KEY`` 是否已配置）"
    )