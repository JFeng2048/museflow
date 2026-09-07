# ---- 构建阶段 ----
# 由于 user-service 依赖同仓库的 proto 与 pkg 模块，构建上下文需为仓库根目录：
#   服务端: docker build -f services/user-service/Dockerfile -t museflow/user-service .
#   worker: docker build -f services/user-service/Dockerfile --target worker -t museflow/user-service-worker .
FROM golang:1.26-alpine AS builder

WORKDIR /src

# 先复制模块定义并下载依赖，利用 Docker 层缓存
# replace 指向本仓库 pkg/ 模块，需一并复制其 go.mod/go.sum 以解析模块图
COPY pkg/envloader/go.mod ./pkg/envloader/go.mod
COPY pkg/logger/go.mod pkg/logger/go.sum ./pkg/logger/
COPY proto/go.mod proto/go.sum ./proto/
COPY services/user-service/go.mod services/user-service/go.sum ./services/user-service/
RUN cd services/user-service && go mod download

# 复制源码
COPY pkg/ ./pkg/
COPY proto/ ./proto/
COPY services/user-service/ ./services/user-service/

# 静态编译，去除调试信息减小体积
# 服务端与 worker（asynq 消费端，负责邮件发送等慢速任务）共用依赖层，分两步编译
FROM builder AS build-server
RUN cd services/user-service && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/user-service ./cmd/server

FROM builder AS build-worker
RUN cd services/user-service && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/user-service-worker ./cmd/worker

# ---- 运行阶段：asynq worker ----
FROM alpine:3.20 AS worker

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 appuser

WORKDIR /app
COPY --from=build-worker /out/user-service-worker /app/user-service-worker

USER appuser

ENTRYPOINT ["/app/user-service-worker"]

# ---- 运行阶段：gRPC 服务端（最后一个阶段，作为默认构建产物） ----
FROM alpine:3.20

# ca-certificates 用于 TLS，tzdata 保证时间正确
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 appuser

WORKDIR /app
COPY --from=build-server /out/user-service /app/user-service

# 以非 root 用户运行
USER appuser

EXPOSE 5002

ENTRYPOINT ["/app/user-service"]
