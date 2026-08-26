import type { CreditPackage, CreditTask, CreditRecord } from '@/types/credit'

export const creditPackages: CreditPackage[] = [
  { id: 'p_30', price: 30, credits: 300, tag: '入门' },
  { id: 'p_98', price: 98, credits: 1100, popular: true, tag: '超值' },
  { id: 'p_298', price: 298, credits: 3600, tag: '创作者' },
  { id: 'p_598', price: 598, credits: 8000, tag: '工作室' },
]

export const creditTasks: CreditTask[] = [
  { id: 't_profile', title: '完善个人资料', reward: 50, type: 'once', done: false, desc: '填写昵称与简介，让书房更有你的气息。' },
  { id: 't_first_novel', title: '创建第一部作品', reward: 100, type: 'once', done: false, desc: '开启你的第一本书。' },
  { id: 't_daily_login', title: '每日签到', reward: 20, type: 'daily', done: false, desc: '连续登录，灵感不断电。' },
  { id: 't_share', title: '分享给一位朋友', reward: 30, type: 'once', done: false, desc: '把好用的书房安利出去。' },
  { id: 't_feedback', title: '提交一条产品反馈', reward: 40, type: 'once', done: false, desc: '你的建议会让我们做得更好。' },
]

export const creditRecords: CreditRecord[] = [
  { id: 'r1', source: 'gift', amount: 500, balance: 500, note: '新用户赠送积分', at: '2026-08-01T10:00:00Z' },
  { id: 'r2', source: 'recharge', amount: 1000, balance: 1500, note: '充值 · 永久积分', at: '2026-08-15T11:30:00Z' },
]
