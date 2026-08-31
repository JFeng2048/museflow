import request from '@/utils/request'
import type {
  AdminUser,
  AdminRole,
  AdminPermission,
  AdminAuditLog,
  AdminUserStatusValue,
  PageResult,
  AdminUserItemDto,
  UserListDto,
  UserDetailDto,
  RoleInfoDto,
  RoleListDto,
  AdminPermissionInfoDto,
  AdminPermissionListDto,
  AuditLogItemDto,
  AuditLogListDto,
} from '@/types/admin'

/**
 * 管理后台接口（/api/v1/admin/*）。
 *
 * 全部接口在网关层由 RequirePermission(user:admin) 统一校验，
 * 前端不做权限判定，无权限时后端返回 403，由调用方提示。
 */

/** Unix 秒 → ISO 字符串。 */
function toISO(seconds?: number): string {
  if (!seconds) return new Date(0).toISOString()
  return new Date(seconds * 1000).toISOString()
}

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

function mapUserItem(item: AdminUserItemDto): AdminUser {
  const u = item.user
  return {
    id: u.uuid,
    name: u.nickname,
    email: u.email,
    avatar: u.avatar_url,
    roles: item.roles ?? [],
    status: (u.status || 1) as AdminUserStatusValue,
    emailVerified: u.email_verified,
    mfaEnabled: u.mfa_enabled,
    createdAt: toISO(u.created_at),
    updatedAt: toISO(item.updated_at),
  }
}

function mapRole(r: RoleInfoDto): AdminRole {
  return {
    id: r.id,
    code: r.code,
    name: r.name,
    description: r.description,
    isSystem: r.is_system,
    createdAt: toISO(r.created_at),
  }
}

function mapPermission(p: AdminPermissionInfoDto): AdminPermission {
  return {
    id: p.id,
    code: p.code,
    name: p.name,
    resource: p.resource,
    action: p.action,
    description: p.description,
  }
}

function mapAuditLog(l: AuditLogItemDto): AdminAuditLog {
  return {
    id: String(l.id),
    time: toISO(l.created_at),
    action: l.action,
    resource: l.resource,
    resourceId: l.resource_id,
    actor: l.user_uuid,
    ip: l.ip || '',
    detail: l.detail,
  }
}

// ---------------- 用户管理 ----------------

export interface ListUsersParams {
  page?: number
  pageSize?: number
  /** 按邮箱或昵称模糊搜索。 */
  keyword?: string
  /** 状态：1=正常 2=冻结 3=已注销 4=待审核。 */
  status?: AdminUserStatusValue
  /** 排序字段：created_at / updated_at / email / nickname / status。 */
  orderBy?: string
  desc?: boolean
}

/** 分页查询用户列表。 */
export function listUsers(params: ListUsersParams = {}): Promise<PageResult<AdminUser>> {
  const query = toQuery({
    page: params.page,
    page_size: params.pageSize,
    keyword: params.keyword,
    status: params.status,
    order_by: params.orderBy,
    desc: params.desc,
  })
  return request.get<UserListDto>(`/admin/users${query}`).then((data) => ({
    items: (data?.users ?? []).map(mapUserItem),
    total: data?.total ?? 0,
    page: data?.page ?? 1,
    pageSize: data?.page_size ?? 20,
  }))
}

/** 查询用户详情（含最终权限列表）。 */
export function getUserDetail(uuid: string): Promise<AdminUser> {
  return request.get<UserDetailDto>(`/admin/users/${encodeURIComponent(uuid)}`).then((data) => ({
    ...mapUserItem(data.user),
    permissions: data?.permissions ?? [],
  }))
}

/** 修改用户状态（冻结 / 解冻 / 注销）。 */
export function updateUserStatus(uuid: string, status: AdminUserStatusValue): Promise<void> {
  return request.put<void>(`/admin/users/${encodeURIComponent(uuid)}/status`, { status })
}

/** 为用户分配角色（按角色编码）。 */
export function assignRole(uuid: string, roleCode: string): Promise<void> {
  return request.put<void>(`/admin/users/${encodeURIComponent(uuid)}/role`, { role_code: roleCode })
}

// ---------------- 角色管理 ----------------

/** 角色列表。 */
export function listRoles(): Promise<AdminRole[]> {
  return request
    .get<RoleListDto>('/admin/roles')
    .then((data) => (data?.roles ?? []).map(mapRole))
}

export interface CreateRolePayload {
  code: string
  name: string
  description?: string
}

/** 创建角色。 */
export function createRole(payload: CreateRolePayload): Promise<AdminRole> {
  return request
    .post<RoleInfoDto>('/admin/roles', {
      code: payload.code,
      name: payload.name,
      description: payload.description,
    })
    .then(mapRole)
}

export interface UpdateRolePayload {
  name: string
  description?: string
}

/** 编辑角色（系统内置角色不可修改）。 */
export function updateRole(id: number, payload: UpdateRolePayload): Promise<void> {
  return request.put<void>(`/admin/roles/${id}`, {
    name: payload.name,
    description: payload.description,
  })
}

/** 删除角色（系统内置角色不可删除）。 */
export function deleteRole(id: number): Promise<void> {
  return request.delete<void>(`/admin/roles/${id}`)
}

/** 为角色分配权限（覆盖式）。 */
export function setRolePermissions(id: number, codes: string[]): Promise<void> {
  return request.put<void>(`/admin/roles/${id}/permissions`, { permission_codes: codes })
}

// ---------------- 权限 ----------------

/** 权限列表，可按资源类型过滤。 */
export function listPermissions(resource?: string): Promise<AdminPermission[]> {
  const query = toQuery({ resource })
  return request
    .get<AdminPermissionListDto>(`/admin/permissions${query}`)
    .then((data) => (data?.permissions ?? []).map(mapPermission))
}

// ---------------- 审计日志 ----------------

export interface ListAuditLogsParams {
  page?: number
  pageSize?: number
  /** 按操作人 UUID 过滤。 */
  userUuid?: string
  /** 按操作类型过滤，如 login / change_password。 */
  action?: string
  /** 起始时间（Unix 秒）。 */
  from?: number
  /** 结束时间（Unix 秒）。 */
  to?: number
}

/** 分页查询审计日志。 */
export function listAuditLogs(params: ListAuditLogsParams = {}): Promise<PageResult<AdminAuditLog>> {
  const query = toQuery({
    page: params.page,
    page_size: params.pageSize,
    user_uuid: params.userUuid,
    action: params.action,
    from: params.from,
    to: params.to,
  })
  return request.get<AuditLogListDto>(`/admin/audit-logs${query}`).then((data) => ({
    items: (data?.logs ?? []).map(mapAuditLog),
    total: data?.total ?? 0,
    page: data?.page ?? 1,
    pageSize: data?.page_size ?? 20,
  }))
}
