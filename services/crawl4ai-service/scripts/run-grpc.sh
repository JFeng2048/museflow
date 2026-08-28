#!/usr/bin/env bash
# 仅运行 gRPC 版本接口。
# 使用前需安装 grpc 依赖组：uv sync --extra grpc
# 并生成桩代码：bash scripts/gen_proto.sh
set -euo pipefail
cd "$(dirname "$0")/.."

echo "启动 crawl4ai-service gRPC 版本（端口 PORT=5003）"
exec uv run -m src.grpc_server
