#!/bin/bash
# 生成 gRPC 代码
#
# 依赖：
#   protoc                                  https://github.com/protocolbuffers/protobuf/releases
#   protoc-gen-go / protoc-gen-go-grpc      通过下面命令安装：
#     go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#     go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
#
# 用法：在仓库根目录执行 bash scripts/gen-proto.sh
set -euo pipefail

# 切换到仓库根目录，保证 --go_out=. 的相对路径稳定
cd "$(dirname "$0")/.."

command -v protoc >/dev/null 2>&1 || { echo "错误：未找到 protoc，请先安装 protobuf 编译器"; exit 1; }
command -v protoc-gen-go >/dev/null 2>&1 || { echo "错误：未找到 protoc-gen-go，请执行 go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"; exit 1; }
command -v protoc-gen-go-grpc >/dev/null 2>&1 || { echo "错误：未找到 protoc-gen-go-grpc，请执行 go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"; exit 1; }

# paths=source_relative 让生成文件与 .proto 同目录，避免按 go_package 创建深层目录
protoc \
  --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/user/*.proto \
  proto/crawl/*.proto

echo "gRPC 代码生成完成：proto/user/*.pb.go, proto/crawl/*.pb.go"
