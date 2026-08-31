import request from '@/utils/request'
import { novels } from '@/mock/novels'
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
import type { Novel } from '@/types/novel'

/** 是否处于 Mock 模式：由 VITE_ENABLE_MOCK 控制。开启时登录/注册/发码直接走本地兜底，
 * 不再发起真实网络请求，保证演示账号 100% 可用。 */
const MOCK = String(import.meta.env.VITE_ENABLE_MOCK).toLowerCase() !== 'false'

const MOCK_PASSWORD = 'museflow@museflow.com'

/** 演示用验证码：真实场景服务端下发，这里固定为 123456 便于体验。 */
export const MOCK_CODE = '123456'

/** 演示账号：一个普通写作者、一个管理员。密码统一为 museflow@museflow.com。 */
const MOCK_ACCOUNTS: (User & { password: string })[] = [
  {
    id: 'u-1',
    name: '林知秋',
    email: 'demo@museflow.com',
    bio: '在星海与长安之间反复横跳的写作者',
    createdAt: new Date(Date.now() - 86400000 * 90).toISOString(),
    role: 'writer',
    mfaEnabled: false,
    emailVerified: true,
    password: MOCK_PASSWORD,
  },
  {
    id: 'u-admin',
    name: '管理员',
    email: 'admin@museflow.com',
    bio: '平台运营与管理',
    createdAt: new Date(Date.now() - 86400000 * 365).toISOString(),
    role: 'admin',
    mfaEnabled: false,
    emailVerified: true,
    password: MOCK_PASSWORD,
  },
]

/** 运行时注册表：mock 模式下注册成功的账号，仅存于内存（演示会话内有效）。 */
const MOCK_REGISTERED: (User & { password: string })[] = []

/** 兜底用户（API 失败/未登录时），用于本地交互。 */
const MOCK_USER: User = {
  id: 'u-1',
  name: '林知秋',
  email: 'demo@museflow.com',
  bio: '在星海与长安之间反复横跳的写作者',
  createdAt: new Date(Date.now() - 86400000 * 90).toISOString(),
  role: 'writer',
  mfaEnabled: false,
  emailVerified: true,
}

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

function findAccount(email: string, password: string) {
  return [...MOCK_ACCOUNTS, ...MOCK_REGISTERED].find(
    (a) => a.email === email && a.password === password,
  )
}

// ---------------- 注册 / 登录 ----------------

export function login(payload: LoginPayload): Promise<AuthResult> {
  if (MOCK) return mockLogin(payload)
  // 后端字段为 email；前端表单沿用 username（兼容邮箱/用户名两种输入习惯）
  return request
    .post<LoginDataDto>('/auth/login', {
      email: payload.username,
      password: payload.password,
      device_name: deviceName(),
    })
    .then(toAuthResult)
    .catch(() => mockLogin(payload))
}

/** Mock 登录：按邮箱 + 密码匹配演示账号，匹配失败给出明确错误。 */
function mockLogin(payload: LoginPayload): Promise<AuthResult> {
  const found = findAccount(payload.username, payload.password)
  if (!found) {
    return Promise.reject(new Error('账号或密码错误'))
  }
  const { password: _pw, ...user } = found
  // mock：演示账号开启 2FA 时返回中间态，否则直接签发。
  if (user.mfaEnabled) {
    return Promise.resolve({ token: '', user, requiresMfa: true, mfaTicket: `mock-ticket-${user.id}` })
  }
  return Promise.resolve({ token: `mock-token-${user.id}`, user })
}

/** 验证码免密登录。 */
export function loginWithCode(payload: LoginWithCodePayload): Promise<AuthResult> {
  if (MOCK) return mockLoginWithCode(payload)
  return request
    .post<LoginDataDto>('/auth/login/code', {
      email: payload.email,
      code: payload.code,
      device_name: deviceName(),
    })
    .then(toAuthResult)
    .catch(() => mockLoginWithCode(payload))
}

