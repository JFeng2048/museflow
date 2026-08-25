import type { Material } from '@/types'

function daysAgo(n: number): string {
  return new Date(Date.now() - n * 86400000).toISOString()
}

export const materials: Material[] = [
  {
    id: 'mt-1',
    title: '《三体》黑暗森林法则摘录',
    type: 'quote',
    source: '刘慈欣《三体II》',
    content: '宇宙就是一座黑暗森林，每个文明都是带枪的猎人……',
    tags: ['科幻', '世界观', '冲突'],
    imported: false,
    createdAt: daysAgo(3),
  },
  {
    id: 'mt-2',
    title: '唐代坊市制度考',
    type: 'world',
    source: '历史资料库',
    content: '长安城实行严格的坊市分离，夜间实行宵禁，朱雀大街为南北中轴。',
    tags: ['唐朝', '城市', '制度'],
    imported: true,
    createdAt: daysAgo(8),
  },
  {
    id: 'mt-3',
    title: '反派动机：被背叛的守护者',
    type: 'plot',
    source: '创作灵感',
    content: '反派曾是秩序的守护者，因一次误判导致挚友牺牲，从此走向极端。',
    tags: ['人物弧光', '反转'],
    imported: false,
    createdAt: daysAgo(5),
  },
  {
    id: 'mt-4',
    title: '深海热泉生态系统',
    type: 'world',
    source: '科普文章',
    content: '海底热泉周围存在不依赖阳光的生态系统，以化能合成细菌为基座。',
    tags: ['深海', '硬科幻', '生态'],
    imported: false,
    createdAt: daysAgo(12),
  },
  {
    id: 'mt-5',
    title: '便利店爆款陈列技巧',
    type: 'image',
    source: '零售观察',
    content: '收银台黄金视线区放置冲动消费品，能提升 18% 客单价。',
    tags: ['日常', '经营'],
    imported: true,
    createdAt: daysAgo(20),
  },
  {
    id: 'mt-6',
    title: '群像戏视角切换模板',
    type: 'plot',
    source: '写作方法',
    content: '以场景为单位切换 POV，每章聚焦一个角色的内心，章末留钩子。',
    tags: ['叙事', '技巧'],
    imported: false,
    createdAt: daysAgo(2),
  },
]
