# user-service 用户服务

MuseFlow 的用户与认证核心微服务，基于 **gRPC + Go** 实现，监听 `:5002`。负责账号生命周期、双令牌认证、邮箱验证码、两步验证（TOTP）、RBAC 权限、操作审计与 OAuth 关联。

## 技术选型

| 领域 | 选型 | 说明 |
| --- | --- | --- |
| 语言 | Go 1.26 | 通过 `go.work` 多模块工作区组织 |
| 通信 | gRPC + Protocol Buffers | `proto/user/user.proto` 定义契约 |
| Web 框架 | — | 纯 gRPC 服务端，HTTP 由 api-gateway 承载 |
| ORM | GORM | `gorm.io/driver/postgres`，连接池 50/10 |
| 数据库 | PostgreSQL | schema 固定 `user_svc`，DDL 见 `services/user-service/database/user_svc.sql` |
| 缓存 | Redis | refresh token 白/黑名单、邮箱验证码存储 |
| 异步队列 | Asynq（基于 Redis） | 邮件等慢速外部依赖异步化，独立 Worker 并发消费 |
| 认证 | JWT + bcrypt | `token` 子包负责签发/校验，密码 bcrypt 哈希 |
| 日志 | slog + lumberjack | 经 `pkg/logger` 统一封装（JSON 格式 + 滚动） |
| 错误码 | `pkg/errcode` | 统一双语（zh/en）错误码，映射 gRPC status |

## 架构设计

分层单向依赖，无循环：`handler → service → repository`，service 内部再拆 `auth/token/dto/rbac/oauth/audit/task/admin` 子包，`token`/`dto` 不反向依赖；`internal/pkg/*` 为与业务无关的基础设施（邮件、队列），可被 Worker 与 service 复用且不反向依赖上层。

```
cmd/server/main.go          gRPC 服务入口：依赖装配、健康检查、优雅关闭、投递异步任务
cmd/worker/main.go          异步任务 Worker 入口：消费 asynq 队列（邮件发送）
internal/
  config/                    配置加载（USER_ 前缀 + 公共 DB_/REDIS_/JWT_SECRET/LOG_）
  handler/                   gRPC handler，proto ↔ service 转换，错误码映射（含 WatchTask 服务端流）
  pkg/
    email/                   邮件能力：SMTP 客户端 + 内嵌模板（HTML/纯文本）
    queue/                   Asynq 封装：任务定义、生产者客户端、任务状态模型
    turnstile/               Cloudflare Turnstile 人机验证（siteverify 客户端）
  service/
    auth/                    注册/登录/刷新/登出/改密/2FA/邮箱验证码（核心）
    token/                   TokenManager、claims、设备指纹
    dto/                     Device、TokenPair 等跨层结构
    task/                    异步任务进度查询与订阅（供 WatchTask 使用）
    rbac/                    角色与权限，带 Redis 缓存（TTL=Refresh TTL）
    oauth/                   第三方账号关联
    audit/                   操作审计落库
    admin/                   后台管理（用户/角色查询）
  worker/handlers/           asynq 任务处理器（邮件发送，回写任务进度）
  repository/                GORM 数据访问 + Redis 存储（TokenStore/VerifyCodeStore/TaskStore）
  model/                     GORM 实体（User 等），表名固定 `user_svc.users`
proto/user/                  跨服务共享契约（user.pb.go / user_grpc.pb.go）
database/                    PostgreSQL DDL 与迁移脚本
Dockerfile                   gRPC 服务镜像
Dockerfile.worker            Worker 镜像
```

### 关键设计

- **双令牌认证**：access token（默认 1h，响应体返回） + refresh token（默认 30d，HttpOnly Cookie）。refresh 白名单存 Redis，登出即吊销；access 进入黑名单。
- **两步验证（TOTP）**：基于 RFC 6238，开启后登录第一步返回 `mfa_ticket`，第二步用 TOTP 换取令牌；提供一次性恢复码。
- **邮箱验证码**：场景化（register/login/reset_password/change_email），Redis 键前缀隔离 + `SetNX` 重发冷却；注册校验通过后标记 `email_verified`；修改邮箱走 `change_email` 场景。
- **人机验证（Cloudflare Turnstile）**：发送验证码是易被脚本刷的接口，发送前服务端调用 siteverify 核验一次性令牌（`internal/pkg/turnstile`）。校验位于最前置——未通过时不生成验证码、不占重发冷却、不入队，避免机器人把冷却期刷满导致真实用户发不出验证码。未配置密钥时降级为跳过（仅适用于本地开发，启动会有告警）；校验服务故障时 **fail-closed**（拒绝而非放行）。
- **邮件异步化 + 进度可订阅**：SMTP 是慢速外部依赖，放在 gRPC 请求链路内会拖慢接口。现在 `SendVerifyCode` 只生成验证码并把发信任务投递到 asynq 队列（返回 `task_id`），由独立 Worker 进程并发消费；Worker 把 `pending → sending → retrying → success/failed` 写入 Redis 并通过 Pub/Sub 广播，网关经 `WatchTask` 流式 RPC 转 SSE 推送给前端。
- **入队失败的补偿**：投递失败时回滚已写入的验证码与重发冷却锁，避免用户白等一个冷却周期。
- **邮箱免密登录**：`LoginWithCode` 复用双令牌签发，兼容 2FA。
- **RBAC 缓存**：权限查询结果缓存至 Redis，TTL 与 refresh token 一致（7 天），降低库压力。
- **防账号枚举**：密码/验证码登录对未知邮箱返回与错误凭证一致的统一错误。

