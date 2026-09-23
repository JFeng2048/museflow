# config-service API

The system configuration domain service (gRPC, `:5004`). Model providers and the model catalog, user-defined models, general system settings, and credential encryption/decryption all live here. It is not exposed directly over HTTP; api-gateway proxies to it under the unified `/api/v1` prefix.

## Purpose

- **Platform providers & models**: vendor, `base_url` and `api_key` are maintained in the admin console; model details hang under a provider and are the models "surfaced" to end users.
- **User-defined providers & models**: users bring their own `base_url` and `api_key`; models can only hang under the user's own providers and never consume platform credits.
- **Available-models merged view**: `ListAvailableModels` merges platform models with user-defined models into a single list; it is the data source for the frontend model picker.
- **General system settings**: changeable key-value plus jsonb configuration; secret values are encrypted at rest; only entries flagged `is_public` are delivered to the frontend.
- **Upstream model catalog probing**: `FetchProviderModels` pulls the vendor `/models` catalog. It is the prerequisite for registering models in the console and the only outbound network path of this service.

See the design doc (Chinese) [Model & System Configuration Design](../../cn/develop/模型与系统配置设计文档.md) for trade-offs, table structure, and the key scheme.

## Tables

DDL lives in `services/config-service/database/config_svc.sql`.

| Table | Description |
| :--- | :--- |
| `config_svc.model_provider` | Platform model provider storing vendor `base_url` and encrypted `api_key` |
| `config_svc.model` | Platform model details: type, call identifier, context window, billing |
| `config_svc.user_model_provider` | User-defined provider, credentials supplied by the user |
| `config_svc.user_model` | User-selected models hanging under user-defined providers |
| `config_svc.system_setting` | General system settings, either `value` or `secret_value` |

## Interface (gRPC methods)

Full contract in `proto/model/model.proto`, `package model`, Go package `github.com/museflow/proto/model;modelpb`.

- Platform providers: `ListProviders` / `CreateProvider` / `UpdateProvider` / `DeleteProvider` / `SetProviderActive`
- Platform models: `ListModels` / `CreateModel` / `UpdateModel` / `DeleteModel` / `SetModelActive`
- User providers: `ListUserProviders` / `CreateUserProvider` / `UpdateUserProvider` / `DeleteUserProvider`
- User models: `ListUserModels` / `CreateUserModel` / `UpdateUserModel` / `DeleteUserModel`
- Merged view: `ListAvailableModels`
- System settings: `ListSettings` / `GetSetting` / `UpsertSetting` / `DeleteSetting`
- Upstream probing: `FetchProviderModels`

Key contract convention: `api_key` in requests is write-only. It is encrypted at rest and never echoed back; responses carry only `api_key_hint` (last 4 characters) and `api_key_updated_at`, and no field can reconstruct the full key.

## HTTP Routes

All under the `/api/v1` prefix. Full fields and examples are in the gateway Swagger (`/swagger/index.html`), tag `model-模型与系统配置`. The `/admin` group requires the `system:admin` permission on top of login; the `/user` group requires login only.

### Platform providers (admin)

| Method | Path | Purpose |
| :--- | :--- | :--- |
| GET | `/api/v1/admin/model-providers` | Provider list, `keyword` / `only_active` / `page` / `page_size` |
| POST | `/api/v1/admin/model-providers` | Create provider; the `api_key` is encrypted and only its last 4 characters are echoed |
| PUT | `/api/v1/admin/model-providers/:id` | Edit provider; `code` is immutable, an empty `api_key` keeps the existing key |
| PUT | `/api/v1/admin/model-providers/:id/active` | Enable / disable |
| DELETE | `/api/v1/admin/model-providers/:id` | Delete provider; fails while models still hang under it |
| POST | `/api/v1/admin/model-providers/remote-models` | Probe the provider model catalog |

### Platform models (admin)

| Method | Path | Purpose |
| :--- | :--- | :--- |
| GET | `/api/v1/admin/models` | Model list, `provider_id` / `model_type` / `keyword` / `only_active` / pagination |
| POST | `/api/v1/admin/models` | Create model; `provider_id` is required |
| PUT | `/api/v1/admin/models/:id` | Edit model; `code` and the owning provider are immutable |
| PUT | `/api/v1/admin/models/:id/active` | Publish / unpublish |
| DELETE | `/api/v1/admin/models/:id` | Delete model |

