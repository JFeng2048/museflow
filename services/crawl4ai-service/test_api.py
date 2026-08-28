"""crawl4ai-service API 测试（覆盖 3 个接口 + 认证 + 响应包装）。"""
from __future__ import annotations

# 让项目根目录（包含 main.py / src/）始终在 ``sys.path`` 中，
# 这样 ``from src.xxx import ...`` 与 ``from main import app`` 都能解析。
import sys
from pathlib import Path

_ROOT = Path(__file__).resolve().parent
if str(_ROOT) not in sys.path:
    sys.path.insert(0, str(_ROOT))

from unittest.mock import patch  # noqa: E402

import pytest  # noqa: E402
from fastapi.testclient import TestClient  # noqa: E402

from src.api import verify_api_key  # noqa: E402
from src.config import Settings, get_settings  # noqa: E402
from src.extractor import (  # noqa: E402
    LLMSmartExtractor,
    build_extractor,
    build_json_schema,
    safe_json_loads,
)
from src.schema import LLMConfig  # noqa: E402

# main.py 位于项目根目录，与 src/ 同级；
# 项目根已通过上方 sys.path 注入，因此 ``from main import app`` 即可解析。
from main import app  # noqa: E402


# ============================================================
# 纯函数 / schema 测试
# ============================================================


def test_safe_json_loads_plain_dict() -> None:
    """普通 JSON 字符串能正确解析。"""
    assert safe_json_loads('{"a": 1, "b": "x"}') == {"a": 1, "b": "x"}


def test_safe_json_loads_strips_markdown_fence() -> None:
    """能去掉 ```json ... ``` 包裹。"""
    raw = '```json\n{"title": "Engineer"}\n```'
    assert safe_json_loads(raw) == {"title": "Engineer"}


def test_safe_json_loads_recovers_partial_json() -> None:
    """当 LLM 输出前后包含杂讯时，仍能截取出 JSON。"""
    raw = '以下是结果：\n{"skills": ["python"]}\n谢谢。'
    assert safe_json_loads(raw) == {"skills": ["python"]}


def test_safe_json_loads_returns_raw_on_failure() -> None:
    """无法解析时返回 ``{"raw": ...}``。"""
    assert safe_json_loads("不是 JSON") == {"raw": "不是 JSON"}


def test_safe_json_loads_unwraps_crawl4ai_success_list() -> None:
    """Crawl4AI 成功的 list 形态 ``[{"error": False, ...}]`` 应解包为单 dict。"""
    raw = [
        {
            "error": False,
            "title": "ExtractionResult",
            "description": "页面主标题",
        }
    ]
    assert safe_json_loads(raw) == {
        "error": False,
        "title": "ExtractionResult",
        "description": "页面主标题",
    }


def test_safe_json_loads_wraps_crawl4ai_multi_item_list() -> None:
    """多条成功结果保留在 ``_items`` 下。"""
    raw = [
        {"error": False, "title": "A"},
        {"error": False, "title": "B"},
    ]
    assert safe_json_loads(raw) == {"_items": raw}


def test_safe_json_loads_wraps_crawl4ai_error_list() -> None:
    """Crawl4AI 失败项 ``[{"error": True, "content": "..."}]`` 走 ``_error_list`` 兜底。"""
    raw = [{"error": True, "index": 0, "content": "boom"}]
    assert safe_json_loads(raw) == {"_error_list": raw}


def test_safe_json_loads_parses_crawl4ai_json_string_then_unwraps() -> None:
    """**根因测试**：Crawl4AI 的 ``extracted_content`` 是 JSON 字符串。

    ``"[{...}]"`` 必须先 ``json.loads`` 出 list，再走单条 dict 解包，
    否则上游 Pydantic 校验 ``ExtractData.data: dict`` 会直接 500。
    """
    raw = '[{"title": "字节跳动招聘", "description": "...", "error": false}]'
    assert safe_json_loads(raw) == {
        "title": "字节跳动招聘",
        "description": "...",
        "error": False,
    }


def test_safe_json_loads_parses_crawl4ai_error_string_to_error_list() -> None:
    """字符串形式的 Crawl4AI 错误列表同样要兜底为 ``_error_list``。"""
    raw = '[{"error": true, "index": 0, "content": "boom"}]'
    assert safe_json_loads(raw) == {
        "_error_list": [{"error": True, "index": 0, "content": "boom"}]
    }


def test_build_json_schema_with_fields() -> None:
    """传入 schema_fields 时生成的 schema 包含每个字段及 required 列表。"""
    from src.schema import ExtractSchema

    fields = [
        ExtractSchema(name="title", description="职位名称", type="string", required=True),
        ExtractSchema(name="salary", description="薪资", type="number"),
    ]
    schema = build_json_schema("提取职位信息", fields)
    assert schema["type"] == "object"
    assert "title" in schema["properties"]
    assert "salary" in schema["properties"]
    assert schema["required"] == ["title"]


