export interface PublishChannel {
  id: string
  /** 平台显示名，如「起点读书」 */
  name: string
  /** 是否启用自动同步 */
  enabled: boolean
  /** 连接状态 */
  status: 'connected' | 'disconnected'
  /** 登录账号（手机号 / 邮箱 / 开放平台账号） */
  account?: string
  /** 笔名 / 作者名 */
  penName?: string
  /** 是否 requires 登录态（如网文平台都需要登录后台） */
  requiresLogin?: boolean
  /** 平台说明 */
  desc?: string
}
