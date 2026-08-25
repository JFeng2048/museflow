import { characters } from './characters'
import type { WorldSetting, Foreshadow } from '@/types'

function daysAgo(n: number): string {
  return new Date(Date.now() - n * 86400000).toISOString()
}

export const worlds: WorldSetting[] = [
  {
    id: 'wd-1',
    novelId: 'nv-1001',
    name: '废弃星域 · 残响带',
    category: '地理',
    summary: '文明湮灭后漂浮着记忆晶体的星区，信号微弱却绵延数光年。',
    details: '残响带由无数破碎空间站与晶体残片构成，拾遗号在此打捞文明遗物。晶体需低温封存，否则记忆会逸散。',
  },
  {
    id: 'wd-2',
    novelId: 'nv-1001',
    name: '记忆晶体法则',
    category: '规则',
    summary: '晶体记录逝者意识片段，可被唤醒但无法长时间共存于现实。',
    details: '唤醒后的意识会随时间衰减，且与唤醒者产生情感共振，过度接触会导致唤醒者记忆混淆。',
  },
  {
    id: 'wd-3',
    novelId: 'nv-1002',
    name: '天宝年间的长安',
    category: '历史',
    summary: '繁盛表象下暗流涌动，坊市制度与宵禁塑造了案件的时空边界。',
    details: '朱雀大街为南北中轴，案发多在夜间，受宵禁限制，凶手必须利用坊门与地下渠道移动。',
  },
  {
    id: 'wd-4',
    novelId: 'nv-1003',
    name: '异界枢纽协议',
    category: '规则',
    summary: '便利店作为两界枢纽，补货清单决定异界可兑换的物资。',
    details: '每周三凌晨是“窗口期”，此时两个世界的物价与重力短暂对齐，是转运珍稀物资的唯一时机。',
  },
]

export const foreshadows: Foreshadow[] = [
  {
    id: 'fs-1',
    novelId: 'nv-1001',
    clue: '林深在首章捡到的晶体里，有一段不属于任何已知文明的童谣。',
    revealChapter: '第二卷 第五章',
    status: 'revealing',
    note: '童谣将在走廊尽头那扇“会呼吸的门”后揭晓，指向拾遗的真实来历。',
  },
  {
    id: 'fs-2',
    novelId: 'nv-1002',
    clue: '县尉书房多出一枚不属于他的玉佩。',
    revealChapter: '第六夜',
    status: 'planted',
    note: '玉佩是刺客故意留下的诱饵，也是反向追踪的线索。',
  },
  {
    id: 'fs-3',
    novelId: 'nv-1003',
    clue: '异界顾客总在周三凌晨购买同一款关东煮。',
    revealChapter: '第 60 章',
    status: 'resolved',
    note: '关东煮是异界“锚点食物”，用来稳定穿越者的现实认知。',
  },
  {
    id: 'fs-4',
    novelId: 'nv-1004',
    clue: '深海来信的署名，与一个已注销的深潜项目编号一致。',
    revealChapter: '第七章',
    status: 'planted',
    note: '暗示来信者并非人类，而是项目遗留的自主意识。',
  },
]

export const loreCount = characters.length + worlds.length + foreshadows.length