def test_build_json_schema_without_fields() -> None:
    """未传 schema_fields 时返回宽松的 schema。"""
    schema = build_json_schema("随便提点什么")
    assert schema["properties"]["data"]["type"] == "object"
    assert schema["required"] == ["data"]


def test_extractor_builds_crawl4ai_llm_config() -> None:
    """build() 必须把 LLMConfig 透传成 :class:`crawl4ai.LLMConfig`，不能是 dict。"""
    from crawl4ai.async_configs import LLMConfig as Crawl4aiLLMConfig

    cfg = LLMConfig(api_key="k", base_url="https://api.example.com/v1", model="gpt-x")
    extractor = build_extractor(
        instruction="提取标题",
        schema_fields=None,
        llm_config=cfg,
        extraction_timeout=120,
    )
    strategy = extractor.build()
    assert isinstance(strategy.llm_config, Crawl4aiLLMConfig)
    assert strategy.llm_config.provider == "openai/gpt-x"
    assert strategy.llm_config.api_token == "k"
    assert strategy.llm_config.base_url == "https://api.example.com/v1"
    assert strategy.llm_config.temperature == 0.0
    assert strategy.llm_config.max_tokens == 2048


# ============================================================
# FastAPI 集成测试
# ============================================================


class _FakeCrawler:
    """不真正启动浏览器的 :class:`CrawlerService` 替身。"""

    def __init__(self) -> None:
        self.started = False
        self.stopped = False

    async def start(self) -> None:
        self.started = True

    async def stop(self) -> None:
        self.stopped = True

    async def crawl(self, url: str, **_kwargs):
        from src.schema import CrawlData

        return CrawlData(
            url=url,
            success=True,
            markdown=f"# 模拟 Markdown - {url}",
            status_code=200,
            elapsed_ms=1,
        )

    async def extract(self, url: str, model: str | None = None, **_kwargs):
        from src.schema import ExtractData

        return ExtractData(
            url=url,
            success=True,
            markdown="# 模拟",
            data={"jobs": [{"name": "Engineer", "location": "Beijing"}]},
            elapsed_ms=1,
            model=model,
        )


@pytest.fixture
def auth_enabled(monkeypatch: pytest.MonkeyPatch):
    """启用 API Key 认证，并把密钥设为 ``test-key``。"""
    monkeypatch.setenv("API_KEY", "test-key")
    get_settings.cache_clear()
    yield
    get_settings.cache_clear()


@pytest.fixture
def auth_disabled(monkeypatch: pytest.MonkeyPatch):
    """显式关闭 API Key 认证。

    设为空字符串 ``""`` 以覆盖项目根目录的 .env（pydantic-settings 中
    环境变量的优先级高于 .env 文件），同时 ``""`` 在 ``verify_api_key``
    里被视为"未配置"。
    """
    monkeypatch.setenv("API_KEY", "")
    get_settings.cache_clear()
    yield
    get_settings.cache_clear()


@pytest.fixture
def client(monkeypatch: pytest.MonkeyPatch, auth_disabled: None):  # noqa: ARG001
    """默认禁用认证的 FastAPI 测试客户端。"""

    async def _noop_start(self):  # noqa: ANN001
        self.started = True

    async def _noop_stop(self):  # noqa: ANN001
        self.stopped = True

    fake = _FakeCrawler()
    monkeypatch.setattr("src.crawler.CrawlerService.start", _noop_start)
    monkeypatch.setattr("src.crawler.CrawlerService.stop", _noop_stop)
    # 关掉 verify_api_key 以便测试默认不需要 header
    app.dependency_overrides[verify_api_key] = lambda: None
    with TestClient(app) as c:
        c.app.state.crawler_service = fake
        yield c
    app.dependency_overrides.clear()


# ---- /health ----

def test_health_endpoint(client: TestClient) -> None:
    """/health 应返回 200 + APIResponse 包装 + auth_enabled 字段。"""
    res = client.get("/health")
    assert res.status_code == 200
    body = res.json()
    assert body["code"] == 200
    assert body["msg"] == "成功"
    assert body["data"]["status"] == "ok"
    assert body["data"]["auth_enabled"] is False
    assert "uptime_seconds" in body["data"]


# ---- /crawl ----

def test_crawl_endpoint_success(client: TestClient) -> None:
    """/crawl 返回 ``APIResponse[CrawlData]``。"""
    res = client.post(
        "/crawl",
        json={"url": "https://example.com", "wait_for": ".job-list"},
    )
    assert res.status_code == 200
    body = res.json()
    assert body["code"] == 200
    assert body["msg"] == "成功"
    assert body["data"]["success"] is True
    # Pydantic ``HttpUrl`` 会把 ``example.com`` 规范化为 ``example.com/``
    assert body["data"]["url"].rstrip("/") == "https://example.com"
    assert "Markdown" in body["data"]["markdown"]


def test_crawl_endpoint_validation_missing_url(client: TestClient) -> None:
    """缺 url 时 Pydantic 应返回 422。"""
    res = client.post("/crawl", json={})
    assert res.status_code == 422


