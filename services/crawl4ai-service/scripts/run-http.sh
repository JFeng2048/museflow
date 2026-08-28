#!/usr/bin/env bash
# 仅运行 HTTP（FastAPI）版本接口。
# 使用前需安装 http 依赖组：uv sync --extra http
set -euo pipefail
cd "$(dirname "$0")/.."

echo "启动 crawl4ai-service HTTP 版本（FastAPI，端口 PORT=5003）"
exec uv run -m src.api
