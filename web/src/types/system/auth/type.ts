export interface SocialBinding {
  /** 第三方昵称 / openid 显示名 */
  nickname: string
  /** 绑定时间 ISO 字符串 */
  boundAt: string
}

export interface UserBindings {
  github?: SocialBinding
  wechat?: SocialBinding
}

export type BindingProvider = 'github' | 'wechat'

export interface User {
  id: string
  name: string
  email: string
  /** 头像：可以是 emoji 字符，或自定义 dataURL/URL */
  avatar?: string
  /** 头像底色（用于 emoji / 缩写头像） */
  avatarColor?: string
  bio?: string
  createdAt: string
  bindings?: UserBindings
}

export interface LoginPayload {
  username: string
  password: string
}

export interface RegisterPayload {
  name?: string
  email: string
  password: string
  confirmPassword: string
}
