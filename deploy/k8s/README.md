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
│   └── deploy.sh
├── README.md
└── .gitignore
```

## 配置约定

基础服务和应用的真实密钥分开管理，分别复制对应示例文件。两个 `secrets.yaml` 都已被忽略，不要提交。

```powershell
Copy-Item deploy/k8s/base/overlays/example.secrets.yaml deploy/k8s/base/overlays/secrets.yaml
Copy-Item deploy/k8s/applications/overlays/example.secrets.yaml deploy/k8s/applications/overlays/secrets.yaml

$baseValues = "deploy/k8s/base/overlays/values.yaml"
$baseSecrets = "deploy/k8s/base/overlays/secrets.yaml"
helm upgrade --install postgres deploy/k8s/base/charts/postgres -n museflow --create-namespace -f $baseValues -f $baseSecrets
helm upgrade --install ollama deploy/k8s/base/charts/ollama -n museflow -f $baseValues -f $baseSecrets
helm upgrade --install redis deploy/k8s/base/charts/redis -n museflow -f $baseValues -f $baseSecrets
helm upgrade --install searxng deploy/k8s/base/charts/searxng -n museflow -f $baseValues -f $baseSecrets
```

业务应用使用统一的 `applications/overlays` 配置：

```powershell
$appValues = "deploy/k8s/applications/overlays/values.yaml"
$appSecrets = "deploy/k8s/applications/overlays/secrets.yaml"
helm upgrade --install api-gateway deploy/k8s/applications/services/api-gateway -n museflow -f $appValues -f $appSecrets
helm upgrade --install user-service deploy/k8s/applications/services/user-service -n museflow -f $appValues -f $appSecrets
helm upgrade --install crawl4ai-service deploy/k8s/applications/services/crawl4ai-service -n museflow -f $appValues -f $appSecrets
helm upgrade --install web deploy/k8s/applications/web/frontend -n museflow -f $appValues -f $appSecrets
```

## 校验

```powershell
helm lint deploy/k8s/base/charts/postgres -f $baseValues -f $baseSecrets
helm lint deploy/k8s/base/charts/ollama -f $baseValues -f $baseSecrets
helm lint deploy/k8s/base/charts/redis -f $baseValues -f $baseSecrets
helm lint deploy/k8s/base/charts/searxng -f $baseValues -f $baseSecrets
helm lint deploy/k8s/applications/services/api-gateway -f $appValues -f $appSecrets
helm lint deploy/k8s/applications/services/user-service -f $appValues -f $appSecrets
helm lint deploy/k8s/applications/services/crawl4ai-service -f $appValues -f $appSecrets
helm lint deploy/k8s/applications/web/frontend -f $appValues -f $appSecrets
```

## 查看、更新与回滚

```powershell
helm list -n museflow
kubectl get pods,svc,deploy,statefulset,pvc -n museflow
kubectl get events -n museflow --sort-by=.lastTimestamp
```

修改 `overlays/<environment>/values.yaml` 或 `secrets.yaml` 后，重新执行对应的
`helm upgrade --install`。查看历史并回滚：

```powershell
helm history postgres -n museflow
helm rollback postgres <REVISION> -n museflow
```

## 卸载和数据

仅卸载 release：

```powershell
helm uninstall postgres ollama redis searxng -n museflow
```

StatefulSet 的 PVC 默认保留。删除 `data-postgres-0`、`data-ollama-0` 或
`data-redis-0` 会永久丢失对应数据，执行前必须完成备份。SearXNG 使用临时缓存，
不创建 PVC。

## 注意事项

- `base/charts/` 只放基础服务 Chart，`applications/` 只放业务应用 Chart。
- PostgreSQL、Redis 和 Ollama 的密码只通过未提交的 `secrets.yaml` 注入。
- PostgreSQL 的 pgvector 初始化脚本只对新建数据目录生效。
- 开发环境 PostgreSQL 默认通过 NodePort `30432` 暴露，生产环境应限制网络访问。
- 业务 Chart 中的镜像、资源、端口和配置可通过各自 `values.yaml` 或环境覆盖文件修改。
