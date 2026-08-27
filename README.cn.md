# 🚀 MuseFlow

> AI 驱动的智能化小说生成平台 · 从灵感到发布的全链路创作工具

<p align="center">
  <a href="README.md">English</a> · <a href="README.cn.md">中文</a>
</p>

[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat&logo=go)](https://golang.org)
[![Vue 3](https://img.shields.io/badge/Vue-3.4+-4FC08D?style=flat&logo=vue.js)](https://vuejs.org)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.28+-326CE5?style=flat&logo=kubernetes)](https://kubernetes.io)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

---

## 📖 简介

**MuseFlow** 是一个基于 **Go 微服务架构** 构建的智能化小说创作平台。它将 **数据采集（Crawl4AI）**、**RAG 知识增强生成**、**异步任务调度** 与 **多平台自动发布** 融为一体，形成从"市场分析 → AI 创作 → 渠道分发"的完整内容生产闭环。

> 名字寓意：**Muse**（灵感女神）+ **Flow**（流程/流动）= 灵感之流，源源不断的创作力。

---

## ✨ 核心特性

| 模块 | 特性 | 说明 |
| :--- | :--- | :--- |
| 📦 **项目管理** | 小说创建/管理/状态流转 | 支持草稿→连载→完结全生命周期 |
| 📚 **设定集 (Lorebook)** | 角色/世界观/伏笔管理 | 保证 AI 生成内容的一致性 |
| ✍️ **智能生成** | 大纲生成/章节续写/改写/扩写 | 基于 RAG 的知识增强生成 |
| 🕷️ **数据采集** | 网络小说/素材自动抓取 | 集成 Crawl4AI，构建个人素材库 |
| 📤 **自动发布** | 定时发布/多平台适配 | 支持番茄小说等平台（策略可扩展） |
| 📊 **数据分析** | 写作统计/角色出场分析 | 数据驱动的创作洞察 |

---

## 🧱 架构概览

仓库为 **Monorepo**，所有代码同处一个 Git 仓库。微服务之间通过 **gRPC** 通信，对外统一由 **API Gateway** 暴露 HTTP 接口。

```
                ┌──────────────────────────┐
   浏览器/客户端 │        API Gateway        │  (Gin, :5001)
   ─────HTTP────▶  /api/v1/* + Swagger       │
                │  JWT 鉴权 / CORS / Cookie  │
                └───────────┬──────────────┘
                            │ gRPC
                ┌───────────▼──────────────┐
                │       User Service        │  (gRPC, :5002)
                │  注册 / 登录 / 双令牌认证    │
                │  用户数据访问 (GORM)        │
                └──────┬───────────┬────────┘
                       │           │
                  ┌────▼───┐  ┌────▼────┐
                  │Postgres│  │ Redis   │
                  │user_svc│  │ 白/黑名单│
                  └────────┘  └─────────┘
```

### 已实现的后端模块

| 目录 | 角色 | 技术栈 | 端口 |
| :--- | :--- | :--- | :--- |
| `services/user-service` | 用户服务（gRPC Server） | gRPC / GORM / PostgreSQL / Redis / JWT | 5002 |
| `services/api-gateway` | API 网关（HTTP Server） | Gin / gRPC Client / JWT / Swagger | 5001 |
| `pkg/envloader` | 分层配置加载库 | Go 标准库 | — |
| `pkg/errcode` | 统一错误码与多语言响应库 | Go 标准库 | — |
| `pkg/logger` | 结构化日志库（slog + lumberjack） | Go slog / lumberjack | — |
| `proto/user` | 共享 gRPC 契约（user.proto 及生成代码） | protobuf | — |
| `database/user_svc.sql` | 用户库表结构（序列、触发器、schema 命名空间） | PostgreSQL DDL | — |

### 双令牌认证（概要）

- **access token**：短期（默认 1h），无状态 JWT，经响应 body 返回，请求时置于 `Authorization` 头
- **refresh token**：长期（默认 30d），JWT + Redis 白名单，存于 HttpOnly Cookie
- **登出**：删除 refresh 白名单 + 将 access token 的 `jti` 写入 Redis 黑名单（TTL 取剩余有效期），到期后自动清理
- **设备指纹**：`sha256(deviceId + User-Agent + IP)`，防止 refresh token 跨设备盗用
- 完整设计见 [`docs/cn/develop/双令牌认证系统设计文档.md`](docs/cn/develop/双令牌认证系统设计文档.md)

---

## 🗂️ 目录结构

```
MuseFlow/
├── proto/                       # 共享 gRPC API 契约（含生成代码）
│   └── user/                    # user.proto 及生成代码（user.pb.go / user_grpc.pb.go）
├── pkg/                         # 跨服务共享的 Go 基础库（独立 go.mod）
│   ├── envloader/               # 分层 .env 加载（系统 > 服务 .env > 根 .env > 默认值）
│   ├── errcode/                 # 统一错误码与多语言（中/英）响应
│   └── logger/                  # 基于 slog + lumberjack 的日志器（含文件切割）
├── services/
│   ├── user-service/            # 用户服务（gRPC，:5002）
│   │   ├── cmd/server/main.go    # 服务入口：装配 config/repo/service/handler
│   │   ├── .env.example          # 服务配置模板（已提交；.env 已被 gitignore）
│   │   └── internal/
│   │       ├── config/           # 配置加载（USER_ / DB_ / REDIS_ / JWT_SECRET / LOG_）
│   │       ├── handler/          # gRPC 处理器（proto ↔ 业务层转换）
│   │       ├── service/          # 业务逻辑层（按领域拆分子包）
│   │       │   ├── auth/         #   认证业务（AuthService：注册/登录/刷新/登出/校验）
│   │       │   ├── token/        #   JWT 管理（manager / claims / fingerprint）
│   │       │   └── dto/          #   Service 内部数据对象（Device / TokenPair）
│   │       ├── repository/       # 数据访问（GORM 模型读写 + Redis 令牌存储）
│   │       └── model/            # GORM 实体（user_svc.users）
│   └── api-gateway/             # API 网关（HTTP，:5001）
│       ├── .env.example          # 服务配置模板（已提交；.env 已被 gitignore）
│       └── internal/
│           ├── config/           # 配置加载（GATEWAY_ / REDIS_ / JWT_SECRET / LOG_）
│           ├── router/           # Gin 路由装配
│           ├── middleware/       # 中间件（CORS / 鉴权 / 访问日志 / 请求 ID）
│           ├── handler/          # HTTP 处理器（dto ↔ proto 转换）
│           ├── client/           # user-service gRPC 客户端
│           └── dto/              # HTTP 层 DTO（请求/响应结构，供 Swagger 生成）
├── database/
│   └── user_svc.sql             # 用户库 DDL（schema 命名空间 user_svc + 序列/触发器）
├── deploy/                      # 部署相关（K8s / Redis 配置等）
├── crawl4ai-service/            # 数据采集服务（Python）
├── docs/                        # 设计文档（含双令牌认证系统设计文档）
├── web/                         # 前端（Vue 3 + TypeScript + Vite）
├── scripts/                     # 代码生成等脚本
├── .env                         # 全局配置文件（已 gitignore，含开发默认值）
├── .env.example                 # 全局配置模板（已提交，复制为 .env 后填写）
├── go.work                      # Go Workspace（本地多模块开发）
├── Makefile                     # 根目录构建脚本（含 Air 热重载、proto 生成等）
├── README.md / README.cn.md
└── LICENSE
```

---

## 🚀 快速开始

### 前置依赖

- Go 1.26+
- PostgreSQL 16+
- Redis 7+
- protoc + `protoc-gen-go` + `protoc-gen-go-grpc`（仅修改 proto 时需要）
- swag（仅重新生成 Swagger 文档时需要）

### 1. 初始化工作区

```bash
make init        # 生成 go.work 并安装 protoc-gen-go / swag 等工具
```

### 2. 准备数据库与缓存

```bash
# 导入用户库表结构（含 schema、序列、触发器）
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f database/user_svc.sql
```

> 服务端**不执行 AutoMigrate**，schema 由 `database/user_svc.sql` 统一管理。

### 3. 配置环境变量（分层管理）

配置采用**分层加载**，每个服务除根目录 `.env` 外，还可在自身目录放置 `.env` 覆盖同名项：

```
services/
├── user-service/
│   └── .env        # 服务专属配置（覆盖全局同名变量）
└── api-gateway/
    └── .env
.env                # 仓库根目录全局配置（默认/公共值）
```

**加载优先级（由高到低）：**

```
系统环境变量  >  服务自身 .env  >  仓库根 .env  >  代码默认值
```

| 前缀 | 服务 | 关键变量 |
| :--- | :--- | :--- |
| `GATEWAY_` | api-gateway | `GATEWAY_PORT`、`GATEWAY_USER_SERVICE_URL`、`GATEWAY_ALLOW_ORIGINS`、`GATEWAY_COOKIE_*` |
| `USER_` | user-service | `USER_PORT`、`USER_ACCESS_TTL_SECONDS` 等 |
| （无前缀） | 公共 | `DB_HOST`、`DB_PORT`、`DB_USER`、`DB_PASSWORD`、`DB_NAME`（所有服务共用的数据库连接）；`REDIS_ADDR`、`REDIS_PASSWORD`、`REDIS_DB`（所有服务共用的 Redis 连接）；`JWT_SECRET`（gateway 与 user-service 共用的 JWT 签名密钥） |
| `LOG_` | 所有服务 | `LOG_LEVEL`、`LOG_FORMAT`、`LOG_OUTPUT_PATH`、`LOG_CONSOLE` 等日志配置 |

`.env` / `services/*/.env` 已被 `.gitignore` 忽略（含真实密钥，不入库）。仓库提供 `.env.example` 与
`services/user-service/.env.example` 作为模板，本地复制为 `.env` 后按需修改：

```bash
cp .env.example .env                              # 全局
cp services/user-service/.env.example services/user-service/.env   # 服务专属
```

如需覆盖某项，可设置系统环境变量（优先级最高），例如：
```bash
set JWT_SECRET=your-strong-secret      # Windows
# export JWT_SECRET=your-strong-secret  # Linux/macOS
```

> 各服务只读取自己前缀的变量。加载优先级：**系统环境变量 > `.env` 文件 > 代码默认值**。

### 4. 本地启动

```bash
# 终端 A：用户服务
make run-user

# 终端 B：API 网关
make run-gateway
```

启动后控制台会打印访问地址与 Swagger 地址：

- API 网关：http://localhost:5001
- Swagger 文档：http://localhost:5001/swagger/index.html
- 用户服务（gRPC）：localhost:5002

### 5. 常用命令

```bash
make build       # 编译全部服务到 bin/
make test        # 运行测试
make vet         # 静态检查
make proto       # 重新生成 gRPC 代码
make swagger     # 重新生成 Swagger 文档
make docker      # 构建 Docker 镜像（上下文为仓库根目录）
```

> **Windows 用户（无 make 环境）**：仓库已提供等价批处理脚本，直接双击或在 PowerShell/CMD 中执行即可：
>
> | 脚本 | 等价命令 | 说明 |
> | :--- | :--- | :--- |
> | `scripts\run-user.bat` | `make run-user` | 普通启动 user-service |
> | `scripts\run-gateway.bat` | `make run-gateway` | 普通启动 api-gateway |
> | `scripts\watch-user.bat` | `make watch-user` | Air 热重载 user-service |
> | `scripts\watch-gateway.bat` | `make watch-gateway` | Air 热重载 api-gateway |
> | `scripts\watch.bat` | `make watch` | 同时热重载两个服务（各开一个窗口） |
>
> Air 需先安装：`go install github.com/air-verse/air@latest`（脚本会自动把 `%USERPROFILE%\go\bin` 加入 PATH）。
> 批处理中的中文提示在部分老版 CMD 编码下可能显示乱码，不影响功能。

---

## 📡 API 一览

| 方法 | 路径 | 鉴权 | 功能 |
| :--- | :--- | :--- | :--- |
| POST | `/api/v1/auth/register` | 否 | 用户注册（email） |
| POST | `/api/v1/auth/login` | 否 | 登录（签发双令牌） |
| POST | `/api/v1/auth/refresh` | 否（Cookie） | 刷新 access token |
| POST | `/api/v1/auth/logout` | 是 | 登出（吊销当前设备） |
| GET  | `/api/v1/user/profile` | 是 | 获取当前用户信息 |
| GET  | `/health` | 否 | 健康检查 |
| GET  | `/swagger/index.html` | 否 | Swagger 文档 |

完整接口定义与字段说明见 Swagger 文档。

---

## 📦 容器化

`services/api-gateway/Dockerfile` 与 `services/user-service/Dockerfile` 均为多阶段构建，
使用 `golang:1.23-alpine` 编译、`alpine:3.20` 运行，并以非 root 用户启动。

```bash
docker build -f services/user-service/Dockerfile -t museflow/user-service .
docker build -f services/api-gateway/Dockerfile  -t museflow/api-gateway  .
```

构建上下文需为**仓库根目录**（服务依赖同仓库的 `proto` 模块）。

---

## 📄 许可证

[MIT License](LICENSE)

---

## 📧 联系方式

- 作者：[jfeng2048]
- 邮箱：[jfeng2048@outlook.com]
- 项目链接：[https://github.com/JFeng2048/museflow](https://github.com/JFeng2048/museflow)

---

**如果觉得这个项目对你有帮助，欢迎 ⭐ Star 支持！**
