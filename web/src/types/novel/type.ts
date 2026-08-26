export type NovelStatus = 'draft' | 'serializing' | 'completed' | 'paused' | 'ongoing'

/** 状态中文标签（界面同时支持 t('novel.status.xxx') 国际化）。 */
export const NOVEL_STATUS_LABEL: Record<NovelStatus, string> = {
  draft: '草稿',
  serializing: '连载中',
  completed: '已完结',
  paused: '已暂停',
  ongoing: '创作中',
}

/** 章节（用于编辑器目录树）。 */
export interface Chapter {
  id: string
  title: string
  words: number
  status: 'draft' | 'written' | 'polished'
  updatedAt: string
  children?: Chapter[]
}

export interface Novel {
  id: string
  /** 所属用户（个人空间隔离）。 */
  userId: string
  title: string
  /** 一句话设定 / 简介。 */
  premise?: string
  description?: string
  genre: string
  status: NovelStatus
  wordCount: number
  chapterCount: number
  /** 每日 / 总字数目标，用于进度激励。 */
  wordGoal?: number
  /** 封面主色（无图时用色块 + 书名首字）。 */
  coverColor?: string
  tags: string[]
  updatedAt: string
  createdAt: string
  chapters?: Chapter[]
}

export interface CreateNovelPayload {
  title: string
  description: string
  genre: string
  tags: string[]
  coverColor?: string
  wordGoal?: number
}

export interface NovelStats {
  totalNovels: number
  totalWords: number
  statusOngoing: number
  statusCompleted: number
  statusDraft: number
  last7Words: number[]
  completionRate: number
}
