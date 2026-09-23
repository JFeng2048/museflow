# config-service 接口说明

系统配置域服务（gRPC，`:5004`）。模型渠道与模型目录、用户自定义模型、通用系统配置的读写与凭证加解密都在这里。HTTP 不直接暴露，由 api-gateway 转发，对外统一挂在 `/api/v1` 下。

## 作用

- **平台渠道与模型**：厂商、`base_url`、`api_key` 由管理端维护，模型明细挂在渠道下，是用户端「刷出来」的可用模型。
- **用户自定义渠道与模型**：用户自带 `base_url` 与 `api_key`，模型只能挂在自己的渠道下，不计平台积分。
- **可用模型合并视图**：`ListAvailableModels` 把平台模型与用户自定义模型合成一份清单返回，是用户端模型选择器的数据源。
- **通用系统配置**：key-value + jsonb 的可变更配置，机密值加密落库；只有标 `is_public` 的项才下发前端。
- **上游模型目录探测**：`FetchProviderModels` 拉取厂商 `/models` 目录，是后台登记模型的前置步骤，也是本服务唯一出网的路径。

设计取舍、表结构与密钥方案见[模型与系统配置设计文档](../develop/模型与系统配置设计文档.md)。

## 数据表

DDL 见 `services/config-service/database/config_svc.sql`。

| 表 | 说明 |
| :--- | :--- |
| `config_svc.model_provider` | 平台模型渠道，存储厂商 `base_url` 与加密后的 `api_key` |
| `config_svc.model` | 平台模型明细：类型、调用标识、上下文窗口、计费 |
| `config_svc.user_model_provider` | 用户自定义渠道，凭据由用户本人填写 |
| `config_svc.user_model` | 用户自选模型，挂在用户自定义渠道下 |
| `config_svc.system_setting` | 通用系统配置，`value` 与 `secret_value` 二选一 |

## 接口（gRPC 方法）

完整契约见 `proto/model/model.proto`，`package model`，Go 包 `github.com/museflow/proto/model;modelpb`。

- 平台渠道：`ListProviders` / `CreateProvider` / `UpdateProvider` / `DeleteProvider` / `SetProviderActive`
- 平台模型：`ListModels` / `CreateModel` / `UpdateModel` / `DeleteModel` / `SetModelActive`
- 用户渠道：`ListUserProviders` / `CreateUserProvider` / `UpdateUserProvider` / `DeleteUserProvider`
- 用户模型：`ListUserModels` / `CreateUserModel` / `UpdateUserModel` / `DeleteUserModel`
- 合并视图：`ListAvailableModels`
- 系统配置：`ListSettings` / `GetSetting` / `UpsertSetting` / `DeleteSetting`
- 上游探测：`FetchProviderModels`

契约层面的密钥约定：请求里的 `api_key` 是只写字段，加密落库后不回显；响应里只有 `api_key_hint`（末 4 位）与 `api_key_updated_at`，没有任何字段能还原出完整密钥。

## 对外 HTTP 路由

前缀 `/api/v1`。完整字段与示例见网关 Swagger（`/swagger/index.html`），标签 `model-模型与系统配置`。`/admin` 组除登录外还需 `system:admin` 权限，`/user` 组只需登录。

### 平台渠道（管理端）

| Method | Path | 作用 |
| :--- | :--- | :--- |
| GET | `/api/v1/admin/model-providers` | 渠道列表，`keyword` / `only_active` / `page` / `page_size` |
| POST | `/api/v1/admin/model-providers` | 新增渠道，`api_key` 加密落库后只回显末 4 位 |
| PUT | `/api/v1/admin/model-providers/:id` | 编辑渠道；`code` 不可改，`api_key` 留空表示不轮换 |
| PUT | `/api/v1/admin/model-providers/:id/active` | 启用 / 停用 |
| DELETE | `/api/v1/admin/model-providers/:id` | 删除渠道；旗下还有模型时返回失败，提示先清理 |
| POST | `/api/v1/admin/model-providers/remote-models` | 拉取渠道模型目录 |

### 平台模型（管理端）

| Method | Path | 作用 |
| :--- | :--- | :--- |
| GET | `/api/v1/admin/models` | 模型列表，`provider_id` / `model_type` / `keyword` / `only_active` / 分页 |
| POST | `/api/v1/admin/models` | 新增模型，需指定 `provider_id` |
| PUT | `/api/v1/admin/models/:id` | 编辑模型；`code` 与所属渠道均不可改 |
| PUT | `/api/v1/admin/models/:id/active` | 上架 / 下架 |
| DELETE | `/api/v1/admin/models/:id` | 删除模型 |

