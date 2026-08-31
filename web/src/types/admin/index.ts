/** 管理后台（控制台）领域类型。仅管理员侧使用，普通用户不感知。 */

import type { UserInfoDto } from '@/types/system/auth'

/**
 * 后端用户状态（admindto.UpdateUserStatusRequest）。
 * 1=正常 2=冻结 3=已注销 4=待审核
 */
export type AdminUserStatusValue = 1 | 2 | 3 | 4

/** 用户管理列表项：由 GET /admin/users 返回，字段与后端一致。 */
export interface AdminUser {
  /** 用户 UUID。 */
  id: string
  name: string
  email: string
  avatar?: string
  /** 角色编码列表，如 ['admin'] / ['user']。 */
  roles: string[]
  status: AdminUserStatusValue
  emailVerified: boolean
  mfaEnabled: boolean
  createdAt: string
  updatedAt: string
  /** 最终权限列表，仅 GET /admin/users/{uuid} 详情接口返回。 */
  permissions?: string[]
}

/** 角色信息。 */
export interface AdminRole {
  id: number
  /** 角色编码，如 admin / user / auditor。 */
  code: string
  name: string
  description: string
  /** 系统内置角色不可修改 / 删除。 */
  isSystem: boolean
  createdAt: string
}

/** 权限信息。 */
export interface AdminPermission {
  id: number
  /** 权限编码，如 user:read。 */
  code: string
  name: string
  resource: string
  action: string
  description: string
}

/** 审计日志条目。 */
export interface AdminAuditLog {
  id: string
  /** 发生时间 ISO 字符串。 */
  time: string
  /** 操作类型，如 login / change_password。 */
  action: string
  /** 资源类型，如 auth / user / role。 */
  resource: string
  resourceId: string
  /** 操作人 UUID，空表示系统操作。 */
  actor: string
  ip: string
  /** 详情（后端为 JSON 字符串）。 */
  detail?: string
}

/** 分页查询结果。 */
export interface PageResult<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
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

/* ---------------- 后端线格式（snake_case） ----------------
 * 对应 api-gateway 的 internal/dto/admin_dto，仅 api 层内部使用。
 */

/** admindto.AdminUserItem */
export interface AdminUserItemDto {
  user: UserInfoDto
  roles: string[]
  updated_at: number
}

/** admindto.UserList */
export interface UserListDto {
  users: AdminUserItemDto[]
  total: number
  page: number
  page_size: number
}

/** admindto.UserDetail */
export interface UserDetailDto {
  user: AdminUserItemDto
  permissions: string[]
}

/** admindto.RoleInfo */
export interface RoleInfoDto {
  id: number
  code: string
  name: string
  description: string
  is_system: boolean
  created_at: number
}

/** admindto.RoleList */
export interface RoleListDto {
  roles: RoleInfoDto[]
}

/** admindto.PermissionInfo（加 Admin 前缀以区别于 system/auth 的同名权限列表）。 */
export interface AdminPermissionInfoDto {
  id: number
  code: string
  name: string
  resource: string
  action: string
  description: string
}

/** admindto.PermissionList */
export interface AdminPermissionListDto {
  permissions: AdminPermissionInfoDto[]
}

/** admindto.AuditLogItem */
export interface AuditLogItemDto {
  id: number
  user_uuid: string
  action: string
  resource: string
  resource_id: string
  ip?: string
  user_agent?: string
  detail?: string
  created_at: number
}

/** admindto.AuditLogList */
export interface AuditLogListDto {
  logs: AuditLogItemDto[]
  total: number
  page: number
  page_size: number
}
