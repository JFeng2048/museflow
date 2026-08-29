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
} from '@/types/system/auth'
import type { Novel } from '@/types/novel'

/** 是否处于 Mock 模式：由 VITE_ENABLE_MOCK 控制。开启时所有接口直接走本地兜底，
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

function findAccount(email: string, password: string) {
  return [...MOCK_ACCOUNTS, ...MOCK_REGISTERED].find(
    (a) => a.email === email && a.password === password,
  )
}

export function login(payload: LoginPayload): Promise<AuthResult> {
  if (MOCK) return mockLogin(payload)
  return request.post<AuthResult>('/auth/login', payload).catch(() => mockLogin(payload))
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
  return request.post<AuthResult>('/auth/login/code', payload).catch(() => mockLoginWithCode(payload))
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
  return request.post<AuthResult>('/auth/mfa/verify-login', { mfa_ticket: mfaTicket, code }).catch(() => {
    if (code === MOCK_CODE || /^[A-Z0-9]{6,10}$/.test(code)) {
      return { token: `mock-token-mfa-${Date.now()}`, user: { ...MOCK_USER } }
    }
    return Promise.reject(new Error('验证码不正确'))
  })
}

export function register(payload: RegisterPayload): Promise<AuthResult> {
  if (MOCK) return mockRegister(payload)
  return request.post<AuthResult>('/auth/register', payload).catch(() => mockRegister(payload))
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

/** 发送邮箱验证码（注册 / 免密登录 / 补验证）。 */
export function sendCode(payload: SendCodePayload): Promise<void> {
  if (MOCK) return Promise.resolve()
  return request.post<void>('/auth/email/send-code', payload).catch(() => undefined)
}

export function fetchProfile(): Promise<User> {
  return request.get<User>('/user/profile').catch(() => MOCK_USER)
}

export function fetchNovelsForUser(): Promise<Novel[]> {
  return request.get<Novel[]>('/user/novels').catch(() => novels)
}

export function bindProvider(provider: BindingProvider): Promise<SocialBinding> {
  const nickname = provider === 'github' ? 'zhiqiu-dev' : '微信用户_8f3a'
  return request
    .post<SocialBinding>(`/user/bind/${provider}`, {})
    .catch(() => ({ nickname, boundAt: new Date().toISOString() }))
}

export function unbindProvider(provider: BindingProvider): Promise<void> {
  return request.delete<void>(`/user/bind/${provider}`).catch(() => undefined)
}

// ---------------- 两步验证（2FA / TOTP） ----------------

export function setupMfa(): Promise<MFASetup> {
  return request.post<MFASetup>('/auth/mfa/enable').catch(() => ({
    secret: 'JBSWY3DPEHPK3PXP',
    otpauthUrl: 'otpauth://totp/MuseFlow:demo@museflow.app?secret=JBSWY3DPEHPK3PXP&issuer=MuseFlow',
  }))
}

/** 校验 TOTP 或恢复码，正式启用 2FA，返回恢复码。 */
export function verifyMfa(code: string): Promise<{ recoveryCodes: string[] }> {
  return request.post<{ recoveryCodes: string[] }>('/auth/mfa/verify', { code }).catch(() => {
    if (!code || code.length < 6) return Promise.reject(new Error('验证码不正确'))
    return { recoveryCodes: Array.from({ length: 8 }, () => randomCode(10)) }
  })
}

export function disableMfa(code: string): Promise<void> {
  return request.post<void>('/auth/mfa/disable', { code }).catch(() => {
    if (!code || code.length < 6) return Promise.reject(new Error('验证码不正确'))
    return undefined
  })
}

export function getMfaStatus(): Promise<MFAStatus> {
  return request.get<MFAStatus>('/auth/mfa/status').catch(() => ({
    enabled: MOCK_USER.mfaEnabled ?? false,
    remainingRecoveryCodes: 8,
  }))
}

export function getRecoveryCodes(): Promise<string[]> {
  return request.get<string[]>('/auth/mfa/recovery-codes').catch(() =>
    Array.from({ length: 8 }, () => randomCode(10)),
  )
}

// ---------------- 会话管理 ----------------

export function listSessions(): Promise<SessionInfo[]> {
  return request.get<SessionInfo[]>('/auth/sessions').catch(() => [
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

export function revokeSession(tokenId: string): Promise<void> {
  return request.delete<void>(`/auth/sessions/${tokenId}`).catch(() => undefined)
}

// ---------------- 密码 ----------------

export function changePassword(payload: ChangePasswordPayload): Promise<void> {
  return request.post<void>('/auth/change-password', payload).catch(() => undefined)
}

function randomCode(len: number): string {
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789'
  let s = ''
  for (let i = 0; i < len; i++) s += chars[Math.floor(Math.random() * chars.length)]
  return s
}
