# ============================================================
# web Dockerfile - 前端 SPA 镜像
#
# 多阶段构建：Node 构建静态产物，Nginx 托管并在 k8s 中由 ConfigMap
# 覆盖站点配置（/api 反向代理）。
#
# 构建命令（仓库根目录，构建上下文为仓库根）：
#   docker build -f deploy/k8s/applications/web/frontend/Dockerfile -t museflow/web .
# ============================================================

# ---- 构建阶段 ----
# lts-alpine：跟随当前 Node LTS 版本
FROM node:lts-alpine AS builder

WORKDIR /app

# 复制包管理文件并安装依赖，利用 Docker 层缓存
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile

# 复制源码并构建
COPY web/ ./
RUN pnpm build

# ---- 运行阶段 ----
# stable-alpine：跟随当前 Nginx 稳定分支
FROM nginx:stable-alpine

# 静态产物
COPY --from=builder /app/dist /usr/share/nginx/html

# 站点配置由 k8s ConfigMap 覆盖，镜像内提供一份默认配置便于单独运行
COPY deploy/k8s/applications/web/frontend/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80