#!/bin/bash
# 生成 gRPC 代码
#
# 优先使用 protoc（官方推荐）：
#   protoc                                  https://github.com/protocolbuffers/protobuf/releases
#   protoc-gen-go / protoc-gen-go-grpc      通过下面命令安装：
#     go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#     go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
#
# 若环境未安装 protoc（例如 Windows 开发机），自动回退到纯 Go 生成器
# tools/protogen：它用 protoreflect 解析 .proto 后调用同样的
# protoc-gen-go / protoc-gen-go-grpc 插件，产出结果一致。
#
# 用法：在仓库根目录执行 bash scripts/gen-proto.sh
set -euo pipefail

# 切换到仓库根目录，保证 --go_out=. 的相对路径稳定
cd "$(dirname "$0")/.."

PROTO_FILES=(proto/user/*.proto proto/crawl/*.proto)

if command -v protoc >/dev/null 2>&1; then
  command -v protoc-gen-go >/dev/null 2>&1 || { echo "错误：未找到 protoc-gen-go，请执行 go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"; exit 1; }
  command -v protoc-gen-go-grpc >/dev/null 2>&1 || { echo "错误：未找到 protoc-gen-go-grpc，请执行 go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"; exit 1; }

  echo "使用 protoc 生成..."
  # paths=source_relative 让生成文件与 .proto 同目录，避免按 go_package 创建深层目录
  protoc \
    --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    "${PROTO_FILES[@]}"
else
  echo "未检测到 protoc，回退到纯 Go 生成器 tools/protogen ..."
  # tools/protogen 是独立模块（不在 go.work 中），需关闭 workspace 模式
  ( cd tools/protogen && GOWORK=off go run . -I ../.. "${PROTO_FILES[@]/#/}" )
fi

echo "gRPC 代码生成完成：proto/user/*.pb.go, proto/crawl/*.pb.go"
