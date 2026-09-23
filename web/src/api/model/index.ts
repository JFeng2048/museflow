import request from '@/utils/request'
import type { PageResult } from '@/types/common'
import type {
  ModelCapabilities,
  ModelProtocol,
  ModelType,
  ModelSource,
  ModelProvider,
  AIModel,
  UserProvider,
  UserModel,
  AvailableModel,
  SystemSetting,
} from '@/types/model'
import type {
  CapabilitiesDto,
  ModelDeleteDto,
  ModelInfoDto,
  ModelListDto,
  ModelPageDto,
  AvailableModelDto,
  ProviderInfoDto,
  SettingInfoDto,
  UserModelInfoDto,
  UserProviderInfoDto,
} from '@/types/model/dto'

/**
 * 模型目录与系统配置接口（/api/v1）。
 *
 * 两类调用方的鉴权强度不同：
 *  - /admin/* 需要 system:admin 权限，维护的是平台渠道（含平台 api_key）与系统配置；
 *  - /user/*  只需登录，只操作当前用户自己的渠道与模型。
 * 前端不做权限判定，无权限时后端返回 403，由调用方提示。
 */

/** 把查询参数拼成 query string，自动丢弃空值。 */
function toQuery(params: Record<string, unknown>): string {
  const usp = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === null || v === '') continue
    usp.set(k, String(v))
  }
  const qs = usp.toString()
  return qs ? `?${qs}` : ''
}

/** 后端能力字段齐全时才认为有效，缺省按全关处理。 */
function mapCapabilities(dto?: CapabilitiesDto): ModelCapabilities {
  return {
    stream: !!dto?.stream,
    toolCall: !!dto?.tool_call,
    jsonMode: !!dto?.json_mode,
    vision: !!dto?.vision,
  }
}

function mapCapabilitiesRequest(caps?: ModelCapabilities): CapabilitiesDto {
  return {
    stream: !!caps?.stream,
    tool_call: !!caps?.toolCall,
    json_mode: !!caps?.jsonMode,
    vision: !!caps?.vision,
  }
}

function mapProvider(dto: ProviderInfoDto): ModelProvider {
  return {
    id: dto.id,
    code: dto.code,
    name: dto.name,
    protocol: dto.protocol as ModelProtocol,
    baseUrl: dto.base_url || '',
    apiKeyHint: dto.api_key_hint || '',
    apiKeyUpdatedAt: dto.api_key_updated_at || '',
    organization: dto.organization || '',
    isActive: !!dto.is_active,
    sortOrder: dto.sort_order || 0,
    createdAt: dto.created_at || '',
    updatedAt: dto.updated_at || '',
  }
}

function mapModel(dto: ModelInfoDto): AIModel {
  return {
    id: dto.id,
    providerId: dto.provider_id,
    code: dto.code,
    name: dto.name,
    modelType: dto.model_type as ModelType,
    apiModel: dto.api_model || '',
    contextWindow: dto.context_window || 0,
    maxOutputTokens: dto.max_output_tokens || 0,
    capabilities: mapCapabilities(dto.capabilities),
    creditCost: dto.credit_cost || 0,
    description: dto.description || '',
    isActive: !!dto.is_active,
    sortOrder: dto.sort_order || 0,
    createdAt: dto.created_at || '',
    updatedAt: dto.updated_at || '',
  }
}

function mapUserProvider(dto: UserProviderInfoDto): UserProvider {
  return {
    id: dto.id,
    userUuid: dto.user_uuid || '',
    name: dto.name,
    protocol: dto.protocol as ModelProtocol,
    baseUrl: dto.base_url || '',
    apiKeyHint: dto.api_key_hint || '',
    apiKeyUpdatedAt: dto.api_key_updated_at || '',
    organization: dto.organization || '',
    isActive: !!dto.is_active,
    createdAt: dto.created_at || '',
    updatedAt: dto.updated_at || '',
  }
}

