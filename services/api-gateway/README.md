# api-gateway API 网关

MuseFlow 的统一 HTTP 入口网关，基于 **Gin + Go** 实现，监听 `:5001`。对外暴露 RESTful JSON 接口并转发到后端 gRPC 微服务（当前对接 user-service），内聚认证、CORS、访问日志与请求链路标识。

## 技术选型

| 领域 | 选型 | 说明 |
| --- | --- | --- |
| 语言 | Go 1.26 | `go.work` 多模块工作区 |
| HTTP 框架 | Gin | 路由、参数校验（binding）、中间件 |
| 上游通信 | gRPC | `google.golang.org/grpc` 客户端转发到 user-service |
| 文档 | Swagger (swaggo) | `swag init` 生成，UI 在 `/swagger/index.html` |
| 日志 | `pkg/logger`（slog + lumberjack） | JSON 格式、滚动切割 |
| 错误码 | `pkg/errcode` | 统一双语响应结构 `Response{code,message,data}` |
| 配置 | `pkg/envloader` | 分层加载，`GATEWAY_` 前缀 + 公共键 |

## 架构设计

```
cmd/server/main.go        入口：加载配置、日志、建立 user-service gRPC 客户端、启动 HTTP Server、优雅关闭
internal/
  config/                  配置（GATEWAY_ 前缀 + JWT_SECRET / LOG_ 公共键）
  client/                  user-service gRPC 客户端封装（惰性连接）
  router/                  路由注册与中间件装配
    v1/                    按业务域拆分路由（auth / user ...）
  middleware/              CORS、JWT 认证、访问日志、请求 ID
  handler/                 HTTP handler：dto ↔ proto 转换、gRPC 错误映射、Cookie 写入
  dto/                     请求/响应结构体（含 Swagger 注解）
proto/user/                与 user-service 共享的契约
```

### 关键设计

- **双令牌透传**：access token 经响应体返回、放入 `Authorization: Bearer`；refresh token 写入 HttpOnly Cookie（受 `CookieSecure`/`SameSite`/`Domain` 控制）。登录/刷新/登出由网关管理 Cookie。
- **认证中间件**：校验 access token 签名（与 user-service 共享 `JWT_SECRET`）；`/auth/*` 下仅登录、注册、重置、邮箱验证码等免认证，其余需携带有效 Bearer。
- **2FA 兼容**：登录返回 `mfa_ticket` 时网关不下发令牌，仅回传票据，前端走 `/auth/mfa/verify-login` 换取。
- **统一错误映射**：gRPC status 经 `errcode` 转换为 HTTP 状态码与双语消息（`writeGRPCError`）。
- **请求链路**：每个请求生成 request-id 并注入日志上下文，便于跨服务追踪。

## 接口概览（HTTP，前缀 `/api/v1`）

| 方法 | 路径 | 说明 | 认证 |
| --- | --- | --- | --- |
| POST | `/auth/register` | 注册（需邮箱验证码） | 否 |
| POST | `/auth/login` | 密码登录（可能触发 2FA） | 否 |
| POST | `/auth/logout` | 登出并吊销令牌 | 否 |
| POST | `/auth/mfa/*` | 2FA 启用/校验/恢复码 | 是 |
| POST | `/auth/password/reset` | 验证码重置密码（码经 `/common/email/send-code` scene=reset_password 获取） | 否 |
| POST | `/auth/email/send-code` | 发送邮箱验证码（register/login/reset_password/change_email） | 否 |
| POST | `/auth/login/code` | 邮箱验证码免密登录 | 否 |
| GET | `/auth/sessions` | 会话列表 | 是 |
| DELETE | `/auth/sessions/:id` | 吊销会话 | 是 |
| POST | `/common/email/send-code` | 公开发送邮箱验证码（同 `/auth/email/send-code`） | 否 |
| POST | `/common/refresh` | 用 refresh Cookie 换新 access | 否 |
| POST | `/user/email/change` | 修改邮箱（需登录，先取 change_email 码） | 是 |
| GET | `/user/profile` | 当前用户资料 | 是 |

完整字段与示例见 Swagger（`/swagger/index.html`）。

## 配置

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `GATEWAY_PORT` | `5001` | HTTP 端口 |
| `GATEWAY_USER_SERVICE_URL` | `localhost:5002` | user-service gRPC 地址 |
| `JWT_SECRET` | — | 与 user-service 共享的验签密钥（必填） |
| `GATEWAY_ALLOW_ORIGINS` | `http://localhost:5173` | CORS 允许来源（逗号分隔） |
| `GATEWAY_COOKIE_SECURE` | `false` | Cookie 仅 HTTPS（生产置 true） |
| `GATEWAY_COOKIE_SAMESITE` | `lax` | SameSite 策略 |
| `GATEWAY_COOKIE_DOMAIN` | — | Cookie 作用域 |
| `LOG_*` | — | 日志（公共） |

## 构建与运行

```bash
# 构建
cd services/api-gateway && go build ./cmd/server

# 运行
go run ./cmd/server

# 热重载（air）
air

# 重新生成 Swagger（需安装 swag）
swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

# 检查
go vet ./...
```

## 测试与质量

后端无独立单测目录，依赖 user-service 单元测试覆盖业务；网关侧重契约与路由。提交前执行 `go vet ./...` 并通过 Swagger 生成校验注解完整。
