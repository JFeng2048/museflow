/** 积分（Credits）体系。 */

export type CreditSource = 'recharge' | 'task' | 'consume' | 'expire' | 'gift'

export interface CreditPackage {
  id: string
  /** 充值金额（元）。 */
  price: number
  /** 获得积分。 */
  credits: number
  /** 是否热门推荐。 */
  popular?: boolean
  tag?: string
}

export interface CreditTask {
  id: string
  title: string
  reward: number
  /** 每日/单次。 */
  type: 'daily' | 'once'
  done: boolean
  /** 描述。 */
  desc: string
}

export interface CreditRecord {
  id: string
  source: CreditSource
  /** 正为获得，负为消费。 */
  amount: number
  balance: number
  note: string
  at: string
}

/** 积分类型：活动积分有有效期，永久积分无有效期。 */
export type CreditGrantKind = 'activity' | 'permanent'

/** 积分发放明细。 */
export interface CreditGrant {
  id: string
  amount: number
  /** 活动积分的到期时间；永久积分为 null。 */
  expireAt: string | null
  used: number
  /** 类型。 */
  kind: CreditGrantKind
}
