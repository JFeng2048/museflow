# MuseFlow Kubernetes 部署

`deploy/k8s/` 是 MuseFlow 在 K3s 上的部署配置：基础服务与应用统一用 Helm Chart 管理，
对外入口分「集群外边缘 Nginx（终止 HTTPS）」与「集群内 Traefik Ingress（按域名/路径路由）」两层。

架构分层、请求链路与 K8s 组件说明见 `docs/cn/architecture/部署架构.md`；本文只讲**怎么部署和运维**。

```text
deploy/k8s/
├── applications/              # 业务应用 Helm Charts
│   ├── services/
│   │   ├── api-gateway/
│   │   ├── user-service/
│   │   └── crawl4ai-service/
│   ├── web/
│   │   └── frontend/
│   └── overlays/              # 应用统一覆盖
│       ├── values.yaml
│       ├── example.secrets.yaml
│       └── secrets.yaml       # 不提交（含真实密钥与对外域名）
├── base/                      # 基础服务 Helm Charts
│   ├── charts/
│   │   ├── postgres/
│   │   ├── ollama/
│   │   ├── redis/
│   │   └── searxng/
│   └── overlays/              # 基础服务统一覆盖
│       ├── values.yaml
│       ├── example.secrets.yaml
│       └── secrets.yaml       # 不提交
├── edge/                      # 对外入口层
│   ├── ingress/               # 集群内路由：Traefik Ingress（Helm Chart）
│   └── nginx/                 # 集群外边缘 Nginx：HTTPS 终止 + 反代 Traefik
│       └── museflow.com.conf
├── scripts/                   # deploy.* 部署脚本、push-image.* 离线镜像导入脚本
├── README.md
└── .gitignore
```

## 前置条件

| 依赖 | 说明 |
| --- | --- |
| K3s 集群 | 含 Traefik Ingress Controller，`IngressClass` 名为 `traefik`，NodePort `31380`(HTTP)/`31443`(HTTPS) |
| `helm` / `kubectl` | 部署与排查；命令里用 `--kubeconfig` 指定目标集群 |
| `docker` | 构建业务镜像；离线导入镜像时用 `docker save` |
| SSH 到节点 | 仅用于离线导入镜像（`push-image.*`）与部署边缘 Nginx 配置 |

镜像构建方式见 `docs/cn/develop/镜像构建指南.md`；节点拉不到镜像时用本文的「离线镜像导入」。

## 配置约定

- **密钥不提交**：`base/overlays/secrets.yaml` 与 `applications/overlays/secrets.yaml` 都被 `.gitignore` 忽略，部署前从对应 `example.secrets.yaml` 复制并填写真实值。
- **域名只配一次**：对外域名（`domain`）集中放在 `applications/overlays/secrets.yaml`，Ingress host、网关 CORS 白名单、Turnstile 允许的 hostname 全部由 Chart 模板派生（见「集群内路由与域名」）。
- **覆盖文件分层**：Chart 自带 `values.yaml` 是默认值，两组 `overlays/` 是环境覆盖，部署时必须同时传 `-f overlays/values.yaml -f overlays/secrets.yaml`。
- **部署范围**：部署脚本不区分 dev/staging/production，按范围选择 `base`（数据库、Redis、Ollama、SearXNG）、`app`（业务应用 + `edge/ingress`）或 `all`，默认 `all`。

## 部署

### 使用部署脚本

```bash
# Linux/macOS
./deploy/k8s/scripts/deploy.sh base
./deploy/k8s/scripts/deploy.sh app --kubeconfig=/path/to/kubeconfig
./deploy/k8s/scripts/deploy.sh all --kubeconfig=/path/to/kubeconfig
```

```powershell
# Windows PowerShell
./deploy/k8s/scripts/deploy.ps1 -Scope base
./deploy/k8s/scripts/deploy.ps1 -Scope app -Kubeconfig 'C:\path\to\kubeconfig'
```

```bat
:: Windows CMD（路径用反斜杠；--kubeconfig= 与 --kubeconfig 空格两种写法都支持）
deploy\k8s\scripts\deploy.bat base
deploy\k8s\scripts\deploy.bat app --kubeconfig=C:\path\to\kubeconfig
deploy\k8s\scripts\deploy.bat all --kubeconfig=C:\path\to\kubeconfig
```