function mapUserModel(dto: UserModelInfoDto): UserModel {
  return {
    id: dto.id,
    userUuid: dto.user_uuid || '',
    // 线格式是 user_provider_id，与平台模型的 provider_id 不同名。
    providerId: dto.user_provider_id,
    name: dto.name,
    modelType: dto.model_type as ModelType,
    apiModel: dto.api_model || '',
    contextWindow: dto.context_window || 0,
    maxOutputTokens: dto.max_output_tokens || 0,
    capabilities: mapCapabilities(dto.capabilities),
    description: dto.description || '',
    isActive: !!dto.is_active,
    createdAt: dto.created_at || '',
    updatedAt: dto.updated_at || '',
  }
}

function mapAvailable(dto: AvailableModelDto): AvailableModel {
  return {
    source: (dto.source === 'custom' ? 'custom' : 'platform') as ModelSource,
    modelId: dto.model_id,
    providerId: dto.provider_id,
    name: dto.name,
    modelType: dto.model_type as ModelType,
    apiModel: dto.api_model || '',
    contextWindow: dto.context_window || 0,
    maxOutputTokens: dto.max_output_tokens || 0,
    capabilities: mapCapabilities(dto.capabilities),
    // 自定义模型的积分价恒为 0：费用由用户直接付给上游厂商。
    creditCost: dto.credit_cost || 0,
    description: dto.description || '',
  }
}

/**
 * 读取端的配置值解码。
 *
 * value 列是 jsonb：value_type=string 的值入库前被 JSON 序列化过一次，
 * 读回来外层会多一对引号（"100"），这里还原成用户原本填写的文本；
 * number / bool / json 入库的本身就是合法 JSON 文本，原样返回。
 */
function decodeSettingValue(valueType: string, raw?: string): string {
  if (!raw) return ''
  if (valueType === 'string') {
    try {
      const parsed: unknown = JSON.parse(raw)
      return typeof parsed === 'string' ? parsed : raw
    } catch {
      return raw
    }
  }
  return raw
}

function mapSetting(dto: SettingInfoDto): SystemSetting {
  return {
    id: dto.id,
    configGroup: dto.config_group || '',
    key: dto.key || '',
    value: decodeSettingValue(dto.value_type || 'string', dto.value),
    secretValue: dto.secret_value || '',
    isSecret: !!dto.is_secret,
    valueType: dto.value_type || 'string',
    isPublic: !!dto.is_public,
    description: dto.description || '',
    createdAt: dto.created_at || '',
    updatedAt: dto.updated_at || '',
  }
}

/** 把分页信封统一成 PageResult，字段缺失时兜底成空列表。 */
function toPage<T, R>(data: ModelPageDto<T> | undefined, map: (dto: T) => R): PageResult<R> {
  return {
    items: (data?.items ?? []).map(map),
    total: data?.total ?? 0,
    page: data?.page ?? 1,
    pageSize: data?.page_size ?? 20,
  }
}

// ---------------- 平台渠道（管理端）----------------

export interface ListProvidersParams {
  page?: number
  pageSize?: number
  /** 按编码或名称模糊搜索。 */
  keyword?: string
  onlyActive?: boolean
}

/** 分页查询平台渠道。 */
export function listProviders(params: ListProvidersParams = {}): Promise<PageResult<ModelProvider>> {
  const query = toQuery({
    page: params.page,
    page_size: params.pageSize,
    keyword: params.keyword,
    only_active: params.onlyActive,
  })
  return request
    .get<ModelPageDto<ProviderInfoDto>>(`/admin/model-providers${query}`)
    .then((data) => toPage(data, mapProvider))
}

export interface CreateProviderPayload {
  /** 渠道编码，被业务代码引用的稳定标识，创建后不可修改。 */
  code: string
  name: string
  protocol: ModelProtocol
  baseUrl: string
  /** 留空表示该渠道暂不配置密钥。 */
  apiKey?: string
  organization?: string
  sortOrder?: number
}