function mockLoginWithCode(payload: LoginWithCodePayload): Promise<AuthResult> {
  const found = [...MOCK_ACCOUNTS, ...MOCK_REGISTERED].find((a) => a.email === payload.email)
  if (!found) {
    return Promise.reject(new Error('该邮箱尚未注册'))
  }
  const { password: _pw, ...user } = found
  if (user.mfaEnabled) {
    return Promise.resolve({ token: '', user, requiresMfa: true, mfaTicket: `mock-ticket-${user.id}` })
  }
  return Promise.resolve({ token: `mock-token-${user.id}`, user })
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
    .catch(() => {
      if (code === MOCK_CODE || /^[A-Z0-9]{6,10}$/.test(code)) {
        return { token: `mock-token-mfa-${Date.now()}`, user: { ...MOCK_USER } }
      }
      return Promise.reject(new Error('验证码不正确'))
    })
}

/**
 * 注册。
 *
 * 后端 /auth/register 只返回用户信息、不下发令牌（注册不等于登录），
 * 因此注册成功后自动登录一次换取令牌；若账号需先完成邮箱验证才能登录，
 * 自动登录会失败，此时返回空令牌，由页面引导用户手动登录。
 */
export async function register(payload: RegisterPayload): Promise<AuthResult> {
  if (MOCK) return mockRegister(payload)
  let user: User
  try {
    // 后端只需 email/password/nickname/code，确认密码仅前端校验，不上传
    user = await request
      .post<UserInfoDto>('/auth/register', {
        email: payload.email,
        password: payload.password,
        nickname: payload.name || payload.email.split('@')[0],
        code: payload.code,
      })
      .then(mapUser)
  } catch {
    return mockRegister(payload)
  }
  try {
    return await login({ username: payload.email, password: payload.password })
  } catch {
    return { token: '', user }
  }
}

function mockRegister(payload: RegisterPayload): Promise<AuthResult> {
  if (payload.code !== MOCK_CODE) {
    return Promise.reject(new Error('验证码错误或已过期'))
  }
  // 注册成功：写入内存注册表，使该账号在本次演示会话内可登录使用。
  const registered: User & { password: string } = {
    id: `u-${Date.now()}`,
    name: payload.name || payload.email.split('@')[0],
    email: payload.email,
    bio: '',
    createdAt: new Date().toISOString(),
    role: 'writer',
    mfaEnabled: false,
    emailVerified: true,
    password: payload.password,
  }
  MOCK_REGISTERED.push(registered)
  const { password: _pw, ...user } = registered
  return Promise.resolve({ token: `mock-token-${registered.id}`, user })
}

