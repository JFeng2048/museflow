"""通过 pydantic-settings 加载的应用配置。"""
from __future__ import annotations

from functools import lru_cache

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """从环境变量 / .env 文件加载的服务配置。

    本服务只负责"爬取 + 提取"，LLM 凭证是 **按请求** 传入的（见
    :class:`src.schema.LLMConfig` 与 :class:`ExtractRequest.llm_config`），
    所以这里不再保存任何 LLM_* 环境变量。
    """

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
    )

    # ---- 服务基础信息 ----
    service_name: str = Field(default="crawl4ai-service")
    service_version: str = Field(default="0.1.0")
    host: str = Field(default="0.0.0.0")
    port: int = Field(default=5003, alias="PORT")  # HTTP / gRPC 共用端口

    # ---- 双版本接口（HTTP / gRPC）监听端口 ----
    # 同一个服务的两套接口：FastAPI（HTTP/JSON，带 Swagger）与 gRPC
    # （proto/crawl 契约），二者共用同一份爬取 / 提取核心逻辑，且共用同一个
    # 监听端口 ``PORT``。跑哪个版本取决于打包时装的依赖组
    # （--extra http / --extra grpc），见 :func:`http_available` / :func:`grpc_available`。
    # 注：HTTP 与 gRPC 是不同传输协议，不能在同一端口上同时监听；
    # 因此「统一端口」意味着部署时按依赖组只启用其中一个版本（或先后绑定会冲突）。

    # ---- 爬虫行为 ----
    default_max_concurrent: int = Field(default=5, ge=1, le=50)
    default_timeout: int = Field(default=60, ge=5, le=600)
    max_retries: int = Field(default=3, ge=0, le=10)
    retry_backoff_factor: float = Field(default=2.0, ge=1.0, le=10.0)
    user_agent: str = Field(
        default="Mozilla/5.0 (compatible; JobInsightCrawler/0.1; +https://jobinsight.local)"
    )
    respect_robots_txt: bool = Field(default=True)
    enable_stealth: bool = Field(default=True)
    headless: bool = Field(default=True)

    # ---- 浏览器二进制 ----
    # 留空时使用 Playwright 自带的 Chromium；
    # 主机上 Playwright 暂不支持的 Linux 发行版（如 ubuntu26.04），
    # 可指向系统已安装的 Chrome：
    #   BROWSER_EXECUTABLE_PATH=/usr/bin/google-chrome
    #   BROWSER_TYPE=chrome
    browser_executable_path: str | None = Field(default=None, alias="BROWSER_EXECUTABLE_PATH")
    browser_type: str = Field(default="chromium")

    # ---- 提取策略 ----
    default_chunk_size: int = Field(default=8192)
    default_overlap_rate: float = Field(default=0.1)

    # ---- API Key 认证 ----
    # 留空时禁用认证（仅供本地开发）。
    # 生产环境必须设置，所有 3 个接口都会校验 ``X-API-Key`` 头。
    api_key: str | None = Field(default=None, alias="API_KEY")

    # ---- 日志 ----
    # 日志输出目录（挂载到容器外便于持久化）。
    # 若文件无法写入（例如权限不足），自动降级为只输出到 stderr。
    log_dir: str = Field(default="/app/logs", alias="LOG_DIR")
    log_file: str = Field(default="service.log", alias="LOG_FILE")
    log_level: str = Field(default="INFO", alias="LOG_LEVEL")


@lru_cache(maxsize=1)
def get_settings() -> Settings:
    """返回缓存的配置单例。"""
    return Settings()


def http_available() -> bool:
    """HTTP（FastAPI）版本接口是否可用：取决于打包时是否装了 ``http`` 依赖组。

    通过探测 ``fastapi`` / ``uvicorn`` 是否可导入判断，避免在纯 gRPC 镜像里
    因缺少 fastapi 而启动失败。
    """
    try:
        import fastapi  # noqa: F401
        import uvicorn  # noqa: F401
        return True
    except ImportError:
        return False


def grpc_available() -> bool:
    """gRPC 版本接口是否可用：取决于打包时是否装了 ``grpc`` 依赖组。

    通过探测 ``grpc`` / ``grpcio`` 是否可导入判断，避免在纯 HTTP 镜像里
    因缺少 grpc 而启动失败。
    """
    try:
        import grpc  # noqa: F401
        return True
    except ImportError:
        return False