/** 新增平台渠道。 */
export function createProvider(payload: CreateProviderPayload): Promise<ModelProvider> {
  return request.post<ProviderInfoDto>('/admin/model-providers', {
    code: payload.code,
    name: payload.name,
    protocol: payload.protocol,
    base_url: payload.baseUrl,
    api_key: payload.apiKey,
    organization: payload.organization,
    sort_order: payload.sortOrder,
  }).then(mapProvider)
}

export interface UpdateProviderPayload {
  name: string
  protocol: ModelProtocol
  baseUrl: string
  /** 留空表示不轮换密钥，后端保留原值。 */
  apiKey?: string
  organization?: string
  sortOrder?: number
}

/** 编辑平台渠道；渠道编码不在请求体内，不可修改。 */
export function updateProvider(id: number, payload: UpdateProviderPayload): Promise<ModelProvider> {
  return request.put<ProviderInfoDto>(`/admin/model-providers/${id}`, {
    name: payload.name,
    protocol: payload.protocol,
    base_url: payload.baseUrl,
    api_key: payload.apiKey,
    organization: payload.organization,
    sort_order: payload.sortOrder,
  }).then(mapProvider)
}

/** 启用 / 停用平台渠道。 */
export function setProviderActive(id: number, isActive: boolean): Promise<ModelProvider> {
  return request
    .put<ProviderInfoDto>(`/admin/model-providers/${id}/active`, { is_active: isActive })
    .then(mapProvider)
}

/** 删除平台渠道；渠道下仍有模型时后端返回 400。 */
export function deleteProvider(id: number): Promise<void> {
  return request.delete<ModelDeleteDto>(`/admin/model-providers/${id}`).then(() => undefined)
}

// ---------------- 平台模型（管理端）----------------

export interface ListModelsParams {
  page?: number
  pageSize?: number
  /** 按渠道过滤，0 表示不限。 */
  providerId?: number
  modelType?: ModelType | ''
  /** 按编码、名称或调用标识模糊搜索。 */
  keyword?: string
  onlyActive?: boolean
}

/** 分页查询平台模型。 */
export function listModels(params: ListModelsParams = {}): Promise<PageResult<AIModel>> {
  const query = toQuery({
    page: params.page,
    page_size: params.pageSize,
    provider_id: params.providerId,
    model_type: params.modelType,
    keyword: params.keyword,
    only_active: params.onlyActive,
  })
  return request
    .get<ModelPageDto<ModelInfoDto>>(`/admin/models${query}`)
    .then((data) => toPage(data, mapModel))
}

export interface CreateModelPayload {
  providerId: number
  /** 模型编码，稳定标识，创建后不可修改。 */
  code: string
  name: string
  modelType: ModelType
  apiModel: string
  contextWindow?: number
  maxOutputTokens?: number
  capabilities?: ModelCapabilities
  creditCost?: number
  description?: string
  sortOrder?: number
}

/** 在指定平台渠道下新增模型。 */
export function createModel(payload: CreateModelPayload): Promise<AIModel> {
  return request.post<ModelInfoDto>('/admin/models', {
    provider_id: payload.providerId,
    code: payload.code,
    name: payload.name,
    model_type: payload.modelType,
    api_model: payload.apiModel,
    context_window: payload.contextWindow,
    max_output_tokens: payload.maxOutputTokens,
    capabilities: mapCapabilitiesRequest(payload.capabilities),
    credit_cost: payload.creditCost,
    description: payload.description,
    sort_order: payload.sortOrder,
  }).then(mapModel)
}

export interface UpdateModelPayload {
  name: string
  modelType: ModelType
  apiModel: string
  contextWindow?: number
  maxOutputTokens?: number
  capabilities?: ModelCapabilities
  creditCost?: number
  description?: string
  sortOrder?: number
}