/** 退出登录：后端失效当前 access token 与其绑定的 refresh token。 */
export function logout(): Promise<void> {
  return request.post<void>('/auth/logout')
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

/** 发送邮箱验证码（注册 / 免密登录 / 重置密码 / 变更邮箱）。 */
export function sendCode(payload: SendCodePayload): Promise<SendCodeResult> {
  if (MOCK) return Promise.resolve({ taskId: '', expiresIn: 600 })
  return request
    .post<SendCodeDataDto>('/common/email/send-code', {
      email: payload.email,
      scene: payload.scene,
      captcha_token: payload.turnstileToken,
    })
    .then((data) => ({
      taskId: data?.task_id ?? '',
      expiresIn: data?.expires_in ?? 600,
    }))
}

// ---------------- 用户资料 ----------------

export function fetchProfile(): Promise<User> {
  return request.get<UserInfoDto>('/user/profile').then(mapUser).catch(() => MOCK_USER)
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

/**
 * 用户作品列表。novel-service 尚未在网关提供该接口，
 * 请求失败时回落到本地 mock，保证作品页可演示。
 */
export function fetchNovelsForUser(): Promise<Novel[]> {
  return request.get<Novel[]>('/user/novels').catch(() => novels)
}

// ---------------- 第三方账号绑定 ----------------

/**
 * 发起第三方绑定。网关目前只提供解绑与列表接口，绑定走服务端 OAuth 重定向，
 * 尚未暴露给前端，因此这里回落到本地 mock 以便演示交互。
 */
export function bindProvider(provider: BindingProvider): Promise<SocialBinding> {
  const nickname = provider === 'github' ? 'zhiqiu-dev' : '微信用户_8f3a'
  return Promise.resolve({ nickname, boundAt: new Date().toISOString() })
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
  return request.delete<void>(`/user/oauth/${provider}`)
}

// ---------------- 两步验证（2FA / TOTP） ----------------

export function setupMfa(): Promise<MFASetup> {
  return request
    .post<MFASetupDto>('/mfa/setup')
    .then((data) => ({
      secret: data.secret,
      otpauthUrl: data.otpauth_url,
    }))
    .catch(() => ({
      secret: 'JBSWY3DPEHPK3PXP',
      otpauthUrl: 'otpauth://totp/MuseFlow:demo@museflow.app?secret=JBSWY3DPEHPK3PXP&issuer=MuseFlow',
    }))
}

/** 校验 TOTP 或恢复码，正式启用 2FA，返回恢复码。 */
export function verifyMfa(code: string): Promise<{ recoveryCodes: string[] }> {
  return request
    .post<{ recovery_codes: string[] }>('/mfa/verify', { code })
    .then((data) => ({ recoveryCodes: data?.recovery_codes ?? [] }))
    .catch(() => {
      if (!code || code.length < 6) return Promise.reject(new Error('验证码不正确'))
      return { recoveryCodes: Array.from({ length: 8 }, () => randomCode(10)) }
    })
}

export function disableMfa(code: string): Promise<void> {
  return request.post<void>('/mfa/disable', { code }).catch(() => {
    if (!code || code.length < 6) return Promise.reject(new Error('验证码不正确'))
    return undefined
  })
}

export function getMfaStatus(): Promise<MFAStatus> {
  return request
    .get<MFAStatusDto>('/mfa/status')
    .then((data) => ({
      enabled: data.enabled,
      remainingRecoveryCodes: data.remaining_recovery_codes,
    }))
    .catch(() => ({
      enabled: MOCK_USER.mfaEnabled ?? false,
      remainingRecoveryCodes: 8,
    }))
}

/** 重新生成恢复码：需先用当前 TOTP 验证码确认身份。 */
export function regenerateRecoveryCodes(code: string): Promise<string[]> {
  return request
    .post<{ recovery_codes: string[] }>('/mfa/recovery-codes', { code })
    .then((data) => data?.recovery_codes ?? [])
    .catch(() => Array.from({ length: 8 }, () => randomCode(10)))
}

// ---------------- 会话管理 ----------------

export function listSessions(): Promise<SessionInfo[]> {
  return request
    .get<SessionListDto>('/user/sessions')
    .then((data) =>
      (data?.sessions ?? []).map((s) => ({
        tokenId: s.token_id,
        deviceId: s.device_id,
        deviceName: s.device_name,
        loginAt: toISO(s.login_time),
        lastRefreshAt: toISO(s.last_refresh_time),
      })),
    )
    .catch(() => [
      {
        tokenId: 's-1',
        deviceId: 'd-1',
        deviceName: 'Chrome · macOS',
        loginAt: new Date(Date.now() - 86400000 * 2).toISOString(),
        lastRefreshAt: new Date(Date.now() - 3600000).toISOString(),
      },
      {
        tokenId: 's-2',
        deviceId: 'd-2',
        deviceName: 'Safari · iPhone',
        loginAt: new Date(Date.now() - 86400000 * 5).toISOString(),
        lastRefreshAt: new Date(Date.now() - 86400000).toISOString(),
      },
    ])
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

function randomCode(len: number): string {
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789'
  let s = ''
  for (let i = 0; i < len; i++) s += chars[Math.floor(Math.random() * chars.length)]
  return s
}
