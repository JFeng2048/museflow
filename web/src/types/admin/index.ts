/** 管理后台（控制台）领域类型。仅管理员侧使用，普通用户不感知。 */

import type { UserRole } from '@/types/system/auth'

/** 用户管理列表里展示的用户（比前台 User 多运营维度字段）。 */
export interface AdminUser {
  id: string
  name: string
  email: string
  role: UserRole
  /** 头像 emoji 或 dataURL，缺省按名字首字渲染。 */
  avatar?: string
  avatarColor?: string
  /** 账号状态：正常 / 停用。停用后该用户无法登录。 */
  status: 'active' | 'disabled'
  /** 是否今日活跃（近 24 小时有登录或写作）。 */
  activeToday: boolean
  /** 累计作品数。 */
  novelCount: number
  /** 累计字数。 */
  totalWords: number
  /** 剩余积分。 */
  credits: number
  createdAt: string
  /** 最近活跃时间 ISO 字符串。 */
  lastActiveAt: string
}

/** 创建用户表单。 */
export interface AdminUserCreate {
  name: string
  email: string
  password: string
  role: UserRole
}

/** 管理台首页核心指标。 */
export interface AdminMetrics {
  /** 注册用户总数。 */
  totalUsers: number
  /** 今日新增注册。 */
  newToday: number
  /** 近 7 日新增注册（按日，供趋势条）。 */
  new7: number[]
  /** 全站作品总数。 */
  totalNovels: number
  /** 全站总字数。 */
  totalWords: number
  /** 今日 AI 生成任务数。 */
  genToday: number
  /** 当前在线服务数 / 服务总数。 */
  servicesOnline: number
  servicesTotal: number
}

/** 系统支持的模型（供应商）配置项，可编辑。 */
export interface AdminModel {
  id: string
  /** 模型展示名，如 GPT-4o。 */
  name: string
  /** 供应商，如 OpenAI / 智谱 / 自建。 */
  provider: string
  /** API Base URL，如 https://api.openai.com/v1。 */
  baseUrl: string
  /** API Key（脱敏展示，可编辑）。 */
  apiKey: string
  /** 用途分类。 */
  category: '对话' | '续写' | '推理' | '嵌入' | '图像'
  /** 单次调用上下文上限（token）。 */
  contextWindow: number
  /** 是否启用。 */
  enabled: boolean
}

/** 公告信息。 */
export interface AdminAnnouncement {
  id: string
  /** 标题。 */
  title: string
  /** 正文（支持纯文本，前端按段落渲染）。 */
  content: string
  /** 级别：普通 / 重要 / 维护。 */
  level: 'normal' | 'important' | 'maintenance'
  /** 是否置顶。 */
  pinned: boolean
  /** 发布时间 ISO 字符串。 */
  publishedAt: string
  /** 状态：草稿 / 已发布。 */
  status: 'draft' | 'published'
}

/** 系统日志条目。 */
export interface AdminLog {
  id: string
  /** 发生时间 ISO 字符串。 */
  time: string
  /** 日志级别。 */
  level: 'info' | 'warn' | 'error'
  /** 关联服务 / 模块。 */
  service: string
  /** 操作用户（可为系统）。 */
  actor: string
  /** 日志内容。 */
  message: string
}

/** 服务监控条目。 */
export interface AdminService {
  id: string
  /** 服务名。 */
  name: string
  /** 服务类型 / 技术栈。 */
  kind: string
  /** 健康状态。 */
  status: 'healthy' | 'degraded' | 'down'
  /** 实例数。 */
  instances: number
  /** 平均响应耗时（ms）。 */
  latency: number
  /** CPU 占用率（0-100）。 */
  cpu: number
  /** 内存占用率（0-100）。 */
  memory: number
  /** 最近一次检查时间 ISO 字符串。 */
  checkedAt: string
  /** 访问地址（可选）。 */
  endpoint?: string
}