### 系统配置（管理端）

| Method | Path | 作用 |
| :--- | :--- | :--- |
| GET | `/api/v1/admin/settings` | 配置列表，`config_group` / `only_public` |
| GET | `/api/v1/admin/settings/:config_group/:key` | 单条配置 |
| PUT | `/api/v1/admin/settings/:config_group/:key` | 写入配置；`value` 与 `secret_value` 二选一 |
| DELETE | `/api/v1/admin/settings/:config_group/:key` | 删除配置 |

### 我的渠道与模型（用户端）

| Method | Path | 作用 |
| :--- | :--- | :--- |
| GET | `/api/v1/user/model-providers` | 我的自定义渠道列表 |
| POST | `/api/v1/user/model-providers` | 新增自定义渠道 |
| PUT | `/api/v1/user/model-providers/:id` | 编辑自定义渠道；`api_key` 留空表示不轮换 |
| DELETE | `/api/v1/user/model-providers/:id` | 删除自定义渠道 |
| POST | `/api/v1/user/model-providers/remote-models` | 拉取我的渠道模型目录 |
| GET | `/api/v1/user/models` | 我的模型列表 |
| POST | `/api/v1/user/models` | 新增自定义模型，只能挂在自己的渠道下 |
| PUT | `/api/v1/user/models/:id` | 编辑自定义模型 |
| DELETE | `/api/v1/user/models/:id` | 删除自定义模型 |

### 视图与公开配置（用户端）

| Method | Path | 作用 |
| :--- | :--- | :--- |
| GET | `/api/v1/user/models/available` | 可用模型合并视图（平台 + 自定义），`model_type` 可空 |
| GET | `/api/v1/user/settings` | 公开系统配置，只回 `is_public=true` 的项 |

**可用模型条目**：`source` 取 `platform` / `custom` 区分来源，`model_id` 与 `provider_id` 按来源分别指向平台表或用户表，`credit_cost` 平台模型按平台定价、自定义模型恒为 0。条目不含任何密钥字段。

## 密钥与脱敏

- 写入：`api_key` / `secret_value` 在服务端 AES-256-GCM 加密后落库，格式 `v1:<nonce_b64>:<ciphertext_b64>`。
- 读取：只有 `api_key_hint`（末 4 位）与 `api_key_updated_at` 会出现在响应里；机密配置的明文只在写入响应中回显一次。
- 用户端接口的 `user_uuid` 一律取自登录态，不接受请求体覆盖，因此访问不到别人的渠道。

## 上游模型目录探测

`POST /admin/model-providers/remote-models` 与 `POST /user/model-providers/remote-models` 共用同一段逻辑，凭证三选一：`provider_id`（平台渠道）、`user_provider_id`（我的渠道）、`base_url` + `api_key`（渠道未保存时的临时探测）。按前两种定位时密钥由服务端解密使用，请求与响应都不携带明文。

返回条目只带 `id` / `object` / `owned_by` / `created_at`，其中 `id` 就是后续创建模型时填的 `api_model`。探测只读上游、不写库，超时 15s、响应体上限 4MB、条目上限 500 条。

## 错误码

业务错误在 `services/config-service/internal/handler/mapError` 统一映射为 gRPC status，再由网关 `writeGRPCError` 转成 HTTP 业务码：

| gRPC status | HTTP 业务码 | 触发场景 |
| :--- | :--- | :--- |
| `InvalidArgument` | 参数错误 | `user_uuid` 非法、探测凭证三选一为空 |
| `AlreadyExists` | 资源冲突 | 渠道编码 / 模型编码重名，同渠道下调用标识重复，同用户名下渠道重名 |
| `FailedPrecondition` | 参数错误 | 渠道下还有模型不能删；密文解不开，需重新填写密钥 |
| `NotFound` | 资源不存在 | 资源不存在，或访问了他人的资源（不暴露他人资源是否存在） |
| `Internal` | 服务内部错误 | 加密失败、数据库异常 |

## 关联文档

- [模型与系统配置设计文档](../develop/模型与系统配置设计文档.md)
- [服务架构设计](../architecture/服务架构设计.md)
- [api-gateway 接口说明](api-gateway.md)