脚本会先校验对应范围的密钥文件是否存在：`base` 需要 `base/overlays/secrets.yaml`，
`app` 需要 `applications/overlays/secrets.yaml`，`all` 需要两者。

`app` 范围覆盖 5 个 release：`api-gateway`、`user-service`、`crawl4ai-service`、`web`、`ingress`。
只改其中一个服务时，建议用下面的单 release 命令，避免顺带部署不需要的服务。

### 手动 Helm（按需单个 release）

```bash
# 首次部署先建命名空间
helm upgrade --install postgres deploy/k8s/base/charts/postgres -n museflow --create-namespace \
  -f deploy/k8s/base/overlays/values.yaml -f deploy/k8s/base/overlays/secrets.yaml \
  --kubeconfig=/path/to/kubeconfig

# 其余基础服务
helm upgrade --install ollama   deploy/k8s/base/charts/ollama   -n museflow -f deploy/k8s/base/overlays/values.yaml -f deploy/k8s/base/overlays/secrets.yaml --kubeconfig=/path/to/kubeconfig
helm upgrade --install redis    deploy/k8s/base/charts/redis    -n museflow -f deploy/k8s/base/overlays/values.yaml -f deploy/k8s/base/overlays/secrets.yaml --kubeconfig=/path/to/kubeconfig
helm upgrade --install searxng  deploy/k8s/base/charts/searxng  -n museflow -f deploy/k8s/base/overlays/values.yaml -f deploy/k8s/base/overlays/secrets.yaml --kubeconfig=/path/to/kubeconfig

# 业务应用
helm upgrade --install api-gateway       deploy/k8s/applications/services/api-gateway       -n museflow -f deploy/k8s/applications/overlays/values.yaml -f deploy/k8s/applications/overlays/secrets.yaml --kubeconfig=/path/to/kubeconfig
helm upgrade --install user-service      deploy/k8s/applications/services/user-service      -n museflow -f deploy/k8s/applications/overlays/values.yaml -f deploy/k8s/applications/overlays/secrets.yaml --kubeconfig=/path/to/kubeconfig
helm upgrade --install crawl4ai-service  deploy/k8s/applications/services/crawl4ai-service  -n museflow -f deploy/k8s/applications/overlays/values.yaml -f deploy/k8s/applications/overlays/secrets.yaml --kubeconfig=/path/to/kubeconfig
helm upgrade --install web               deploy/k8s/applications/web/frontend               -n museflow -f deploy/k8s/applications/overlays/values.yaml -f deploy/k8s/applications/overlays/secrets.yaml --kubeconfig=/path/to/kubeconfig

# 集群内路由入口（域名来自 applications/overlays/secrets.yaml 的 domain，必须带上该文件）
helm upgrade --install ingress deploy/k8s/edge/ingress -n museflow \
  -f deploy/k8s/applications/overlays/values.yaml -f deploy/k8s/applications/overlays/secrets.yaml \
  --kubeconfig=/path/to/kubeconfig
```

## 集群内路由与域名

路由规则声明在 `edge/ingress`，由 Traefik 执行：

```text
浏览器 --HTTPS--> 边缘 Nginx（证书） --HTTP--> 节点IP:31380（Traefik）
  /        -> web:80            前端静态产物，frontend 自身不做任何反向代理
  /api     -> api-gateway:5001  /api/v1/... 原样透传
  /health  -> api-gateway:5001  网关健康检查，不需要时可从 paths 中移除
```

Traefik 按最长前缀优先匹配，`/api` 不会被 `/` 覆盖。K3s 自带 `traefik` 这个 IngressClass，
Ingress 使用 `ingressClassName: traefik`；由于 HTTPS 已在边缘终止，Ingress 默认不声明 `tls`
（改由 cert-manager 在集群内签发时，设置 `ingress.tls.enabled=true` 与 `ingress.tls.secretName`）。

对外域名只在 `applications/overlays/secrets.yaml` 配置一次（示例域名 `museflow.com`）：

