# postgres-chart

基于 `pgvector/pgvector` 镜像的 PostgreSQL Helm Chart（内置 pgvector 扩展、性能调优配置、初始化脚本与健康检查）。

## 目录结构

```
postgres-chart/
├── Chart.yaml              # Chart 元数据（提交）
├── values.yaml             # 默认配置模板 = .env.example（提交）
├── values-dev.yaml         # 本机开发配置（git 忽略）
├── values-prod.yaml        # 本机生产配置（git 忽略）
├── .helmignore
├── README.md
└── templates/
    ├── _helpers.tpl        # 公共模板函数
    ├── secret.yaml         # 凭据 Secret
    ├── configmap.yaml      # postgresql.conf + 初始化 SQL
    ├── service.yaml        # Service
    ├── statefulset.yaml    # StatefulSet + PVC
    └── NOTES.txt           # 安装后提示
```

## 配置文件约定

| 文件 | 提交? | 用途 |
| --- | --- | --- |
| `values.yaml` | ✅ | 模板/默认值，改配置不写这里 |
| `values-dev.yaml` / `values-prod.yaml` | ❌ | 本机环境配置，改账号密码、端口等写这里 |

## 常用命令（增删改查）

> 以下命令均为完整命令，可直接复制执行。
> **请在仓库根目录下执行**（Chart 使用相对路径）。
> 只需将 `<你的kubeconfig路径>` 替换为你的 kubeconfig 实际路径。

### 增 — 安装（仅首次部署需要）

```powershell
helm upgrade --install postgres deploy\k8s\postgres-chart -n museflow --create-namespace -f deploy\k8s\postgres-chart\values-dev.yaml --kubeconfig <你的kubeconfig路径>
```

### 查 — 查看与验证

```powershell
helm list -n museflow --kubeconfig <你的kubeconfig路径>                                        # 查看 release
helm get values postgres -n museflow --kubeconfig <你的kubeconfig路径>                         # 实际生效配置
kubectl --kubeconfig <你的kubeconfig路径> get svc,statefulset,pvc,pod -n museflow              # 资源状态
kubectl --kubeconfig <你的kubeconfig路径> logs -n museflow postgres-0 --tail=50                 # 数据库日志
kubectl --kubeconfig <你的kubeconfig路径> exec -n museflow postgres-0 -- psql -U jfeng2048 -d pgvector -c "SELECT extversion FROM pg_extension WHERE extname='vector';"   # 验证扩展
psql -h <节点公网IP> -p 30432 -U jfeng2048 -d pgvector                          # 外部连接（需放行 TCP 30432）
```

### 改 — 更新配置

修改 `values-dev.yaml`（账号密码 / 端口 / 资源等）后重新执行：

```powershell
helm upgrade --install postgres deploy\k8s\postgres-chart -n museflow -f deploy\k8s\postgres-chart\values-dev.yaml --kubeconfig <你的kubeconfig路径>
```

回滚到上一版本：

```powershell
helm rollback postgres -n museflow --kubeconfig <你的kubeconfig路径>
```

### 删 — 删除

仅删除 release（**保留数据**）：

```powershell
helm uninstall postgres -n museflow --kubeconfig <你的kubeconfig路径>
```

彻底删除（**含 PVC 数据，不可恢复**）：

```powershell
helm uninstall postgres -n museflow --kubeconfig <你的kubeconfig路径>
kubectl --kubeconfig <你的kubeconfig路径> delete pvc pgdata-postgres-0 -n museflow
```

重置数据库（删数据后重建，如改账号密码）：

```powershell
helm uninstall postgres -n museflow --kubeconfig <你的kubeconfig路径>
kubectl --kubeconfig <你的kubeconfig路径> delete pvc pgdata-postgres-0 -n museflow
helm upgrade --install postgres deploy\k8s\postgres-chart -n museflow --create-namespace -f deploy\k8s\postgres-chart\values-dev.yaml --kubeconfig <你的kubeconfig路径>
```

> 删除顺序不可颠倒：**必须先卸载 release，再删 PVC**，否则 PVC 卡在 `Terminating`（Pod 仍占用卷）。

## 关键配置项（values-dev.yaml）

```yaml
credentials:                 # 数据库账号密码（首次初始化生效）
  user: jfeng2048
  password: jfeng2048@admin
  database: pgvector
service:
  type: NodePort             # ClusterIP=仅集群内 / NodePort=对外暴露
  port: 15432                # 集群内访问端口
  nodePort: 30432            # 对外端口（30000-32767）
persistence:
  size: 5Gi
```

> 已初始化的数据库改 `credentials` 不生效，需按「删 → 重置数据库」重建，或在库内 `ALTER USER ... WITH PASSWORD ...`。

## 注意事项

- 启动后 30s 内为初始化阶段，日志出现 `FATAL: role ... does not exist` 多为探针账号与凭据不一致，检查 `credentials`。
- 卸载不删 PVC，数据保留；清数据请连 PVC 一起删。
- 公网暴露请先修改默认密码。