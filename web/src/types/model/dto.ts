/**
 * 模型目录与系统配置的「线格式」结构：字段名与后端 HTTP DTO
 * （services/api-gateway/internal/dto/model_dto）完全一致，snake_case 原样保留。
 *
 * 这一层只在 `@/api/model` 的映射函数里出现，视图与 store 一律使用
 * `@/types/model` 里的 camelCase 领域模型，不感知后端字段命名。
 */

/** 模型能力开关（线格式）。 */
export interface CapabilitiesDto {
  stream?: boolean
  tool_call?: boolean
  json_mode?: boolean
  vision?: boolean
}

/** 平台渠道（线格式）。api_key 只写不读，响应只有 hint 与轮换时间。 */
export interface ProviderInfoDto {
  id: number
  code: string
  name: string
  protocol: string
  base_url: string
  api_key_hint?: string
  api_key_updated_at?: string
  organization?: string
  extra?: Record<string, string>
  is_active: boolean
  sort_order?: number
  created_at?: string
  updated_at?: string
}

/** 平台模型（线格式）。 */
export interface ModelInfoDto {
  id: number
  provider_id: number
  code: string
  name: string
  model_type: string
  api_model: string
  context_window?: number
  max_output_tokens?: number
  capabilities?: CapabilitiesDto
  credit_cost?: number
  description?: string
  is_active: boolean
  sort_order?: number
  created_at?: string
  updated_at?: string
}

/** 用户自定义渠道（线格式）。 */
export interface UserProviderInfoDto {
  id: number
  user_uuid: string
  name: string
  protocol: string
  base_url: string
  api_key_hint?: string
  api_key_updated_at?: string
  organization?: string
  extra?: Record<string, string>
  is_active: boolean
  created_at?: string
  updated_at?: string
}

/** 用户自定义模型（线格式）。所属渠道字段是 user_provider_id，与平台模型不同名。 */
export interface UserModelInfoDto {
  id: number
  user_uuid: string
  user_provider_id: number
  name: string
  model_type: string
  api_model: string
  context_window?: number
  max_output_tokens?: number
  capabilities?: CapabilitiesDto
  description?: string
  is_active: boolean
  created_at?: string
  updated_at?: string
}

/** 用户端可用模型（线格式）：平台模型 + 我的自定义模型，不含任何密钥。 */
export interface AvailableModelDto {
  source: string
  model_id: number
  provider_id: number
  name: string
  model_type: string
  api_model: string
  context_window?: number
  max_output_tokens?: number
  capabilities?: CapabilitiesDto
  credit_cost?: number
  description?: string
}

/** 系统配置（线格式）。secret_value 只在写入响应里出现。 */
export interface SettingInfoDto {
  id: number
  config_group: string
  key: string
  value?: string
  secret_value?: string
  is_secret?: boolean
  value_type?: string
  is_public?: boolean
  description?: string
  created_at?: string
  updated_at?: string
}

/** 分页信封（渠道 / 模型 / 用户渠道 / 用户模型共用）。 */
export interface ModelPageDto<T> {
  items?: T[]
  total?: number
  page?: number
  page_size?: number
}

/** 无分页信封（系统配置 / 可用模型）。 */
export interface ModelListDto<T> {
  items?: T[]
}

/** 删除结果信封。 */
export interface ModelDeleteDto {
  success?: boolean
}