```yaml
domain:
  host: museflow.com
  scheme: https   # 浏览器实际访问协议，由边缘 Nginx 决定
```

| 派生目标 | 派生规则 |
| --- | --- |
| Ingress `rules[].host` | `domain.host` |
| `GATEWAY_ALLOW_ORIGINS` | `<domain.scheme>://<domain.host>`，附加 `apiGateway.extraAllowOrigins` |
| `USER_TURNSTILE_ALLOWED_HOSTNAMES` | `domain.host`，附加 `userService.extraTurnstileHostnames` |

`domain.host` 缺失时 `helm template` / `helm lint` 会直接报错，不会静默生成空域名。

## 边缘 Nginx 与证书

HTTPS 在集群外终止，配置在 `deploy/k8s/edge/nginx/museflow.com.conf`：80 端口只放行 ACME
校验并 301 跳转到 HTTPS，配置证书路径、真实 IP 透传、AI 接口长超时与 `502/503/504` 错误页；
只做整体反代，不写任何路径分流。文件里的 `museflow.com` 是**示例域名**，部署时替换为
`domain.host` 的真实值（`server_name`、证书目录名、签发命令一起替换）。

| 仓库路径 | 服务器路径 |
| --- | --- |
| `deploy/k8s/edge/nginx/museflow.com.conf` | `/etc/nginx/conf.d/museflow.com.conf` |
| — | `/etc/nginx/certs/museflow.com/fullchain.pem` |
| — | `/etc/nginx/certs/museflow.com/privkey.pem` |

```bash
# 签发证书（certbot webroot）。DNS-01 或 standalone 模式可删掉配置里 80 端口的
# /.well-known/acme-challenge/ 段。下面沿用示例域名，按需替换。
sudo mkdir -p /var/www/acme
sudo certbot certonly --webroot -w /var/www/acme -d museflow.com
sudo mkdir -p /etc/nginx/certs/museflow.com
sudo ln -sf /etc/letsencrypt/live/museflow.com/fullchain.pem /etc/nginx/certs/museflow.com/fullchain.pem
sudo ln -sf /etc/letsencrypt/live/museflow.com/privkey.pem  /etc/nginx/certs/museflow.com/privkey.pem

# 校验并重载
sudo nginx -t && sudo nginx -s reload
```

> 边缘 Nginx 与 K3s 在同一台机器时，把 upstream 改成 `127.0.0.1:31380`，避免绕自己的内网地址。

## 离线镜像导入

节点拉不到镜像（无外网或没有私有仓库）时，用 `scripts/push-image.*` 在本机导出镜像并送到节点：

```bat
:: Windows（CMD 中路径必须用反斜杠；./deploy/... 这种写法只在 PowerShell / bash 里有效）
deploy\k8s\scripts\push-image.bat 192.168.142.121 root museflow/web:latest
deploy\k8s\scripts\push-image.bat 192.168.142.121 root museflow/web:latest museflow/api-gateway:latest --dir=/docker/images
```

```bash
# Linux/macOS（提示权限不足时先 chmod +x deploy/k8s/scripts/push-image.sh，或用 bash 直接执行）
./deploy/k8s/scripts/push-image.sh 192.168.142.121 root museflow/web:latest
./deploy/k8s/scripts/push-image.sh 192.168.142.121 root museflow/web:latest --sudo --dir=/docker/images
```

三个脚本等价（bat 只做参数转换后调 ps1），参数一致：

| 参数 | 说明 |
| --- | --- |
| `<服务器IP>` | K3s 节点地址 |
| `<账号>` | SSH 账号，需能写目标目录、且对 containerd 有权限 |
| `<镜像>` | 一个或多个镜像名，如 `museflow/web:latest` |
| `--port` | SSH 端口，默认 22 |
| `--dir` | 服务器上存放 tar 的目录，默认 `/docker/images` |
| `--sudo` / `--no-sudo` | 强制/禁用远程 sudo；默认自动判断（root 不用，免密 sudo 用） |
| `--skip-key-setup` | 不询问安装公钥，直接用密码认证 |

执行流程与提示：

