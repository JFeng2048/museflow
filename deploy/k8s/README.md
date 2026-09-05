# MuseFlow Kubernetes 部署

基础服务和业务应用统一使用 Helm Chart。基础服务 Chart 放在 `base/charts/`，
业务应用 Chart 放在 `applications/`；两类服务分别拥有自己的环境覆盖配置。

```text
k8s/
├── applications/             # 业务应用 Helm Charts
│   ├── services/
│   │   ├── api-gateway/
│   │   ├── user-service/
│   │   └── crawl4ai-service/
│   ├── web/
│   │   └── frontend/
│   └── overlays/              # 应用统一覆盖
│       ├── values.yaml
│       ├── example.secrets.yaml
│       └── secrets.yaml       # 不提交
├── base/                      # 基础服务
│   ├── charts/
│   │   ├── postgres/
│   │   ├── ollama/
│   │   ├── redis/
│   │   └── searxng/
│   └── overlays/              # 基础服务统一覆盖
│       ├── values.yaml
│       ├── example.secrets.yaml
│       └── secrets.yaml       # 不提交
├── scripts/
│   ├── deploy.sh
│   ├── deploy.bat
│   └── deploy.ps1
├── README.md
└── .gitignore
```

## 配置约定

基础服务和应用的真实密钥分开管理，分别复制对应示例文件。两个 `secrets.yaml` 都已被忽略，不要提交。

部署脚本不区分 dev、staging 或 production，按部署范围选择 `base`、`app` 或 `all`。
默认范围是 `all`。以下命令默认使用当前 kubeconfig；如果目标集群不在默认配置中，
请在对应命令末尾追加 `--kubeconfig=/path/to/your/kubeconfig`。

## 使用部署脚本

Linux/macOS 使用 Bash 脚本：

```bash
./deploy/k8s/scripts/deploy.sh base
./deploy/k8s/scripts/deploy.sh app --kubeconfig=/path/to/your/kubeconfig
./deploy/k8s/scripts/deploy.sh all --kubeconfig=/path/to/your/kubeconfig
```

Windows 使用 PowerShell：

```powershell
./deploy/k8s/scripts/deploy.ps1 -Scope base
./deploy/k8s/scripts/deploy.ps1 -Scope app -Kubeconfig 'C:\path\to\kubeconfig'
./deploy/k8s/scripts/deploy.ps1 -Scope all -Kubeconfig 'C:\path\to\kubeconfig'
```

Windows CMD 使用批处理脚本：

```bat
deploy\k8s\scripts\deploy.bat base
deploy\k8s\scripts\deploy.bat app --kubeconfig=C:\path\to\kubeconfig
deploy\k8s\scripts\deploy.bat all --kubeconfig=C:\path\to\kubeconfig
```

脚本会检查对应的真实密钥文件：部署 `base` 需要
`base/overlays/secrets.yaml`，部署 `app` 需要
`applications/overlays/secrets.yaml`，部署 `all` 需要两者。

## 手动 Helm 部署

如果不使用上面的部署脚本，也可以直接执行 Helm 命令。先创建对应的真实密钥文件：

```bash
cp deploy/k8s/base/overlays/example.secrets.yaml deploy/k8s/base/overlays/secrets.yaml
cp deploy/k8s/applications/overlays/example.secrets.yaml deploy/k8s/applications/overlays/secrets.yaml

helm upgrade --install postgres deploy/k8s/base/charts/postgres -n museflow --create-namespace -f deploy/k8s/base/overlays/values.yaml -f deploy/k8s/base/overlays/secrets.yaml --kubeconfig=/path/to/your/kubeconfig
helm upgrade --install ollama deploy/k8s/base/charts/ollama -n museflow -f deploy/k8s/base/overlays/values.yaml -f deploy/k8s/base/overlays/secrets.yaml --kubeconfig=/path/to/your/kubeconfig
helm upgrade --install redis deploy/k8s/base/charts/redis -n museflow -f deploy/k8s/base/overlays/values.yaml -f deploy/k8s/base/overlays/secrets.yaml --kubeconfig=/path/to/your/kubeconfig
helm upgrade --install searxng deploy/k8s/base/charts/searxng -n museflow -f deploy/k8s/base/overlays/values.yaml -f deploy/k8s/base/overlays/secrets.yaml --kubeconfig=/path/to/your/kubeconfig
```

业务应用需要单独执行以下命令：

```bash
helm upgrade --install api-gateway deploy/k8s/applications/services/api-gateway -n museflow -f deploy/k8s/applications/overlays/values.yaml -f deploy/k8s/applications/overlays/secrets.yaml --kubeconfig=/path/to/your/kubeconfig
helm upgrade --install user-service deploy/k8s/applications/services/user-service -n museflow -f deploy/k8s/applications/overlays/values.yaml -f deploy/k8s/applications/overlays/secrets.yaml --kubeconfig=/path/to/your/kubeconfig
helm upgrade --install crawl4ai-service deploy/k8s/applications/services/crawl4ai-service -n museflow -f deploy/k8s/applications/overlays/values.yaml -f deploy/k8s/applications/overlays/secrets.yaml --kubeconfig=/path/to/your/kubeconfig
helm upgrade --install web deploy/k8s/applications/web/frontend -n museflow -f deploy/k8s/applications/overlays/values.yaml -f deploy/k8s/applications/overlays/secrets.yaml --kubeconfig=/path/to/your/kubeconfig
```

