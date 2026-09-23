/**
 * 模型目录与系统配置的领域模型。
 *
 * 字段与后端 HTTP DTO（services/api-gateway/internal/dto/model_dto）一一对应，
 * 但统一成前端习惯的 camelCase；snake_case 线格式到领域模型的转换集中在
 * `@/api/model`，视图与 store 不感知后端字段命名。
 *
 * 密钥约定（契约层面强制）：api_key 是只写字段，任何响应都不回显完整密钥，
 * 只有 apiKeyHint（末 4 位）与轮换时间。前端据此把密钥当作「一次性凭据」：
 * 新建时填写一次，编辑时留空表示不轮换。
 */
import type { Component } from 'vue'
import {
  ChatbubbleOutline,
  CodeSlashOutline,
  ConstructOutline,
  CubeOutline,
  EyeOutline,
  GitNetworkOutline,
  ImagesOutline,
  PulseOutline,
  VideocamOutline,
  VolumeHighOutline,
} from '@vicons/ionicons5'

/** 模型能力开关，对应 config_svc 里的 capabilities JSONB 键。 */
export interface ModelCapabilities {
  /** 流式输出 */
  stream: boolean
  /** 工具 / 函数调用 */
  toolCall: boolean
  /** JSON 模式输出 */
  jsonMode: boolean
  /** 图像输入 */
  vision: boolean
}

/** 上游厂商协议。 */
export type ModelProtocol = 'openai' | 'anthropic' | 'gemini' | 'custom'

/** 模型类型：对话、嵌入、重排、视觉、图像、语音、视频。 */
export type ModelType = 'chat' | 'embedding' | 'rerank' | 'vision' | 'image' | 'audio' | 'video'

/** 可用模型来源：platform=平台模型，custom=用户自定义模型。 */
export type ModelSource = 'platform' | 'custom'

export interface ProtocolMeta {
  value: ModelProtocol
  label: string
  /** base URL 输入框占位提示。 */
  baseUrlHint: string
}

/** 协议元信息：标签与各家默认 Base URL。 */
export const PROTOCOLS: ProtocolMeta[] = [
  { value: 'openai', label: 'OpenAI 兼容', baseUrlHint: 'https://api.openai.com/v1' },
  { value: 'anthropic', label: 'Anthropic', baseUrlHint: 'https://api.anthropic.com' },
  { value: 'gemini', label: 'Gemini', baseUrlHint: 'https://generativelanguage.googleapis.com' },
  { value: 'custom', label: '自定义', baseUrlHint: 'https://your-endpoint/v1' },
]

export interface ModelTypeMeta {
  value: ModelType
  label: string
  icon: Component
}

/** 模型类型元信息：图标 + 文案，供筛选项与表格标签共用。 */
export const MODEL_TYPES: ModelTypeMeta[] = [
  { value: 'chat', label: '对话', icon: ChatbubbleOutline },
  { value: 'embedding', label: '嵌入', icon: CubeOutline },
  { value: 'rerank', label: '重排', icon: GitNetworkOutline },
  { value: 'vision', label: '视觉', icon: EyeOutline },
  { value: 'image', label: '图像', icon: ImagesOutline },
  { value: 'audio', label: '语音', icon: VolumeHighOutline },
  { value: 'video', label: '视频', icon: VideocamOutline },
]

export interface CapabilityMeta {
  key: keyof ModelCapabilities
  label: string
  icon: Component
}

/** 能力开关元信息：表格里用图标 + tooltip 展示，避免四列文字占地方。 */
export const CAPABILITIES: CapabilityMeta[] = [
  { key: 'stream', label: '流式输出', icon: PulseOutline },
  { key: 'toolCall', label: '工具调用', icon: ConstructOutline },
  { key: 'jsonMode', label: 'JSON 模式', icon: CodeSlashOutline },
  { key: 'vision', label: '图像输入', icon: EyeOutline },
]

