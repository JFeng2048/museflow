import request from '@/utils/request'
import type {
  LoginPayload,
  RegisterPayload,
  User,
  SocialBinding,
  BindingProvider,
  SendCodePayload,
  LoginWithCodePayload,
  AuthResult,
  MFASetup,
  MFAStatus,
  SessionInfo,
  ChangePasswordPayload,
  ResetPasswordPayload,
  UpdateProfilePayload,
  ChangeEmailPayload,
  UserInfoDto,
  LoginDataDto,
  SessionListDto,
  MFASetupDto,
  MFAStatusDto,
  SendCodeDataDto,
  OAuthBindingListDto,
  PermissionListDto,
} from '@/types/system/auth'

/* ---------------- 线格式 → 领域模型转换 ----------------
 * 后端用 snake_case 且时间为 Unix 秒，前端统一用 camelCase 与 ISO 字符串，
 * 转换集中在这一层，视图与 store 不受后端字段命名影响。
 */

/** Unix 秒 → ISO 字符串。 */
function toISO(seconds?: number): string {
  if (!seconds) return new Date(0).toISOString()
  return new Date(seconds * 1000).toISOString()
}

function mapUser(dto: UserInfoDto): User {
  return {
    id: dto.uuid,
    name: dto.nickname,
    email: dto.email,
    avatar: dto.avatar_url,
    bio: dto.bio,
    createdAt: toISO(dto.created_at),
    mfaEnabled: dto.mfa_enabled,
    emailVerified: dto.email_verified,
  }
}

/**
 * 登录类接口统一出口。
 *
 * 后端 UserInfo 不含角色字段，权限码由服务端在每次请求时从 token 解析并校验，
 * 前端不做权限判定，角色信息由 store 自行维护（缺省按写作者处理）。
 */
function toAuthResult(dto: LoginDataDto): AuthResult {
  return {
    token: dto.access_token ?? '',
    user: mapUser(dto.user),
    requiresMfa: dto.requires_mfa,
    mfaTicket: dto.mfa_ticket,
  }
}

/** 设备名称：后端用于会话列表展示，前端按 UA 粗略识别即可。 */
function deviceName(): string {
  const ua = navigator.userAgent
  const browser = /Edg\//.test(ua)
    ? 'Edge'
    : /Chrome\//.test(ua)
      ? 'Chrome'
      : /Firefox\//.test(ua)
        ? 'Firefox'
        : /Safari\//.test(ua)
          ? 'Safari'
          : 'Browser'
  const os = /Windows/.test(ua)
    ? 'Windows'
    : /Mac OS X/.test(ua)
      ? 'macOS'
      : /Android/.test(ua)
        ? 'Android'
        : /iPhone|iPad/.test(ua)
          ? 'iOS'
          : 'Unknown'
  return `${browser} · ${os}`
}

// ---------------- 注册 / 登录 ----------------

export function login(payload: LoginPayload): Promise<AuthResult> {
  // 后端字段为 email；前端表单沿用 username（兼容邮箱/用户名两种输入习惯）
  return request
    .post<LoginDataDto>('/auth/login', {
      email: payload.username,
      password: payload.password,
      device_name: deviceName(),
    })
    .then(toAuthResult)
}

/** 验证码免密登录。 */
export function loginWithCode(payload: LoginWithCodePayload): Promise<AuthResult> {
  return request
    .post<LoginDataDto>('/auth/login/code', {
      email: payload.email,
      code: payload.code,
      device_name: deviceName(),
    })
    .then(toAuthResult)
}

/** 2FA 登录二次验证（验证码或恢复码）。 */
export function verifyMfaLogin(mfaTicket: string, code: string): Promise<AuthResult> {
  return request
    .post<LoginDataDto>('/auth/mfa/verify-login', {
      mfa_ticket: mfaTicket,
      code,
      device_name: deviceName(),
    })
    .then(toAuthResult)
}

/**
 * 注册。
 *
 * 后端 /auth/register 只返回用户信息、不下发令牌（注册不等于登录），
 * 因此注册成功后自动登录一次换取令牌；若账号需先完成邮箱验证才能登录，
 * 自动登录会失败，此时返回空令牌，由页面引导用户手动登录。
 */
export async function register(payload: RegisterPayload): Promise<AuthResult> {
  // 后端只需 email/password/nickname/code，确认密码仅前端校验，不上传
  const user = await request
    .post<UserInfoDto>('/auth/register', {
      email: payload.email,
      password: payload.password,
      nickname: payload.name || payload.email.split('@')[0],
      code: payload.code,
    })
    .then(mapUser)

  try {
    return await login({ username: payload.email, password: payload.password })
  } catch {
    return { token: '', user }
  }
}

/**
 * 退出登录：让服务端失效当前 access token 与其绑定的 refresh token。
 * 显式传入 token，便于本地已清空登录态后仍能带上旧令牌完成失效。
 */
export function logout(token?: string): Promise<void> {
  return request.post<void>(
    '/auth/logout',
    undefined,
    token ? { headers: { Authorization: `Bearer ${token}` } } : undefined,
  )
}

/** 重置密码：先用 scene=reset_password 发码，再带验证码提交新密码。 */
export function resetPassword(payload: ResetPasswordPayload): Promise<void> {
  return request.post<void>('/auth/password/reset', {
    email: payload.email,
    code: payload.code,
    new_password: payload.newPassword,
  })
}

// ---------------- 邮箱验证码 ----------------

