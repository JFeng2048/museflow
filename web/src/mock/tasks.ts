import type { GenerationTask } from '@/types'

function minutesAgo(n: number): string {
  return new Date(Date.now() - n * 60000).toISOString()
}

export const tasks: GenerationTask[] = [
  {
    id: 'tk-1',
    novelId: 'nv-1001',
    novelTitle: '星海拾遗者',
    type: 'continue',
    prompt: '续写第二卷第五章，让林深在走廊尽头发现一扇会呼吸的门。',
    status: 'running',
    progress: 64,
    words: 0,
    createdAt: minutesAgo(8),
  },
  {
    id: 'tk-2',
    novelId: 'nv-1002',
    novelTitle: '长安十二夜',
    type: 'outline',
    prompt: '为第三夜生成三条悬疑支线大纲。',
    status: 'success',
    progress: 100,
    words: 1280,
    createdAt: minutesAgo(120),
    finishedAt: minutesAgo(112),
  },
  {
    id: 'tk-3',
    novelId: 'nv-1003',
    novelTitle: '我的便利店通异界',
    type: 'rewrite',
    prompt: '把第 88 章团购节的对话改得更轻快一些。',
    status: 'success',
    progress: 100,
    words: 960,
    createdAt: minutesAgo(300),
    finishedAt: minutesAgo(290),
  },
  {
    id: 'tk-4',
    novelId: 'nv-1004',
    novelTitle: '深海来信',
    type: 'expand',
    prompt: '扩写序章中潜航器下潜的感官描写。',
    status: 'failed',
    progress: 30,
    words: 0,
    createdAt: minutesAgo(420),
    finishedAt: minutesAgo(410),
  },
  {
    id: 'tk-5',
    novelId: 'nv-1001',
    novelTitle: '星海拾遗者',
    type: 'polish',
    prompt: '润色第一章开头三段，强化氛围。',
    status: 'pending',
    progress: 0,
    words: 0,
    createdAt: minutesAgo(2),
  },
]