/** 平台渠道（管理端维护：平台提供的 base_url + api_key）。 */
export interface ModelProvider {
  id: number
  /** 渠道编码，被业务代码引用的稳定标识，创建后不可修改。 */
  code: string
  name: string
  protocol: ModelProtocol
  baseUrl: string
  /** API Key 末 4 位；空串表示尚未配置密钥。 */
  apiKeyHint: string
  /** 最近一次写入密钥的时间（ISO）；空串表示从未配置。 */
  apiKeyUpdatedAt: string
  organization: string
  isActive: boolean
  sortOrder: number
  createdAt: string
  updatedAt: string
}

/** 平台模型（挂在平台渠道下，用户端可见的主体）。 */
export interface AIModel {
  id: number
  providerId: number
  /** 模型编码，稳定标识，创建后不可修改。 */
  code: string
  name: string
  modelType: ModelType
  /** 调用上游时使用的模型标识，如 gpt-4o。 */
  apiModel: string
  contextWindow: number
  maxOutputTokens: number
  capabilities: ModelCapabilities
  /** 单次调用消耗的平台积分。 */
  creditCost: number
  description: string
  isActive: boolean
  sortOrder: number
  createdAt: string
  updatedAt: string
}

/** 用户自定义渠道（用户自带 base_url + api_key）。 */
export interface UserProvider {
  id: number
  userUuid: string
  name: string
  protocol: ModelProtocol
  baseUrl: string
  apiKeyHint: string
  apiKeyUpdatedAt: string
  organization: string
  isActive: boolean
  createdAt: string
  updatedAt: string
}

/** 用户自选模型（挂在用户自定义渠道下，不计平台积分）。 */
export interface UserModel {
  id: number
  userUuid: string
  providerId: number
  name: string
  modelType: ModelType
  apiModel: string
  contextWindow: number
  maxOutputTokens: number
  capabilities: ModelCapabilities
  description: string
  isActive: boolean
  createdAt: string
  updatedAt: string
}

/** 用户端可用模型：平台模型 + 用户自定义模型的合并视图，不含任何密钥。 */
export interface AvailableModel {
  source: ModelSource
  modelId: number
  providerId: number
  name: string
  modelType: ModelType
  apiModel: string
  contextWindow: number
  maxOutputTokens: number
  capabilities: ModelCapabilities
  /** 平台模型按平台定价；自定义模型恒为 0（用户直接向上游付费）。 */
  creditCost: number
  description: string
}

/** 系统配置项（config_svc.system_setting）。 */
export interface SystemSetting {
  id: number
  configGroup: string
  key: string
  /** 非机密值；isSecret 为 true 时为空串。 */
  value: string
  /** 机密值明文，只在写入响应里回显一次，列表查询恒为空。 */
  secretValue: string
  isSecret: boolean
  /** string / number / bool / json */
  valueType: string
  /** 是否可下发前端（用户端 /user/settings 只返回公开项）。 */
  isPublic: boolean
  description: string
  createdAt: string
  updatedAt: string
}

/** 系统配置的值类型候选。 */
export const VALUE_TYPES: string[] = ['string', 'number', 'bool', 'json']

/** 模型类型 → 元信息，取不到时兜底成对话。 */
export function modelTypeMeta(type: string): ModelTypeMeta {
  return MODEL_TYPES.find((m) => m.value === type) ?? MODEL_TYPES[0]
}

/** 协议 → 元信息，取不到时兜底成自定义。 */
export function protocolMeta(protocol: string): ProtocolMeta {
  return PROTOCOLS.find((p) => p.value === protocol) ?? PROTOCOLS[PROTOCOLS.length - 1]
}

/** 密钥脱敏展示：只有末 4 位；未配置时返回空串由调用方给提示。 */
export function maskApiKey(hint: string): string {
  return hint ? `•••• ${hint}` : ''
}

/** 空能力集合，供新建模型时初始化表单。 */
export const EMPTY_CAPABILITIES: ModelCapabilities = {
  stream: false,
  toolCall: false,
  jsonMode: false,
  vision: false,
}
