# ---- 构建阶段 ----
# 构建上下文需为仓库根目录（config-service 依赖同仓库的 proto 与 pkg 模块）：
#   docker build -f services/config-service/Dockerfile -t museflow/config-service .
FROM golang:1.26-alpine AS builder

WORKDIR /src

# 先复制模块定义并下载依赖，利用 Docker 层缓存。
# replace 指向本仓库 pkg/ 模块，需一并复制其 go.mod/go.sum 以解析模块图。
COPY pkg/envloader/go.mod ./pkg/envloader/go.mod
COPY pkg/logger/go.mod pkg/logger/go.sum ./pkg/logger/
COPY proto/go.mod proto/go.sum ./proto/
COPY services/config-service/go.mod services/config-service/go.sum ./services/config-service/
RUN cd services/config-service && go mod download

# 复制源码
COPY pkg/ ./pkg/
COPY proto/ ./proto/
COPY services/config-service/ ./services/config-service/

# 静态编译，去除调试信息减小体积
RUN cd services/config-service && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/config-service ./cmd/server

# ---- 运行阶段 ----
FROM alpine:3.20

# ca-certificates 用于 TLS，tzdata 保证时间正确
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 appuser

WORKDIR /app
COPY --from=builder /out/config-service /app/config-service

# 以非 root 用户运行
USER appuser

EXPOSE 5004

ENTRYPOINT ["/app/config-service"]
