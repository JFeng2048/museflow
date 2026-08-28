import request from '@/utils/request'
import { novels } from '@/mock/novels'
import type { LoginPayload, RegisterPayload, User, SocialBinding, BindingProvider } from '@/types/system/auth'
import type { Novel } from '@/types/novel'
import type { AuthResult } from './type'

const MOCK_PASSWORD = 'museflow@museflow.com'

const MOCK_USER: User = {
  id: 'u-1',
  name: '林知秋',
  email: 'demo@museflow.com',
  bio: '在星海与长安之间反复横跳的写作者',
  createdAt: new Date(Date.now() - 86400000 * 90).toISOString(),
  role: 'writer',
}

/** 演示账号：一个普通写作者、一个管理员。密码统一为 museflow@museflow.com。 */
const MOCK_ACCOUNTS: (User & { password: string })[] = [
  {
    id: 'u-1',
    name: '林知秋',
    email: 'demo@museflow.com',
    bio: '在星海与长安之间反复横跳的写作者',
    createdAt: new Date(Date.now() - 86400000 * 90).toISOString(),
    role: 'writer',
    password: MOCK_PASSWORD,
  },
  {
    id: 'u-admin',
    name: '管理员',
    email: 'admin@museflow.com',
    bio: '平台运营与管理',
    createdAt: new Date(Date.now() - 86400000 * 365).toISOString(),
    role: 'admin',
    password: MOCK_PASSWORD,
  },
]

export function login(payload: LoginPayload): Promise<AuthResult> {
  return request.post<AuthResult>('/auth/login', payload).catch(() => {
    const found = MOCK_ACCOUNTS.find(
      (a) => a.email === payload.username && a.password === payload.password,
    )
    if (!found) {
      return Promise.reject(new Error('账号或密码错误'))
    }
    const { password: _pw, ...user } = found
    return { token: `mock-token-${user.id}`, user }
  })
}

export function register(payload: RegisterPayload): Promise<AuthResult> {
  return request.post<AuthResult>('/auth/register', payload).catch(() => ({
    token: 'mock-token-demo',
    user: { ...MOCK_USER, name: payload.name || MOCK_USER.name, email: payload.email },
  }))
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