## 校验

```bash
helm lint deploy/k8s/base/charts/postgres -f deploy/k8s/base/overlays/values.yaml -f deploy/k8s/base/overlays/secrets.yaml --kubeconfig=/path/to/your/kubeconfig
helm lint deploy/k8s/base/charts/ollama -f deploy/k8s/base/overlays/values.yaml -f deploy/k8s/base/overlays/secrets.yaml --kubeconfig=/path/to/your/kubeconfig
helm lint deploy/k8s/base/charts/redis -f deploy/k8s/base/overlays/values.yaml -f deploy/k8s/base/overlays/secrets.yaml --kubeconfig=/path/to/your/kubeconfig
helm lint deploy/k8s/base/charts/searxng -f deploy/k8s/base/overlays/values.yaml -f deploy/k8s/base/overlays/secrets.yaml --kubeconfig=/path/to/your/kubeconfig
helm lint deploy/k8s/applications/services/api-gateway -f deploy/k8s/applications/overlays/values.yaml -f deploy/k8s/applications/overlays/secrets.yaml --kubeconfig=/path/to/your/kubeconfig
helm lint deploy/k8s/applications/services/user-service -f deploy/k8s/applications/overlays/values.yaml -f deploy/k8s/applications/overlays/secrets.yaml --kubeconfig=/path/to/your/kubeconfig
helm lint deploy/k8s/applications/services/crawl4ai-service -f deploy/k8s/applications/overlays/values.yaml -f deploy/k8s/applications/overlays/secrets.yaml --kubeconfig=/path/to/your/kubeconfig
helm lint deploy/k8s/applications/web/frontend -f deploy/k8s/applications/overlays/values.yaml -f deploy/k8s/applications/overlays/secrets.yaml --kubeconfig=/path/to/your/kubeconfig
```

## 查看、更新与回滚

```bash
helm list -n museflow --kubeconfig=/path/to/your/kubeconfig
kubectl get pods,svc,deploy,statefulset,pvc -n museflow --kubeconfig=/path/to/your/kubeconfig
kubectl get events -n museflow --sort-by=.lastTimestamp --kubeconfig=/path/to/your/kubeconfig
```

修改 `base/overlays/values.yaml`、`applications/overlays/values.yaml` 或对应的
`secrets.yaml` 后，重新执行 `base`、`app` 或 `all` 范围的部署脚本。查看历史并回滚：

```bash
helm history postgres -n museflow --kubeconfig=/path/to/your/kubeconfig
helm rollback postgres <REVISION> -n museflow --kubeconfig=/path/to/your/kubeconfig
```

## 卸载和数据

仅卸载 release：

```bash
helm uninstall postgres ollama redis searxng -n museflow --kubeconfig=/path/to/your/kubeconfig
```

StatefulSet 的 PVC 默认保留。删除 `data-postgres-0`、`data-ollama-0` 或
`data-redis-0` 会永久丢失对应数据，执行前必须完成备份。SearXNG 使用临时缓存，
不创建 PVC。

## 注意事项

- `base/charts/` 只放基础服务 Chart，`applications/` 只放业务应用 Chart。
- PostgreSQL 使用 `pgvector/pgvector:pg18` 镜像，并在初始化脚本中创建 pgvector 扩展。
- PostgreSQL 开发配置中 `Service port=15432`、`NodePort=30432`：集群内使用 `postgres:15432`，集群外使用 `节点IP:30432`。
- 如果云集群提供 LoadBalancer，并且要求集群外使用 `15432`，将 `base/overlays/values.yaml` 中 PostgreSQL Service 改为 `type: LoadBalancer`、`port: 15432`，并删除 `nodePort`。
- 也可以通过命令行临时使用 LoadBalancer 暴露 `15432`：

	```bash
	helm upgrade --install postgres deploy/k8s/base/charts/postgres -n museflow --create-namespace -f deploy/k8s/base/overlays/values.yaml -f deploy/k8s/base/overlays/secrets.yaml --set postgres.service.type=LoadBalancer --set postgres.service.port=15432 --set postgres.service.nodePort= --kubeconfig=/path/to/your/kubeconfig
	```

- 如果必须使用裸机节点的 `节点IP:15432`，需要集群管理员把 NodePort 范围调整为包含 `15432`，然后将 `nodePort` 设置为 `15432`；默认 Kubernetes 集群不允许该端口。
- PostgreSQL、Redis 和 Ollama 的密码只通过未提交的 `secrets.yaml` 注入。
- PostgreSQL 的 pgvector 初始化脚本只对新建数据目录生效。
- 开发环境 PostgreSQL 默认通过 NodePort `30432` 暴露，生产环境应限制网络访问。
- 业务 Chart 中的镜像、资源、端口和配置可通过各自 `values.yaml` 或环境覆盖文件修改。
