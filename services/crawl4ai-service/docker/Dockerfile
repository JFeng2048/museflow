# ---- Stage 1: builder ----
FROM astral/uv:python3.13-bookworm-slim AS builder

# VERSION 决定打包哪个版本接口依赖组：http / grpc / both
#   http  —— 只装 FastAPI 版依赖（uv sync --extra http）
#   grpc  —— 只装 gRPC 版依赖（uv sync --extra grpc）
#   both  —— 两个版本都装（默认）
ARG VERSION=both
ENV VERSION=${VERSION}

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    UV_LINK_MODE=copy

WORKDIR /app

COPY pyproject.toml README.md ./
# 需要 proto 契约与生成脚本，用于产出 gRPC 桩代码
COPY proto/crawl ./proto/crawl
COPY scripts ./scripts

# 仅安装目标版本接口依赖组 + 基础依赖（dev 组不装）
RUN if [ "$VERSION" = "http" ]; then \
        uv sync --no-cache --no-dev --extra http; \
    elif [ "$VERSION" = "grpc" ]; then \
        uv sync --no-cache --no-dev --extra grpc; \
    else \
        uv sync --no-cache --no-dev --extra http --extra grpc; \
    fi

# 生成 src/crawl_pb2.py 与 src/crawl_pb2_grpc.py（仅 grpc / both 需要）。
# 纯 http 镜像没有 grpcio-tools，这里放占位空文件，保证 runtime 阶段 COPY 不报错
# （http 版本不会 import grpc_server，占位文件不会被使用）。
RUN if [ "$VERSION" != "http" ]; then \
        bash scripts/gen_proto.sh; \
    else \
        touch /app/src/crawl_pb2.py /app/src/crawl_pb2_grpc.py; \
    fi

ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright
RUN . /app/.venv/bin/activate \
    && python -m playwright install chromium

# ---- Stage 2: runtime ----
FROM astral/uv:python3.13-bookworm-slim AS runtime

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PATH="/app/.venv/bin:${PATH}" \
    PLAYWRIGHT_BROWSERS_PATH=/ms-playwright \
    PYTHONPATH="/app" \
    LOG_LEVEL=INFO

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        fonts-liberation \
        libasound2 \
        libatk-bridge2.0-0 \
        libatk1.0-0 \
        libcairo2 \
        libcups2 \
        libdbus-1-3 \
        libdrm2 \
        libgbm1 \
        libglib2.0-0 \
        libnspr4 \
        libnss3 \
        libpango-1.0-0 \
        libx11-6 \
        libxcb1 \
        libxcomposite1 \
        libxdamage1 \
        libxext6 \
        libxfixes3 \
        libxkbcommon0 \
        libxrandr2 \
        libxshmfence1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/.venv /app/.venv
COPY --from=builder /ms-playwright /ms-playwright

COPY src ./src
COPY main.py .
# 拷贝构建期生成的 gRPC 桩代码（仅 grpc / both 版本需要）
COPY --from=builder /app/src/crawl_pb2.py ./src/crawl_pb2.py
COPY --from=builder /app/src/crawl_pb2_grpc.py ./src/crawl_pb2_grpc.py

RUN useradd --create-home --uid 1000 app \
    && chown -R app:app /app
USER app

# 两个版本共用同一端口（PORT，默认 5003）。
EXPOSE 5003

# 统一启动入口：根据实际安装的依赖自动拉起 HTTP（FastAPI）和 / 或 gRPC
# 版本接口（共用端口 PORT），二者共用同一份爬取核心逻辑。
CMD ["python", "-m", "main"]