/** 编辑平台模型；模型编码与所属渠道均不可修改。 */
export function updateModel(id: number, payload: UpdateModelPayload): Promise<AIModel> {
  return request.put<ModelInfoDto>(`/admin/models/${id}`, {
    name: payload.name,
    model_type: payload.modelType,
    api_model: payload.apiModel,
    context_window: payload.contextWindow,
    max_output_tokens: payload.maxOutputTokens,
    capabilities: mapCapabilitiesRequest(payload.capabilities),
    credit_cost: payload.creditCost,
    description: payload.description,
    sort_order: payload.sortOrder,
  }).then(mapModel)
}

/** 上架 / 下架平台模型。 */
export function setModelActive(id: number, isActive: boolean): Promise<AIModel> {
  return request
    .put<ModelInfoDto>(`/admin/models/${id}/active`, { is_active: isActive })
    .then(mapModel)
}

/** 删除平台模型。 */
export function deleteModel(id: number): Promise<void> {
  return request.delete<ModelDeleteDto>(`/admin/models/${id}`).then(() => undefined)
}

// ---------------- 系统配置（管理端）----------------

export interface ListSettingsParams {
  /** 按分组过滤，空表示全部分组。 */
  configGroup?: string
  onlyPublic?: boolean
}

/** 系统配置列表；机密项只返回是否已配置，绝不返回明文。 */
export function listSettings(params: ListSettingsParams = {}): Promise<SystemSetting[]> {
  const query = toQuery({ config_group: params.configGroup, only_public: params.onlyPublic })
  return request
    .get<ModelListDto<SettingInfoDto>>(`/admin/settings${query}`)
    .then((data) => (data?.items ?? []).map(mapSetting))
}

/** 按分组与键名读取单条配置。 */
export function getSetting(configGroup: string, key: string): Promise<SystemSetting> {
  return request
    .get<SettingInfoDto>(`/admin/settings/${encodeURIComponent(configGroup)}/${encodeURIComponent(key)}`)
    .then(mapSetting)
}

export interface UpsertSettingPayload {
  /** 非机密值；isSecret 为 true 时留空。 */
  value?: string
  /** 机密值明文，只在写入响应里回显一次。 */
  secretValue?: string
  isSecret?: boolean
  valueType?: string
  isPublic?: boolean
  description?: string
}

/** 按 (config_group, key) 存在则更新、不存在则创建。 */
export function upsertSetting(
  configGroup: string,
  key: string,
  payload: UpsertSettingPayload,
): Promise<SystemSetting> {
  return request
    .put<SettingInfoDto>(`/admin/settings/${encodeURIComponent(configGroup)}/${encodeURIComponent(key)}`, {
      value: payload.value,
      secret_value: payload.secretValue,
      is_secret: payload.isSecret,
      value_type: payload.valueType,
      is_public: payload.isPublic,
      description: payload.description,
    })
    .then(mapSetting)
}

/** 按分组与键名删除配置。 */
export function deleteSetting(configGroup: string, key: string): Promise<void> {
  return request
    .delete<ModelDeleteDto>(
      `/admin/settings/${encodeURIComponent(configGroup)}/${encodeURIComponent(key)}`,
    )
    .then(() => undefined)
}

// ---------------- 我的自定义渠道（用户端）----------------

export interface ListUserProvidersParams {
  page?: number
  pageSize?: number
}

/** 我的自定义渠道列表。 */
export function listUserProviders(
  params: ListUserProvidersParams = {},
): Promise<PageResult<UserProvider>> {
  const query = toQuery({ page: params.page, page_size: params.pageSize })
  return request
    .get<ModelPageDto<UserProviderInfoDto>>(`/user/model-providers${query}`)
    .then((data) => toPage(data, mapUserProvider))
}

export interface UserProviderPayload {
  name: string
  protocol: ModelProtocol
  baseUrl: string
  /** 新建必填；编辑时留空表示不轮换。 */
  apiKey?: string
  organization?: string
  /** 可空：只在显式传入时改动启用状态。 */
  isActive?: boolean
}

function toUserProviderBody(payload: UserProviderPayload) {
  return {
    name: payload.name,
    protocol: payload.protocol,
    base_url: payload.baseUrl,
    api_key: payload.apiKey,
    organization: payload.organization,
    is_active: payload.isActive,
  }
}