1. **预检**：连接服务器、`mkdir -p` 目标目录、报告磁盘剩余、自动识别 containerd socket（K3s 优先 `/run/k3s/containerd/containerd.sock`）、报告导入将使用的命令（`k3s ctr` 或 `ctr -a <socket>`）、提权方式，并列出已存在的 tar（**同名文件将被覆盖**）。
2. **导出**：`docker save` 到本地 `tmp/images/`（已 gitignore），打印文件大小与耗时。
3. **上传**：`scp` 覆盖到 `/docker/images/<镜像名>.tar`；文件名规则为把 `/` 与 `:` 换成 `-`，例如 `museflow/web:latest` → `museflow-web-latest.tar`。
4. **导入**：远端先比对文件字节数，再执行 `k3s ctr -n k8s.io images import`（无 `k3s` 时用 `ctr -a <socket>`），最后回查镜像列表确认。
5. **汇总**：列出本次导入的镜像，并提示重启工作负载。

密码处理：不作为参数传递，由 `ssh`/`scp` 提示输入，输入时**不回显、不落盘**，也不写入任何凭据文件。
认证顺序（三个脚本一致）：

1. 已有 SSH 公钥 → 直接免密，零次输入；
2. 没有公钥 → 询问「是否现场配置公钥」，同意后**只输入一次密码，之后永久免密**（幂等追加到服务器 `~/.ssh/authorized_keys`）；
3. 拒绝或配置失败 → 退回密码认证：Linux/macOS 通过 SSH 连接复用，整轮只输入一次（装有 `sshpass` 则完全不进入交互）；Windows 内置 OpenSSH 不支持连接复用，预检/上传/导入会各提示一次。

## 校验

```bash
helm lint deploy/k8s/base/charts/postgres        -f deploy/k8s/base/overlays/values.yaml         -f deploy/k8s/base/overlays/secrets.yaml         --kubeconfig=/path/to/kubeconfig
helm lint deploy/k8s/base/charts/ollama          -f deploy/k8s/base/overlays/values.yaml         -f deploy/k8s/base/overlays/secrets.yaml         --kubeconfig=/path/to/kubeconfig
helm lint deploy/k8s/base/charts/redis           -f deploy/k8s/base/overlays/values.yaml         -f deploy/k8s/base/overlays/secrets.yaml         --kubeconfig=/path/to/kubeconfig
helm lint deploy/k8s/base/charts/searxng         -f deploy/k8s/base/overlays/values.yaml         -f deploy/k8s/base/overlays/secrets.yaml         --kubeconfig=/path/to/kubeconfig
helm lint deploy/k8s/applications/services/api-gateway      -f deploy/k8s/applications/overlays/values.yaml -f deploy/k8s/applications/overlays/secrets.yaml --kubeconfig=/path/to/kubeconfig
helm lint deploy/k8s/applications/services/user-service     -f deploy/k8s/applications/overlays/values.yaml -f deploy/k8s/applications/overlays/secrets.yaml --kubeconfig=/path/to/kubeconfig
helm lint deploy/k8s/applications/services/crawl4ai-service -f deploy/k8s/applications/overlays/values.yaml -f deploy/k8s/applications/overlays/secrets.yaml --kubeconfig=/path/to/kubeconfig
helm lint deploy/k8s/applications/web/frontend              -f deploy/k8s/applications/overlays/values.yaml -f deploy/k8s/applications/overlays/secrets.yaml --kubeconfig=/path/to/kubeconfig
helm lint deploy/k8s/edge/ingress                           -f deploy/k8s/applications/overlays/values.yaml -f deploy/k8s/applications/overlays/secrets.yaml --kubeconfig=/path/to/kubeconfig
```

## 日常运维

```bash
# 查看
helm list -n museflow --kubeconfig=/path/to/kubeconfig
kubectl get pods,svc,deploy,statefulset,pvc,ingress -n museflow --kubeconfig=/path/to/kubeconfig
kubectl get endpoints web api-gateway -n museflow --kubeconfig=/path/to/kubeconfig   # Ingress 后端是否有 IP
kubectl get events -n museflow --sort-by=.lastTimestamp --kubeconfig=/path/to/kubeconfig

# 验证路由（带 Host 头直接打到 Traefik NodePort）
curl -I -H "Host: <域名>" http://<节点IP>:31380/
curl    -H "Host: <域名>" http://<节点IP>:31380/health
```