## 配置

配置经 `pkg/envloader` 分层加载，优先级：系统环境变量 > 服务 `.env` > 仓库根 `.env` > 默认值。详见 `.env.example`。

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `USER_PORT` | `5002` | gRPC 端口 |
| `DB_HOST`/`DB_PORT`/`DB_USER`/`DB_PASSWORD`/`DB_NAME` | — | PostgreSQL 连接（公共） |
| `REDIS_ADDR`/`REDIS_PASSWORD`/`REDIS_DB` | `localhost:6379` | Redis（公共） |
| `JWT_SECRET` | — | JWT 签名密钥（与 gateway 共享，必填） |
| `USER_ACCESS_TTL_SECONDS` | `3600` | access token 有效期 |
| `USER_REFRESH_TTL_SECONDS` | `2592000` | refresh token 有效期 |
| `USER_BCRYPT_COST` | `10` | bcrypt 成本 |
| `USER_MFA_*` | — | 2FA 发行方/偏移/恢复码数量与长度 |
| `USER_CODE_TTL_SECONDS`/`CODE_LENGTH`/`CODE_RESEND_COOLDOWN_SECONDS` | `600`/`6`/`60` | 验证码参数 |
| `USER_TURNSTILE_SECRET` | — | Turnstile **密钥（Secret Key）**，服务端专用；未配置则人机验证降级为跳过 |
| `USER_TURNSTILE_ENDPOINT` | Cloudflare 官方地址 | siteverify 地址 |
| `USER_TURNSTILE_TIMEOUT_SECONDS` | `5` | 单次人机校验超时 |
| `USER_TURNSTILE_ALLOWED_HOSTNAMES` | — | 来源域名白名单（逗号分隔），留空表示不校验 |
| `USER_SMTP_*` | — | 邮件发送（未配置则日志降级） |
| `USER_QUEUE_NAME` | `email` | 异步任务队列名 |
| `USER_QUEUE_MAX_RETRY` | `3` | 任务失败最大重试次数 |
| `USER_QUEUE_TIMEOUT_SECONDS` | `30` | 单个任务执行超时 |
| `USER_QUEUE_STATUS_TTL_SECONDS` | `600` | 任务进度保留时长（客户端可订阅窗口） |
| `USER_WORKER_CONCURRENCY` | `20` | Worker 并发数，决定发信吞吐 |
| `LOG_*` | — | 日志级别/格式/输出（公共） |

## 构建与运行

服务与 Worker 是两个独立进程，需分别启动（本地开发也要开两个终端）：

```bash
# ---- gRPC 服务 ----
cd services/user-service && go build ./cmd/server
go run ./cmd/server

# ---- 异步任务 Worker（不启动则邮件不会真正发出） ----
go build ./cmd/worker
go run ./cmd/worker

# 热重载（需安装 air；Worker 使用独立配置）
air                          # gRPC 服务
air -c .air.worker.toml      # Worker

# Windows 下可直接使用仓库根目录脚本
scripts\run-user.bat         scripts\run-user-worker.bat
scripts\watch-user.bat       scripts\watch-user-worker.bat

# 测试
go test ./...

# 代码检查
go vet ./...
```

容器镜像同样拆成两个（构建上下文为仓库根目录）：

```bash
docker build -f services/user-service/Dockerfile -t museflow/user-service .
docker build -f services/user-service/Dockerfile.worker -t museflow/user-service-worker .
```

数据库 schema 由 `services/user-service/database/user_svc.sql` 维护（**不**走 GORM AutoMigrate）；字段变更见 `services/user-service/database/migrations/`。

## 接口概览（gRPC）

注册 `Register`、登录 `Login`、刷新 `RefreshToken`、登出 `Logout`、改密 `ChangePassword`、资料更新 `UpdateProfile`、双因子 `EnableMFA`/`VerifyMFA`/`DisableMFA`/`GenerateRecoveryCodes`/`RegenerateRecoveryCodes`、邮箱 `SendVerifyCode`/`WatchTask`/`LoginWithCode`/`ChangeEmail`、会话 `ListSessions`/`RevokeSession`、后台 `ListUsers`/`AssignRole` 等，完整契约见 `proto/user/user.proto`。

其中 `WatchTask` 是唯一的**服务端流式** RPC：订阅异步任务进度，供网关转换为 SSE。

## 测试

`internal/service/auth` 为主覆盖包，使用内存 `UserRepository`/`TokenStore`/`VerifyCodeStore` 替身（fake）隔离测试，`oauth` 包另有独立替身；邮件与 Worker 侧同样用替身覆盖：`internal/pkg/email`（模板渲染与 MIME 组装）、`internal/worker/handlers`（状态流转与重试策略）、`internal/service/task`（进度订阅时序）、`internal/pkg/turnstile`（用 `httptest` 模拟 siteverify，覆盖 action/hostname 校验与 fail-closed 行为）。提交前需通过 `go test ./...` 与 `go vet ./...`。
