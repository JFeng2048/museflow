export interface User {
  id: string
  username: string
  nickname: string
  email: string
  avatar?: string
  bio?: string
  createdAt: string
}

export interface LoginPayload {
  username: string
  password: string
}

export interface RegisterPayload {
  username: string
  email: string
  password: string
  confirmPassword: string
}
