import type { Novel } from '@/types'

function daysAgo(n: number): string {
  return new Date(Date.now() - n * 86400000).toISOString()
}

export const novels: Novel[] = [
  {
    id: 'nv-1001',
    title: '星海拾遗者',
    description: '一艘考古船在废弃星域打捞文明残片，却在记忆晶体里唤醒了一个沉睡千年的意识。',
    genre: '太空歌剧',
    status: 'serializing',
    wordCount: 186500,
    chapterCount: 48,
    tags: ['星际', '悬疑', '群像'],
    updatedAt: daysAgo(1),
    createdAt: daysAgo(120),
    chapters: [
      { id: 'ch-1', title: '第一卷 · 残响', words: 32000, status: 'polished', updatedAt: daysAgo(40), children: [
        { id: 'ch-1-1', title: '第一章 漂流的信标', words: 4200, status: 'polished', updatedAt: daysAgo(40) },
        { id: 'ch-1-2', title: '第二章 晶体里的声音', words: 3900, status: 'polished', updatedAt: daysAgo(38) },
        { id: 'ch-1-3', title: '第三章 不眠的船员', words: 4100, status: 'written', updatedAt: daysAgo(35) },
      ] },
      { id: 'ch-2', title: '第二卷 · 深空回廊', words: 58000, status: 'written', updatedAt: daysAgo(10), children: [
        { id: 'ch-2-1', title: '第四章 引力陷阱', words: 4600, status: 'written', updatedAt: daysAgo(12) },
        { id: 'ch-2-2', title: '第五章 没有出口的走廊', words: 4400, status: 'draft', updatedAt: daysAgo(10) },
      ] },
      { id: 'ch-3', title: '第三卷 · 拾遗者', words: 96500, status: 'draft', updatedAt: daysAgo(1) },
    ],
  },
  {
    id: 'nv-1002',
    title: '长安十二夜',
    description: '天宝年间的长安城，一个落魄县尉要在十二个夜晚内找出潜伏的刺客与失窃的传国玉玺。',
    genre: '历史悬疑',
    status: 'draft',
    wordCount: 42600,
    chapterCount: 12,
    tags: ['唐朝', '探案', '权谋'],
    updatedAt: daysAgo(4),
    createdAt: daysAgo(28),
    chapters: [
      { id: 'ch-1', title: '第一夜 灯下黑', words: 3800, status: 'written', updatedAt: daysAgo(4) },
      { id: 'ch-2', title: '第二夜 雪落朱雀街', words: 3600, status: 'draft', updatedAt: daysAgo(5) },
    ],
  },
  {
    id: 'nv-1003',
    title: '我的便利店通异界',
    description: '一家 24 小时便利店成了两个世界的枢纽，老板在补货和拯救世界之间反复横跳。',
    genre: '轻小说',
    status: 'completed',
    wordCount: 312000,
    chapterCount: 120,
    tags: ['日常', '奇幻', '搞笑'],
    updatedAt: daysAgo(20),
    createdAt: daysAgo(300),
    chapters: [
      { id: 'ch-1', title: '第 1 章 开门营业', words: 2800, status: 'polished', updatedAt: daysAgo(200) },
      { id: 'ch-2', title: '第 88 章 异界团购节', words: 3100, status: 'polished', updatedAt: daysAgo(30) },
    ],
  },
  {
    id: 'nv-1004',
    title: '深海来信',
    description: '一封来自马里亚纳海沟的邮件，把海洋生物学家卷入一场关于人类起源的惊天秘密。',
    genre: '科幻',
    status: 'paused',
    wordCount: 78500,
    chapterCount: 22,
    tags: ['深海', '硬科幻'],
    updatedAt: daysAgo(60),
    createdAt: daysAgo(150),
    chapters: [
      { id: 'ch-1', title: '序章 一万米之下', words: 3400, status: 'written', updatedAt: daysAgo(60) },
    ],
  },
]
