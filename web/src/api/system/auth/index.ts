import request from '@/utils/request'
import { novels } from '@/mock/novels'
import type { LoginPayload, RegisterPayload, User, SocialBinding, BindingProvider } from '@/types/system/auth'
import type { Novel } from '@/types/novel'
import type { AuthResult } from './type'

const MOCK_USER: User = {
  id: 'u-1',
  name: '林知秋',
  email: 'demo@museflow.app',
  bio: '在星海与长安之间反复横跳的写作者',
  createdAt: new Date(Date.now() - 86400000 * 90).toISOString(),
}

export function login(payload: LoginPayload): Promise<AuthResult> {
  return request.post<AuthResult>('/auth/login', payload).catch(() => ({
    token: 'mock-token-demo',
    user: { ...MOCK_USER, email: payload.username },
  }))
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
