# ---- Stage 1: builder ----
FROM astral/uv:python3.13-bookworm-slim AS builder

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    UV_LINK_MODE=copy

WORKDIR /app

COPY pyproject.toml README.md ./

RUN uv sync --no-cache

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

RUN useradd --create-home --uid 1000 app \
    && chown -R app:app /app
USER app

EXPOSE 11235

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "11235"]