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

/** 验证码场景：注册 / 免密登录 / 重置密码 / 变更邮箱。 */
export type CodeScene = 'register' | 'login' | 'reset_password' | 'change_email'

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

export interface UpdateProfilePayload {
  /** 昵称，留空表示不修改。 */
  name?: string
  bio?: string
  avatar?: string
}

export interface ChangeEmailPayload {
  newEmail: string
  code: string
}

/* ---------------- 后端线格式（snake_case） ----------------
 * 对应 api-gateway 的 internal/dto，仅 api 层内部用于转换，
 * 视图与 store 仍使用上面的 camelCase 领域类型。
 */

/** userdto.UserInfo */
export interface UserInfoDto {
  uuid: string
  email: string
  phone?: string
  nickname: string
  avatar_url?: string
  bio?: string
  status: number
  email_verified: boolean
  phone_verified: boolean
  mfa_enabled: boolean
  last_login_at?: number
  created_at: number
}

/** authdto.LoginResponseData：登录 / 验证码登录 / 2FA 二次验证共用。 */
export interface LoginDataDto {
  access_token?: string
  token_type?: string
  expires_in?: number
  user: UserInfoDto
  requires_mfa: boolean
  mfa_ticket?: string
  recovery_codes?: string[]
  remaining_recovery_codes?: number
}

/** userdto.SessionInfo */
export interface SessionInfoDto {
  token_id: string
  device_id: string
  device_name: string
  login_time: number
  last_refresh_time: number
}

/** userdto.SessionList：会话列表被包在 sessions 字段里。 */
export interface SessionListDto {
  sessions: SessionInfoDto[]
}

/** authdto.MFASetupData */
export interface MFASetupDto {
  secret: string
  otpauth_url: string
}

/** authdto.MFAStatusData */
export interface MFAStatusDto {
  enabled: boolean
  remaining_recovery_codes: number
}

/** commondto.SendVerifyCodeData：HTTP 202，异步发送。 */
export interface SendCodeDataDto {
  task_id: string
  expires_in: number
}

/** userdto.OAuthBinding */
export interface OAuthBindingDto {
  provider: string
  provider_user_id: string
  provider_email?: string
  provider_nickname?: string
  provider_avatar?: string
  last_login_at?: number
  created_at: number
}

/** userdto.OAuthBindingList */
export interface OAuthBindingListDto {
  bindings: OAuthBindingDto[]
}

/** userdto.PermissionListData */
export interface PermissionListDto {
  permissions: string[]
}
