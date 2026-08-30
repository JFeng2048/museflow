/** 用户角色：写作者（普通用户）与管理员共用同一前端，按角色控制入口。 */
export type UserRole = 'writer' | 'admin'

/** 已激活的视图：写作者工作台（user）或管理后台（admin）。 */
export type ViewMode = 'user' | 'admin'

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
  /** 角色；缺省按写作者处理（兼容未显式标注的旧 mock 用户）。 */
  role?: UserRole
  /** 是否开启两步验证（2FA）。 */
  mfaEnabled?: boolean
  /** 邮箱是否已验证。 */
  emailVerified?: boolean
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
  /** 注册用的邮箱验证码。 */
  code?: string
}

/** 验证码场景：注册校验 / 免密登录 / 补验证邮箱。 */
export type CodeScene = 'register' | 'login' | 'verify'

export interface SendCodePayload {
  email: string
  scene: CodeScene
  /** Cloudflare Turnstile 人机验证令牌，真实环境必填。 */
  turnstileToken?: string
}

export interface LoginWithCodePayload {
  email: string
  code: string
}

/** 登录/注册返回结果，新增 2FA 中间态字段。 */
export interface AuthResult {
  token: string
  user: User
  /** 账号开启 2FA 时需要二次验证，本次不返回有效令牌。 */
  requiresMfa?: boolean
  /** 2FA 中间票据，用于 VerifyMFALogin。 */
  mfaTicket?: string
}

/** TOTP 设置信息：密钥 + 二维码绑定地址。 */
export interface MFASetup {
  /** base32 共享密钥。 */
  secret: string
  /** otpauth:// 绑定地址，供前端生成二维码。 */
  otpauthUrl: string
}

/** 2FA 状态。 */
export interface MFAStatus {
  enabled: boolean
  /** 剩余可用的恢复码数量。 */
  remainingRecoveryCodes: number
}

/** 单个活跃会话（设备）。 */
export interface SessionInfo {
  /** 会话标识，用于强制下线。 */
  tokenId: string
  deviceId: string
  deviceName: string
  /** 登录时间 ISO 字符串。 */
  loginAt: string
  /** 最后刷新时间 ISO 字符串。 */
  lastRefreshAt: string
}

export interface ChangePasswordPayload {
  oldPassword: string
  newPassword: string
}

export interface ResetPasswordPayload {
  email: string
  code: string
  newPassword: string
}
