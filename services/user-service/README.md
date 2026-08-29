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
| 认证 | JWT + bcrypt | `token` 子包负责签发/校验，密码 bcrypt 哈希 |
| 日志 | slog + lumberjack | 经 `pkg/logger` 统一封装（JSON 格式 + 滚动） |
| 错误码 | `pkg/errcode` | 统一双语（zh/en）错误码，映射 gRPC status |

## 架构设计

分层单向依赖，无循环：`handler → service → repository`，service 内部再拆 `auth/token/dto/rbac/oauth/audit/notify/admin` 子包，`token`/`dto` 不反向依赖。

```
cmd/server/main.go          入口：依赖装配、gRPC Server、健康检查、优雅关闭
internal/
  config/                    配置加载（USER_ 前缀 + 公共 DB_/REDIS_/JWT_SECRET/LOG_）
  handler/                   gRPC handler，proto ↔ service 转换，错误码映射
  service/
    auth/                    注册/登录/刷新/登出/改密/2FA/邮箱验证码（核心）
    token/                   TokenManager、claims、设备指纹
    dto/                     Device、TokenPair 等跨层结构
    rbac/                    角色与权限，带 Redis 缓存（TTL=Refresh TTL）
    oauth/                   第三方账号关联
    audit/                   操作审计落库
    notify/                  邮件发送（SMTP，未配置时降级为日志）
    admin/                   后台管理（用户/角色查询）
  repository/                GORM 数据访问 + Redis 存储（TokenStore/VerifyCodeStore）
  model/                     GORM 实体（User 等），表名固定 `user_svc.users`
proto/user/                  跨服务共享契约（user.pb.go / user_grpc.pb.go）
database/                    PostgreSQL DDL 与迁移脚本
```

### 关键设计

- **双令牌认证**：access token（默认 1h，响应体返回） + refresh token（默认 30d，HttpOnly Cookie）。refresh 白名单存 Redis，登出即吊销；access 进入黑名单。
- **两步验证（TOTP）**：基于 RFC 6238，开启后登录第一步返回 `mfa_ticket`，第二步用 TOTP 换取令牌；提供一次性恢复码。
- **邮箱验证码**：场景化（register/login/reset_password/change_email），Redis 键前缀隔离 + `SetNX` 重发冷却；注册校验通过后标记 `email_verified`；修改邮箱走 `change_email` 场景。
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
| `USER_SMTP_*` | — | 邮件发送（未配置则日志降级） |
| `LOG_*` | — | 日志级别/格式/输出（公共） |

## 构建与运行

```bash
# 构建
cd services/user-service && go build ./cmd/server

# 运行（需先就绪 PostgreSQL / Redis 并按根 .env 配置）
go run ./cmd/server

# 热重载（需安装 air）
air

# 测试
go test ./...

# 代码检查
go vet ./...
```

数据库 schema 由 `services/user-service/database/user_svc.sql` 维护（**不**走 GORM AutoMigrate）；字段变更见 `services/user-service/database/migrations/`。

## 接口概览（gRPC）

注册 `Register`、登录 `Login`、刷新 `RefreshToken`、登出 `Logout`、改密 `ChangePassword`、资料更新 `UpdateProfile`、双因子 `EnableMFA`/`VerifyMFA`/`DisableMFA`/`GenerateRecoveryCodes`/`RegenerateRecoveryCodes`、邮箱 `SendVerifyCode`/`LoginWithCode`/`ChangeEmail`、会话 `ListSessions`/`RevokeSession`、后台 `ListUsers`/`AssignRole` 等，完整契约见 `proto/user/user.proto`。

## 测试

`internal/service/auth` 为主覆盖包，使用内存 `UserRepository`/`TokenStore`/`VerifyCodeStore` 替身（fake）隔离测试；`oauth` 包另有独立替身。提交前需通过 `go test ./...` 与 `go vet ./...`。