/** 添加自定义渠道（用户自带 base_url + api_key）。 */
export function createUserProvider(payload: UserProviderPayload): Promise<UserProvider> {
  return request.post<UserProviderInfoDto>('/user/model-providers', toUserProviderBody(payload)).then(mapUserProvider)
}

/** 编辑自定义渠道；api_key 留空表示不轮换密钥。 */
export function updateUserProvider(id: number, payload: UserProviderPayload): Promise<UserProvider> {
  return request
    .put<UserProviderInfoDto>(`/user/model-providers/${id}`, toUserProviderBody(payload))
    .then(mapUserProvider)
}

/** 删除自定义渠道；渠道下仍有模型时后端返回 400。 */
export function deleteUserProvider(id: number): Promise<void> {
  return request.delete<ModelDeleteDto>(`/user/model-providers/${id}`).then(() => undefined)
}

// ---------------- 我的自定义模型（用户端）----------------

export interface ListUserModelsParams {
  page?: number
  pageSize?: number
  /** 按自定义渠道过滤，0 表示不限。 */
  userProviderId?: number
}

/** 我的自定义模型列表。 */
export function listUserModels(params: ListUserModelsParams = {}): Promise<PageResult<UserModel>> {
  const query = toQuery({
    page: params.page,
    page_size: params.pageSize,
    user_provider_id: params.userProviderId,
  })
  return request
    .get<ModelPageDto<UserModelInfoDto>>(`/user/models${query}`)
    .then((data) => toPage(data, mapUserModel))
}

export interface UserModelPayload {
  /** 自定义渠道 ID。 */
  providerId: number
  name: string
  modelType: ModelType
  apiModel: string
  contextWindow?: number
  maxOutputTokens?: number
  capabilities?: ModelCapabilities
  description?: string
  /** 可空：只在显式传入时改动启用状态。 */
  isActive?: boolean
}

function toUserModelBody(payload: UserModelPayload) {
  return {
    provider_id: payload.providerId,
    name: payload.name,
    model_type: payload.modelType,
    api_model: payload.apiModel,
    context_window: payload.contextWindow,
    max_output_tokens: payload.maxOutputTokens,
    capabilities: mapCapabilitiesRequest(payload.capabilities),
    description: payload.description,
    is_active: payload.isActive,
  }
}

/** 在我的某个自定义渠道下登记模型；不计平台积分。 */
export function createUserModel(payload: UserModelPayload): Promise<UserModel> {
  return request.post<UserModelInfoDto>('/user/models', toUserModelBody(payload)).then(mapUserModel)
}

/** 编辑我的自定义模型；所属渠道不可修改。 */
export function updateUserModel(id: number, payload: UserModelPayload): Promise<UserModel> {
  return request.put<UserModelInfoDto>(`/user/models/${id}`, toUserModelBody(payload)).then(mapUserModel)
}

/** 删除我的自定义模型。 */
export function deleteUserModel(id: number): Promise<void> {
  return request.delete<ModelDeleteDto>(`/user/models/${id}`).then(() => undefined)
}

// ---------------- 可用模型与公开配置（用户端）----------------

export interface ListAvailableModelsParams {
  modelType?: ModelType | ''
}

/** 可用模型清单：平台已上架模型 + 我的自定义模型，供模型选择器使用。 */
export function listAvailableModels(
  params: ListAvailableModelsParams = {},
): Promise<AvailableModel[]> {
  const query = toQuery({ model_type: params.modelType })
  return request
    .get<ModelListDto<AvailableModelDto>>(`/user/models/available${query}`)
    .then((data) => (data?.items ?? []).map(mapAvailable))
}

/** 可公开下发的系统配置（如积分单价、功能开关），只读。 */
export function listPublicSettings(): Promise<SystemSetting[]> {
  return request
    .get<ModelListDto<SettingInfoDto>>('/user/settings')
    .then((data) => (data?.items ?? []).map(mapSetting))
}