### System settings (admin)

| Method | Path | Purpose |
| :--- | :--- | :--- |
| GET | `/api/v1/admin/settings` | Setting list, `config_group` / `only_public` |
| GET | `/api/v1/admin/settings/:config_group/:key` | Single setting |
| PUT | `/api/v1/admin/settings/:config_group/:key` | Write setting; either `value` or `secret_value` |
| DELETE | `/api/v1/admin/settings/:config_group/:key` | Delete setting |

### My providers & models (user)

| Method | Path | Purpose |
| :--- | :--- | :--- |
| GET | `/api/v1/user/model-providers` | My custom provider list |
| POST | `/api/v1/user/model-providers` | Create custom provider |
| PUT | `/api/v1/user/model-providers/:id` | Edit custom provider; an empty `api_key` keeps the existing key |
| DELETE | `/api/v1/user/model-providers/:id` | Delete custom provider |
| POST | `/api/v1/user/model-providers/remote-models` | Probe my provider catalog |
| GET | `/api/v1/user/models` | My model list |
| POST | `/api/v1/user/models` | Create custom model, only under my own providers |
| PUT | `/api/v1/user/models/:id` | Edit custom model |
| DELETE | `/api/v1/user/models/:id` | Delete custom model |

### Views & public settings (user)

| Method | Path | Purpose |
| :--- | :--- | :--- |
| GET | `/api/v1/user/models/available` | Available models merged view (platform + custom), `model_type` optional |
| GET | `/api/v1/user/settings` | Public system settings, only `is_public=true` entries |

**Available model entry**: `source` is `platform` or `custom` to mark the origin; `model_id` and `provider_id` point into the platform tables or the user tables depending on the source; `credit_cost` follows platform pricing for platform models and is always 0 for custom models. Entries never contain key fields.

## Keys & masking

- Write: `api_key` / `secret_value` are encrypted server-side with AES-256-GCM before persisting, in the format `v1:<nonce_b64>:<ciphertext_b64>`.
- Read: only `api_key_hint` (last 4 characters) and `api_key_updated_at` appear in responses; the plaintext of a secret setting is echoed exactly once in the write response.
- `user_uuid` on user-side endpoints always comes from the login session and cannot be overridden by the request body, so another user's providers are unreachable.

## Upstream catalog probing

`POST /admin/model-providers/remote-models` and `POST /user/model-providers/remote-models` share the same logic, with one of three credential sources: `provider_id` (platform provider), `user_provider_id` (my provider), or `base_url` + `api_key` (ad-hoc probing before the provider is saved). With the first two the server decrypts the key internally, and neither request nor response carries plaintext.

Returned entries carry only `id` / `object` / `owned_by` / `created_at`, where `id` is the `api_model` to fill in when creating the model afterwards. Probing is read-only against the upstream and writes nothing to the database, with a 15s timeout, a 4MB response cap, and a 500-entry cap.

## Error codes

Business errors are mapped to gRPC status in `services/config-service/internal/handler/mapError`, then translated into HTTP business codes by the gateway `writeGRPCError`:

| gRPC status | HTTP business code | Trigger |
| :--- | :--- | :--- |
| `InvalidArgument` | invalid parameter | Illegal `user_uuid`, or no probing credential provided |
| `AlreadyExists` | resource conflict | Duplicate provider/model code, duplicate call identifier under a provider, duplicate provider name for a user |
| `FailedPrecondition` | invalid parameter | Provider still has models so it cannot be deleted; ciphertext cannot be decrypted, the key must be re-entered |
| `NotFound` | resource missing | Resource does not exist, or belongs to someone else (existence is not disclosed) |
| `Internal` | internal error | Encryption failure, database error |

## Related docs

- Model & System Configuration Design (Chinese): `../../cn/develop/模型与系统配置设计文档.md`
- [Service Architecture](../architecture/service-architecture.md)
- [api-gateway API](api-gateway.md)