# ---- /extract ----

def test_extract_endpoint_requires_llm_config(client: TestClient) -> None:
    """/extract 缺 ``llm`` 时 Pydantic 应返回 422。"""
    res = client.post(
        "/extract",
        json={
            "url": "https://example.com",
            "instruction": "提取职位信息",
        },
    )
    assert res.status_code == 422
    body = res.json()
    # 缺必填字段 ``llm`` 时，Pydantic v2 的 error 形如
    # ``{"type": "missing", "loc": ["body", "llm"], "msg": "Field required"}``。
    missing = [
        item for item in body.get("detail", [])
        if item.get("type") == "missing"
    ]
    assert any("llm" in (item.get("loc") or []) for item in missing), body


def test_extract_endpoint_passes_llm_config_through(client: TestClient) -> None:
    """``llm`` 字段会原样回显到响应 ``data.model``。"""
    with patch("src.api.build_extractor") as mock_build:
        mock_build.return_value = LLMSmartExtractor(
            llm_config=LLMConfig(api_key="k", base_url="https://x/v1", model="m"),
            instruction="x",
        )
        res = client.post(
            "/extract",
            json={
                "url": "https://example.com",
                "instruction": "提取内容",
                "llm": {
                    "api_key": "k",
                    "base_url": "https://x/v1",
                    "model": "m",
                },
            },
        )
        assert res.status_code == 200, res.text
        body = res.json()
        assert body["data"]["model"] == "m"
        kwargs = mock_build.call_args.kwargs
        assert kwargs["llm_config"].model == "m"
        assert "use_llm" not in kwargs


# ---- 认证 ----

def test_health_requires_api_key_when_configured(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """API_KEY 配置后，缺 / 错 / 对 Bearer token 应分别返回 401 / 401 / 200。"""
    monkeypatch.setenv("API_KEY", "test-key")
    get_settings.cache_clear()

    async def _noop_start(self):  # noqa: ANN001
        self.started = True

    async def _noop_stop(self):  # noqa: ANN001
        self.stopped = True

    monkeypatch.setattr("src.crawler.CrawlerService.start", _noop_start)
    monkeypatch.setattr("src.crawler.CrawlerService.stop", _noop_stop)
    # 清掉默认的 override 让 verify_api_key 真正生效
    app.dependency_overrides.clear()

    try:
        with TestClient(app) as c:
            c.app.state.crawler_service = _FakeCrawler()

            # 缺 header → 401
            res = c.get("/health")
            assert res.status_code == 401
            body = res.json()
            assert body["code"] == 401
            assert "API Key" in body["msg"] or "Authorization" in body["msg"]

            # 错的 token → 401
            res = c.get("/health", headers={"Authorization": "Bearer wrong"})
            assert res.status_code == 401

            # 对的 token → 200
            res = c.get("/health", headers={"Authorization": "Bearer test-key"})
            assert res.status_code == 200
            body = res.json()
            assert body["data"]["auth_enabled"] is True
    finally:
        get_settings.cache_clear()


def test_crawl_requires_api_key_when_configured(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """/crawl 同样走 Bearer 认证。"""
    monkeypatch.setenv("API_KEY", "test-key")
    get_settings.cache_clear()
    try:
        async def _noop_start(self):  # noqa: ANN001
            self.started = True

        async def _noop_stop(self):  # noqa: ANN001
            self.stopped = True

        monkeypatch.setattr("src.crawler.CrawlerService.start", _noop_start)
        monkeypatch.setattr("src.crawler.CrawlerService.stop", _noop_stop)
        app.dependency_overrides.clear()

        with TestClient(app) as c:
            c.app.state.crawler_service = _FakeCrawler()
            res = c.post("/crawl", json={"url": "https://example.com"})
            assert res.status_code == 401

            res = c.post(
                "/crawl",
                json={"url": "https://example.com"},
                headers={"Authorization": "Bearer test-key"},
            )
            assert res.status_code == 200
    finally:
        get_settings.cache_clear()


def test_openapi_exposes_bearer_security_scheme(client: TestClient) -> None:
    """OpenAPI schema 中应暴露 ``HTTPBearer`` 安全方案（Swagger Authorize 按钮的依据）。"""
    schema = client.get("/openapi.json").json()
    schemes = (schema.get("components") or {}).get("securitySchemes") or {}
    assert "HTTPBearer" in schemes
    target = schemes["HTTPBearer"]
    assert target.get("type") == "http"
    assert target.get("scheme", "").lower() == "bearer"


# 保留对 Settings 类的 smoke test，确保 LLM_* 字段已彻底从全局配置移除。
def test_settings_no_longer_exposes_llm_fields() -> None:
    s = Settings()
    assert not hasattr(s, "llm_api_key")
    assert not hasattr(s, "llm_base_url")
    assert not hasattr(s, "llm_model")
    assert not hasattr(s, "llm_configured")
    # 但 api_key 字段已存在
    assert hasattr(s, "api_key")