**改完配置要重启工作负载**：Chart 的 Deployment 没有 ConfigMap 校验注解，`helm upgrade`
只更新 ConfigMap/Secret，不会滚动重启 Pod；镜像 tag 是 `latest` 且策略为 `IfNotPresent`，
导入新镜像后同样需要重启才会生效。

```bash
kubectl -n museflow rollout restart deployment/web --kubeconfig=/path/to/kubeconfig
kubectl -n museflow rollout status  deployment/web --kubeconfig=/path/to/kubeconfig
```

修改 `base/overlays/values.yaml`、`applications/overlays/values.yaml` 或对应 `secrets.yaml` 后，
重新执行 `base`、`app` 或 `all` 范围的部署脚本。查看历史并回滚：

```bash
helm history postgres -n museflow --kubeconfig=/path/to/kubeconfig
helm rollback postgres <REVISION> -n museflow --kubeconfig=/path/to/kubeconfig
```

常见故障（502/404/Cookie 丢失/SSE 不实时）的排查顺序见 `docs/cn/architecture/部署架构.md` 的「排查」一节。

## 卸载与数据

```bash
helm uninstall postgres ollama redis searxng -n museflow --kubeconfig=/path/to/kubeconfig
```

StatefulSet 的 PVC 默认保留。删除 `data-postgres-0`、`data-ollama-0` 或 `data-redis-0`
会永久丢失对应数据，执行前必须完成备份。SearXNG 使用临时缓存，不创建 PVC。

## 注意事项

- **边界**：`base/charts/` 只放基础服务 Chart，`applications/` 只放业务应用 Chart；`edge/ingress` 是集群内资源（Helm 管理，随 `app` 范围部署），`edge/nginx` 是服务器上的 Nginx 配置（手工部署，不参与 Helm）。
- **对外可达范围**：只有 `web` 与 `api-gateway` 经 Traefik Ingress 暴露；`user-service` 仅集群内可达，`frontend` 不做任何反向代理（`/api` 由 Ingress 交给网关）。
- **域名与 Cookie**：`GATEWAY_COOKIE_SECURE=true` 依赖边缘入口的 HTTPS；临时直连 NodePort（HTTP）调试时浏览器不会回传 Cookie，需要临时改回 `false`。
- **数据库/Redis 暴露方式**：Chart 默认都是 ClusterIP（`postgres:5432`、`redis:6379`）；`base/overlays/values.yaml` 把 PostgreSQL 改成了 NodePort（Service 端口 `15432` → 节点端口 `30432`），方便本地调试。Service 类型与端口在 `base/overlays/values.yaml` 或各 Chart 的 `values.yaml` 中调整，生产环境建议保持集群内可达；集群内始终用 `postgres:15432`、`redis:6379` 访问。
  - 临时改用 LoadBalancer 暴露 15432：`helm upgrade --install postgres ... --set postgres.service.type=LoadBalancer --set postgres.service.port=15432 --set postgres.service.nodePort=`
  - 若要用 `节点IP:15432` 这类低于 30000 的端口做 NodePort，需要集群管理员调整 NodePort 范围。
- **外部依赖**：`DB_HOST`/`DB_PORT`/`REDIS_ADDR` 支持集群内服务名、集群外 IP 或域名，走集群外时需确保网络路由、防火墙与访问白名单已放行。PostgreSQL、Redis、Ollama 的密码只通过未提交的 `secrets.yaml` 注入。
- **PostgreSQL**：使用 `pgvector/pgvector:pg18` 镜像，初始化脚本创建 pgvector 扩展，且**只对新建数据目录生效**。
- **脚本文件格式**：`scripts/*.bat` 必须保持「纯 ASCII + CRLF」——CMD 按本地代码页读取批处理文件，UTF-8 中文注释会让行解析错乱（把注释碎片当命令执行），不只是显示乱码；中文提示统一由 `.ps1` 输出。`scripts/*.sh` 保持 LF。
- **可调项**：业务 Chart 的镜像、资源、端口与配置，都可通过各自 `values.yaml` 或 `overlays/` 覆盖文件调整。
