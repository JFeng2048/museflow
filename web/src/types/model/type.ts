/** 模型与供应商配置（支持系统预设 + 用户自定义）。 */

/** 通讯协议。 */
export type ModelProtocol = 'openai' | 'anthropic' | 'gemini' | 'custom'

export interface ProtocolMeta {
  value: ModelProtocol
  label: string
  /** base URL 占位提示。 */
  baseUrlHint: string
  /** 鉴权请求头字段。 */
  authHeader: string
}

export const PROTOCOLS: ProtocolMeta[] = [
  { value: 'openai', label: 'OpenAI 兼容', baseUrlHint: 'https://api.openai.com/v1', authHeader: 'Authorization' },
  { value: 'anthropic', label: 'Anthropic', baseUrlHint: 'https://api.anthropic.com', authHeader: 'x-api-key' },
  { value: 'gemini', label: 'Gemini', baseUrlHint: 'https://generativelanguage.googleapis.com', authHeader: 'x-goog-api-key' },
  { value: 'custom', label: '自定义', baseUrlHint: 'https://your-endpoint/v1', authHeader: 'Authorization' },
]

export interface ModelProvider {
  id: string
  /** 系统预设供应商不可删除/编辑。 */
  system?: boolean
  name: string
  protocol: ModelProtocol
  baseUrl: string
  apiKey: string
  /** 组织（部分供应商需要）。 */
  organization?: string
  createdAt: string
}

export interface AIModel {
  id: string
  providerId: string
  name: string
  /** 实际调用时使用的模型标识，如 gpt-4o。 */
  apiModel: string
  /** 单次调用消耗的积分（系统模型计费；自定义模型免费）。 */
  creditCost: number
  /** 上下文窗口（千 token）。 */
  contextK: number
  /** 系统预设模型标价为平台定价；自定义模型 creditCost=0。 */
  system?: boolean
  enabled: boolean
  description?: string
}