/** 发送邮箱验证码的结果：邮件为异步发送，可用 taskId 订阅进度。 */
export interface SendCodeResult {
  /** 异步任务 ID，用于 SSE 订阅（GET /common/tasks/{task_id}/stream）。 */
  taskId: string
  /** 验证码有效期（秒）。 */
  expiresIn: number
}

/**
 * 发送邮箱验证码（注册 / 免密登录 / 重置密码 / 变更邮箱）。
 * 验证码由服务端下发到邮箱，前端不预填任何值。
 */
export function sendCode(payload: SendCodePayload): Promise<SendCodeResult> {
  return request
    .post<SendCodeDataDto>('/common/email/send-code', {
      email: payload.email,
      scene: payload.scene,
      captcha_token: payload.turnstileToken,
    }, { timeout: 60000 })
    .then((data) => ({
      taskId: data?.task_id ?? '',
      expiresIn: data?.expires_in ?? 600,
    }))
}

// ---------------- 用户资料 ----------------

export function fetchProfile(): Promise<User> {
  return request.get<UserInfoDto>('/user/profile').then(mapUser)
}

/** 更新个人资料（昵称 / 头像 / 简介），留空字段表示不修改。 */
export function updateProfile(payload: UpdateProfilePayload): Promise<User> {
  return request
    .put<UserInfoDto>('/user/profile', {
      nickname: payload.name,
      avatar_url: payload.avatar,
      bio: payload.bio,
    })
    .then(mapUser)
}

/** 变更邮箱：先向新邮箱发码（scene=change_email），再带新邮箱与验证码提交。 */
export function changeEmail(payload: ChangeEmailPayload): Promise<void> {
  return request.post<void>('/user/email/change', {
    new_email: payload.newEmail,
    code: payload.code,
  })
}

/** 当前登录用户的权限码列表。 */
export function fetchMyPermissions(): Promise<string[]> {
  return request
    .get<PermissionListDto>('/user/permissions')
    .then((data) => data?.permissions ?? [])
}

// ---------------- 第三方账号绑定 ----------------

/**
 * 绑定第三方账号。
 * 网关目前只提供绑定列表与解绑接口，绑定需走服务端 OAuth 重定向、尚未暴露，
 * 因此这里暂时直接失败，由调用方提示「暂不支持」。
 */
export function bindProvider(_provider: BindingProvider): Promise<SocialBinding> {
  return Promise.reject(new Error('第三方绑定尚未开放，请前往服务端完成 OAuth 授权'))
}

/** 已绑定的第三方账号列表。 */
export function listOAuthBindings(): Promise<Record<string, SocialBinding>> {
  return request.get<OAuthBindingListDto>('/user/oauth').then((data) => {
    const result: Record<string, SocialBinding> = {}
    for (const b of data?.bindings ?? []) {
      result[b.provider] = {
        nickname: b.provider_nickname || b.provider_email || b.provider_user_id,
        boundAt: toISO(b.created_at),
      }
    }
    return result
  })
}

/** 解除第三方账号绑定。 */
export function unbindProvider(provider: BindingProvider): Promise<void> {
  return request.delete<void>(`/user/oauth/${encodeURIComponent(provider)}`)
}

// ---------------- 两步验证（2FA / TOTP） ----------------

export function setupMfa(): Promise<MFASetup> {
  return request.post<MFASetupDto>('/mfa/setup').then((data) => ({
    secret: data.secret,
    otpauthUrl: data.otpauth_url,
  }))
}

/** 校验 TOTP 或恢复码，正式启用 2FA，返回恢复码。 */
export function verifyMfa(code: string): Promise<{ recoveryCodes: string[] }> {
  return request
    .post<{ recovery_codes: string[] }>('/mfa/verify', { code })
    .then((data) => ({ recoveryCodes: data?.recovery_codes ?? [] }))
}

export function disableMfa(code: string): Promise<void> {
  return request.post<void>('/mfa/disable', { code })
}

export function getMfaStatus(): Promise<MFAStatus> {
  return request.get<MFAStatusDto>('/mfa/status').then((data) => ({
    enabled: data.enabled,
    remainingRecoveryCodes: data.remaining_recovery_codes,
  }))
}

/** 重新生成恢复码：需先用当前 TOTP 验证码确认身份。 */
export function regenerateRecoveryCodes(code: string): Promise<string[]> {
  return request
    .post<{ recovery_codes: string[] }>('/mfa/recovery-codes', { code })
    .then((data) => data?.recovery_codes ?? [])
}

// ---------------- 会话管理 ----------------

export function listSessions(): Promise<SessionInfo[]> {
  return request.get<SessionListDto>('/user/sessions').then((data) =>
    (data?.sessions ?? []).map((s) => ({
      tokenId: s.token_id,
      deviceId: s.device_id,
      deviceName: s.device_name,
      loginAt: toISO(s.login_time),
      lastRefreshAt: toISO(s.last_refresh_time),
    })),
  )
}

/** 强制下线指定会话。 */
export function revokeSession(tokenId: string): Promise<void> {
  return request.delete<void>(`/user/sessions/${encodeURIComponent(tokenId)}`)
}

// ---------------- 密码 ----------------

/** 修改密码（需登录），后端字段为 old_password / new_password。 */
export function changePassword(payload: ChangePasswordPayload): Promise<void> {
  return request.put<void>('/user/password', {
    old_password: payload.oldPassword,
    new_password: payload.newPassword,
  })
}
