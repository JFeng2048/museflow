---
name: museflow-deploy
description: >-
  This skill should be used when deploying MuseFlow (a Go microservices + Vue 3
  monorepo) to its K3s cluster via Docker images and Helm. It covers building
  service/frontend images from the repo root, pushing them offline to a K3s node
  with the push-image scripts (SSH public-key auth), and rolling them out with
  Helm + kubectl. Trigger on requests like "部署 / 打包 / docker 部署 / deploy
  MuseFlow", or whenever changes under services/* or web/ need to reach the
  cluster. Server IP, SSH account, and kubeconfig path are NEVER assumed—prompt
  the user for any missing values instead of hardcoding them.
---

# MuseFlow 部署（Docker 镜像 + Helm）

## 用途
把本仓库改动后的服务/前端构建成 Docker 镜像，离线推送到 K3s 节点并导入 containerd，再用 Helm 滚动部署到 `museflow` 命名空间。

## 何时使用
- 用户要求「部署 / 打包 / deploy / docker 部署 MuseFlow」。
- `services/*` 或 `web/` 有改动需要上集群。
- 需要重新构建某个镜像（web / user-service / api-gateway / worker / crawl4ai）。

## 前置信息（必须向用户索取，禁止写死）
执行前务必向用户索取以下信息，**不要假定默认值，也不要把它们写进本 skill 文件**：
- **目标 K3s 节点 IP**：推送镜像的 SSH 目标（docker save → scp → ctr import 的节点）。
- **SSH 账号**：如 `root` / `ubuntu`（节点需对 containerd 有权限）。
- **kubeconfig 路径**：用于 `helm` / `kubectl` 的 `--kubeconfig=`。
- **本次要部署的镜像**：根据改动判断（见步骤 1），不确定时直接问用户「这次部署哪些服务？」。

可选（有默认值，仅在用户说明不同时询问）：
- SSH 端口（默认 `22`）、远端镜像目录（默认 `/docker/images`）。

> 安全红线：不要提交 kubeconfig、`applications/overlays/secrets.yaml`（含真实密钥，已被 `.gitignore` 忽略）；不要把这些值写进 skill 文件。

## 步骤

### 1. 判断要构建 / 部署哪些镜像
- `web/` 有任何改动 → `museflow/web:latest`（前端动态菜单等）。
- `services/user-service/` 改动 → `museflow/user-service:latest`；若涉及异步任务，另构建 `museflow/user-service-worker:latest`（同目录 `Dockerfile.worker`）。
- `services/api-gateway/` 改动 → `museflow/api-gateway:latest`。
- `services/crawl4ai-service/` 改动 → `museflow/crawl4ai-service:latest`。
- 不确定时询问用户「这次要部署哪些服务？」。

### 2. 构建镜像（构建上下文必须是仓库根目录）
在仓库根目录执行，例如：
```bash
# 前端
docker build -f deploy/k8s/applications/web/frontend/Dockerfile -t museflow/web:latest .
# 用户服务（gRPC 服务，依赖同仓库 pkg/ proto/，上下文须为根目录）
docker build -f services/user-service/Dockerfile -t museflow/user-service:latest .
# worker
docker build -f services/user-service/Dockerfile.worker -t museflow/user-service-worker:latest .
# 网关
docker build -f services/api-gateway/Dockerfile -t museflow/api-gateway:latest .
# crawl4ai
docker build -f services/crawl4ai-service/docker/Dockerfile -t museflow/crawl4ai-service:latest .
```
注意：后端 Dockerfile 会从根目录 `COPY pkg/`、`proto/`、`services/<svc>/`，web Dockerfile 也会 `COPY web/...`，因此**必须从仓库根目录构建**，不能进入子目录构建。

### 3. 推送镜像到 K3s 节点（离线：docker save → scp → ctr import）
脚本位于 `deploy/k8s/scripts/`，三种等价入口：`push-image.sh`（Linux/macOS）、`push-image.ps1`（PowerShell，承载实际逻辑）、`push-image.bat`（Windows 调 `.ps1`）。

推送前请向用户索取 `<节点IP>` 与 `<账号>`。

**Windows 多镜像可靠写法**（`.bat` 对多个位置参数存在 PowerShell 数组绑定问题，直接调 `.ps1` 并显式传 `-Image`）：
```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File deploy/k8s/scripts/push-image.ps1 -Server <节点IP> -User <账号> -Image museflow/web:latest museflow/user-service:latest
```
**Linux/macOS**：
```bash
./deploy/k8s/scripts/push-image.sh <节点IP> <账号> museflow/web:latest museflow/user-service:latest
```
说明：脚本支持 SSH 公钥免密（已配置则零输入）；未配置会询问是否现场装一次公钥（只需输一次密码，之后永久免密）。同名 tar 会被覆盖。

### 4. 用 Helm 部署并滚动重启
对每个要部署的 release，拼接以下命令（`<kubeconfig>` 向用户索取）：
```bash
helm upgrade --install <release> deploy/k8s/applications/<chart路径> -n museflow \
  -f deploy/k8s/applications/overlays/values.yaml \
  -f deploy/k8s/applications/overlays/secrets.yaml \
  --kubeconfig=<kubeconfig>
kubectl -n museflow rollout restart deployment/<release> --kubeconfig=<kubeconfig>
kubectl -n museflow rollout status deployment/<release> --timeout=120s --kubeconfig=<kubeconfig>
```
release 与 chart 路径对照：

| release            | chart 路径                      |
| ------------------ | ------------------------------- |
| `web`              | `web/frontend`                  |
| `user-service`     | `services/user-service`         |
| `api-gateway`      | `services/api-gateway`          |
| `crawl4ai-service` | `services/crawl4ai-service`     |

> 镜像 tag 为 `latest` 且 `imagePullPolicy=IfNotPresent`：导入新镜像后**必须 rollout restart** 才会生效。

## 注意事项
- 构建上下文一律为仓库根目录。
- 不要提交 kubeconfig 与 `applications/overlays/secrets.yaml`（含真实密钥，已 gitignore）。
- 完整部署说明见仓库 `deploy/k8s/README.md`；本 skill 只覆盖「构建 → 推送 → 部署」的常用链路。
