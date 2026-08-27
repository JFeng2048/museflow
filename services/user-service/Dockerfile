# ---- 构建阶段 ----
# 由于 user-service 依赖同仓库的 proto 模块，构建上下文需为仓库根目录：
#   docker build -f services/user-service/Dockerfile -t museflow/user-service .
FROM golang:1.23-alpine AS builder

WORKDIR /src

# 先复制模块定义并下载依赖，利用 Docker 层缓存
COPY proto/go.mod proto/go.sum ./proto/
COPY services/user-service/go.mod services/user-service/go.sum ./services/user-service/
RUN cd services/user-service && go mod download

# 复制源码
COPY proto/ ./proto/
COPY services/user-service/ ./services/user-service/

# 静态编译，去除调试信息减小体积
RUN cd services/user-service && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/user-service ./cmd/server

# ---- 运行阶段 ----
FROM alpine:3.20

# ca-certificates 用于 TLS，tzdata 保证时间正确
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 appuser

WORKDIR /app
COPY --from=builder /out/user-service /app/user-service

# 以非 root 用户运行
USER appuser

EXPOSE 5002

ENTRYPOINT ["/app/user-service"]
