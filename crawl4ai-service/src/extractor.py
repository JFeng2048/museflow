"""Crawl4AI 提取策略封装。

本服务只提供 :class:`LLMSmartExtractor` 一种提取策略：使用 OpenAI 兼容的 LLM，
按自然语言 *instruction* 和可选 JSON schema 抽取结构化数据。
LLM 凭证（API Key / base URL / model 等）由调用方在
:class:`src.schema.LLMConfig` 中按请求传入，本服务不再持有任何
全局 LLM 配置。
"""
from __future__ import annotations

import json
from typing import Any

from crawl4ai.async_configs import LLMConfig as _Crawl4aiLLMConfig
from crawl4ai.extraction_strategy import LLMExtractionStrategy

from src.schema import ExtractSchema, LLMConfig


def _schema_field_to_json(field: ExtractSchema) -> dict[str, Any]:
    """把 :class:`ExtractSchema` 字段转换成类 JSON Schema 的字典。"""
    schema: dict[str, Any] = {"type": field.type, "description": field.description}
    if field.items is not None:
        schema["items"] = field.items
    return schema


def build_json_schema(
    instruction: str,
    schema_fields: list[ExtractSchema] | None = None,
) -> dict[str, Any]:
    """为 LLM 提取器构建 JSON schema。

    若未提供 schema_fields，则返回一个宽松的 object schema。
    """
    if not schema_fields:
        return {
            "title": "ExtractionResult",
            "description": instruction,
            "type": "object",
            "properties": {"data": {"type": "object", "description": "提取的数据"}},
            "required": ["data"],
        }

    properties = {f.name: _schema_field_to_json(f) for f in schema_fields}
    required = [f.name for f in schema_fields if f.required]
    return {
        "title": "ExtractionResult",
        "description": instruction,
        "type": "object",
        "properties": properties,
        "required": required or list(properties.keys()),
    }


def safe_json_loads(raw: str | Any) -> dict[str, Any]:
    """尽力而为的 JSON 解析，自动处理 Markdown 代码块包裹的情况。

    兼容 Crawl4AI ``LLMExtractionStrategy`` 返回的两种 list 形态：

    1. **成功**：``[{"error": False, "title": "...", "schema_field": ...}, ...]``
       长度通常为 1（单段）或 N（多段 / 列表型 schema）。取出元素透传。
    2. **失败**：``[{"error": True, "index": 0, "content": "<异常文本>"}]``
       整段塞进 ``{"_error_list": [...]}``，避免上游 :class:`ExtractData`
       强类型 ``dict`` 字段校验直接 500。

    注意：Crawl4AI 的 :attr:`CrawlResult.extracted_content` 是 **JSON 字符串**，
    不是 Python 对象。需要先 ``json.loads``，**解析完再走一次 list/dict 形态归一化**。
    """

    def _normalize(value: Any) -> dict[str, Any]:
        """把任意 ``value`` 归一化为 ``dict``，保证上游 Pydantic 字段校验通过。"""
        if isinstance(value, dict):
            return value
        if isinstance(value, list):
            if not value:
                return {}
            # 全是 Crawl4AI 失败项 → 走错误列表兜底
            if all(
                isinstance(item, dict) and item.get("error") is True
                for item in value
            ):
                return {"_error_list": value}
            # 单条结果：拆出来当主数据
            if len(value) == 1:
                item = value[0]
                if isinstance(item, dict):
                    return item
                return {"data": item}
            # 多条结果（多段切分 / 列表型 schema）：保留为 ``_items``
            return {"_items": value}
        if value is None:
            return {}
        return {"data": str(value)}

    if isinstance(raw, (dict, list)):
        return _normalize(raw)
    if raw is None:
        return {}
    if not isinstance(raw, str):
        return {"data": str(raw)}

    text = raw.strip()
    # 去掉 ```json ... ``` 之类的代码块标记。
    if text.startswith("```"):
        text = text.strip("`")
        if text.startswith("json"):
            text = text[4:]
        text = text.strip()
    if text.endswith("```"):
        text = text[:-3].strip()

    try:
        parsed = json.loads(text)
    except json.JSONDecodeError:
        # 尝试从文本中截取第一个 JSON 对象。
        start, end = text.find("{"), text.rfind("}")
        if start != -1 and end > start:
            try:
                parsed = json.loads(text[start : end + 1])
            except json.JSONDecodeError:
                return {"raw": text}
        else:
            return {"raw": text}

    # 解析成功后**再过一遍** list/dict 归一化——避免上游拿到裸 list 触发 500
    return _normalize(parsed)


class LLMSmartExtractor:
    """基于 LLM 的智能提取器（OpenAI 兼容）。

    构造时必须传入 :class:`LLMConfig`，所有 LLM 凭证随请求而来。
    不再支持纯 CSS / XPath 模式（参见 :class:`src.crawler.CrawlerService`）。
    """

    name = "llm_smart"

    def __init__(
        self,
        llm_config: LLMConfig,
        instruction: str,
        schema_fields: list[ExtractSchema] | None = None,
        extraction_timeout: int = 120,
    ) -> None:
        self._llm_config = llm_config
        self._instruction = instruction
        self._schema = build_json_schema(instruction, schema_fields)
        self._timeout = extraction_timeout

    def build(self) -> LLMExtractionStrategy:
        # 注意：本版 Crawl4AI 的 ``LLMExtractionStrategy`` 要求 ``llm_config``
        # 必须是 :class:`crawl4ai.async_configs.LLMConfig` 实例，
        # **不能是裸 dict**——否则在 :meth:`extract` 里访问
        # ``self.llm_config.provider`` 会报
        # ``AttributeError: 'dict' object has no attribute 'provider'``，
        # 线程池把异常收进 ``extracted_content``，最终导致响应 500。
        c4_llm_config = _Crawl4aiLLMConfig(
            provider=f"openai/{self._llm_config.model}",
            api_token=self._llm_config.api_key,
            base_url=self._llm_config.base_url,
            temperature=self._llm_config.temperature,
            max_tokens=self._llm_config.max_tokens,
        )
        return LLMExtractionStrategy(
            llm_config=c4_llm_config,
            schema=self._schema,
            instruction=self._instruction,
            extraction_timeout=self._timeout,
            input_format="markdown",
            apply_chunking=False,
            verbose=False,
        )


def build_extractor(
    instruction: str,
    schema_fields: list[ExtractSchema] | None,
    llm_config: LLMConfig,
    extraction_timeout: int,
) -> LLMSmartExtractor:
    """工厂方法：构造 :class:`LLMSmartExtractor`。

    调用方应在请求体里提供 :class:`LLMConfig`（Pydantic 已保证非空），
    此处不再校验。
    """
    return LLMSmartExtractor(
        llm_config=llm_config,
        instruction=instruction,
        schema_fields=schema_fields,
        extraction_timeout=extraction_timeout,
    )
