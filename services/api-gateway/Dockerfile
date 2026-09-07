# ---- 构建阶段 ----
# 由于 api-gateway 依赖同仓库的 proto 模块，构建上下文需为仓库根目录：
#   docker build -f services/api-gateway/Dockerfile -t museflow/api-gateway .
FROM golang:1.26-alpine AS builder

WORKDIR /src

# 先复制模块定义并下载依赖，利用 Docker 层缓存
# replace 指向本仓库 pkg/ 模块，需一并复制其 go.mod/go.sum 以解析模块图
COPY pkg/envloader/go.mod ./pkg/envloader/go.mod
COPY pkg/errcode/go.mod pkg/errcode/go.sum ./pkg/errcode/
COPY pkg/logger/go.mod pkg/logger/go.sum ./pkg/logger/
COPY proto/go.mod proto/go.sum ./proto/
COPY services/api-gateway/go.mod services/api-gateway/go.sum ./services/api-gateway/
RUN cd services/api-gateway && go mod download

# 复制源码（docs/ 为 swag 生成的 Swagger 文档，需一并复制）
COPY pkg/ ./pkg/
COPY proto/ ./proto/
COPY services/api-gateway/ ./services/api-gateway/

# 静态编译，去除调试信息减小体积
RUN cd services/api-gateway && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api-gateway ./cmd/server

# ---- 运行阶段 ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 appuser

WORKDIR /app
COPY --from=builder /out/api-gateway /app/api-gateway

USER appuser

EXPOSE 5001

ENTRYPOINT ["/app/api-gateway"]
