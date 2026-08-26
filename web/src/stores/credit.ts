import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { CreditTask, CreditRecord, CreditGrant, CreditPackage } from '@/types/credit'
import { creditTasks, creditRecords, creditPackages } from '@/mock/credits'

export const useCreditStore = defineStore('credit', () => {
  const records = ref<CreditRecord[]>([...creditRecords])
  const tasks = ref<CreditTask[]>([...creditTasks])
  const packages = ref<CreditPackage[]>([...creditPackages])
  // 积分发放记录：活动积分有过期时间（先到期先扣减），永久积分没有。
  const grants = ref<CreditGrant[]>([
    // 活动积分：1500 分（已用 5 分 + 任务用 100 分），将于 2026-09-02 到期。
    { id: 'g_act_1', kind: 'activity', amount: 1500, used: 5, expireAt: '2026-09-02T00:00:00.000Z' },
    // 任务完成获得的 100 活动积分，30 天后到期。
    { id: 'g_act_2', kind: 'activity', amount: 100, used: 0, expireAt: future(30) },
    // 永久积分：1000 分。
    { id: 'g_perm_1', kind: 'permanent', amount: 1000, used: 0, expireAt: null },
  ])

  // 活动积分余额（已扣除已用 + 已过期）。
  const activityBalance = computed(() => {
    const now = Date.now()
    return grants.value
      .filter((g) => g.kind === 'activity' && g.expireAt && new Date(g.expireAt).getTime() > now)
      .reduce((s, g) => s + (g.amount - g.used), 0)
  })

  // 永久积分余额（仅扣除已用，无过期）。
  const permanentBalance = computed(() =>
    grants.value
      .filter((g) => g.kind === 'permanent')
      .reduce((s, g) => s + (g.amount - g.used), 0),
  )

  // 总可用积分。
  const validBalance = computed(() => activityBalance.value + permanentBalance.value)

  // 最近一个活动积分的到期时间。
  const nextExpiry = computed(() => {
    const now = Date.now()
    const pending = grants.value
      .filter((g) => g.kind === 'activity' && g.expireAt && new Date(g.expireAt).getTime() > now && g.used < g.amount)
      .sort((a, b) => +new Date(a.expireAt!) - +new Date(b.expireAt!))
    return pending[0]?.expireAt || null
  })

  // 充值永久积分。
  function recharge(credits: number, note?: string) {
    grants.value.push({
      id: 'g_perm_' + Math.random().toString(36).slice(2, 6),
      kind: 'permanent',
      amount: credits,
      used: 0,
      expireAt: null,
    })
    pushRecord({ source: 'recharge', amount: credits, note: note || `充值 · ${credits} 永久积分` })
  }

  function pushRecord(r: Omit<CreditRecord, 'id' | 'at' | 'balance'>) {
    records.value.unshift({
      ...r,
      id: 'r_' + Math.random().toString(36).slice(2, 8),
      at: new Date().toISOString(),
      balance: validBalance.value,
    })
  }

  // 消费积分：优先扣减即将过期的活动积分，活动积分不足再扣永久积分。
  function consume(amount: number, note: string): boolean {
    if (amount <= 0) return true
    if (validBalance.value < amount) return false
    let remain = amount
    const now = Date.now()
    // 先扣活动积分（按到期时间升序）。
    const ordered = [...grants.value].sort((a, b) => {
      const ax = a.expireAt ? +new Date(a.expireAt) : Number.POSITIVE_INFINITY
      const bx = b.expireAt ? +new Date(b.expireAt) : Number.POSITIVE_INFINITY
      return ax - bx
    })
    for (const g of ordered) {
      if (remain <= 0) break
      if (g.kind === 'activity' && g.expireAt && new Date(g.expireAt).getTime() <= now) continue
      const avail = g.amount - g.used
      if (avail <= 0) continue
      const take = Math.min(avail, remain)
      g.used += take
      remain -= take
    }
    pushRecord({ source: 'consume', amount: -amount, note })
    return true
  }

  // 完成任务：奖励进入活动积分（90 天有效）。
  function completeTask(taskId: string) {
    const t = tasks.value.find((x) => x.id === taskId)
    if (!t || t.done) return
    t.done = true
    grants.value.push({
      id: 'gt_' + Math.random().toString(36).slice(2, 6),
      kind: 'activity',
      amount: t.reward,
      used: 0,
      expireAt: future(90),
    })
    pushRecord({ source: 'task', amount: t.reward, note: `任务奖励 · ${t.title}` })
  }

  // 处理已过期的活动积分（把余额从活动中回收并入账）。
  function expireDue() {
    const now = Date.now()
    for (const g of grants.value) {
      if (g.kind === 'activity' && g.expireAt && new Date(g.expireAt).getTime() <= now && g.used < g.amount) {
        const lost = g.amount - g.used
        g.used = g.amount
        pushRecord({ source: 'expire', amount: -lost, note: '积分到期回收' })
      }
    }
  }

  return {
    records,
    tasks,
    packages,
    grants,
    activityBalance,
    permanentBalance,
    validBalance,
    nextExpiry,
    consume,
    recharge,
    completeTask,
    expireDue,
  }
})

function future(days: number): string {
  return new Date(Date.now() + days * 86400000).toISOString()
}
