import request from '@/utils/request'
import { novels } from '@/mock'
import type { LoginPayload, RegisterPayload, User, Novel } from '@/types'

const MOCK_USER: User = {
  id: 'u-1',
  username: 'demo',
  nickname: '灵感捕手',
  email: 'demo@museflow.ai',
  bio: '在星海与长安之间反复横跳的写作者',
  createdAt: new Date(Date.now() - 86400000 * 90).toISOString(),
}

export function login(payload: LoginPayload) {
  return request.post<{ token: string; user: User }>('/auth/login', payload).catch(() => ({
    token: 'mock-token-demo',
    user: { ...MOCK_USER, username: payload.username },
  }))
}

export function register(payload: RegisterPayload) {
  return request.post<{ token: string; user: User }>('/auth/register', payload).catch(() => ({
    token: 'mock-token-demo',
    user: { ...MOCK_USER, username: payload.username, email: payload.email },
  }))
}

export function fetchProfile() {
  return request.get<User>('/user/profile').catch(() => MOCK_USER)
}

export function fetchNovelsForUser() {
  return request.get<Novel[]>('/user/novels').catch(() => novels)
}
