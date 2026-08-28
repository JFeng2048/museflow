#!/usr/bin/env bash
# 生成 CrawlService 的 gRPC Python 桩代码。
#
# grpcio-tools 自带 protoc 编译器，无需系统安装 protoc。
# 生成的文件输出到 src/crawl_pb2.py 与 src/crawl_pb2_grpc.py。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
PROTO_DIR="${SERVICE_DIR}/../../proto/crawl"
OUT_DIR="${SERVICE_DIR}/src"

cd "${SERVICE_DIR}"
python -m grpc_tools.protoc \
  -I"${PROTO_DIR}" \
  --python_out="${OUT_DIR}" \
  --grpc_python_out="${OUT_DIR}" \
  "${PROTO_DIR}/crawl.proto"

echo "已生成: ${OUT_DIR}/crawl_pb2.py, ${OUT_DIR}/crawl_pb2_grpc.py"
