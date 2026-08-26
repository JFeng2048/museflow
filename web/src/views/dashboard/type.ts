import type { Novel } from '@/types'

export interface DashboardStat {
  totalNovels: number
  totalWords: number
  ongoing: number
  completed: number
}

export type RecentNovel = Novel
