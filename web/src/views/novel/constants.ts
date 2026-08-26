import type { NovelStatus } from '@/types'

export const NOVEL_STATUS_TABS: { labelKey: string; value: NovelStatus | 'all' }[] = [
  { labelKey: 'novel.tabAll', value: 'all' },
  { labelKey: 'novel.statusOngoing', value: 'ongoing' },
  { labelKey: 'novel.statusSerializing', value: 'serializing' },
  { labelKey: 'novel.statusCompleted', value: 'completed' },
  { labelKey: 'novel.statusPaused', value: 'paused' },
  { labelKey: 'novel.statusDraft', value: 'draft' },
]

export const NOVEL_STATUS_META: Record<
  NovelStatus,
  { labelKey: string; type: 'default' | 'success' | 'warning' | 'error' | 'info' }
> = {
  draft: { labelKey: 'novel.statusDraft', type: 'default' },
  ongoing: { labelKey: 'novel.statusOngoing', type: 'info' },
  serializing: { labelKey: 'novel.statusSerializing', type: 'info' },
  completed: { labelKey: 'novel.statusCompleted', type: 'success' },
  paused: { labelKey: 'novel.statusPaused', type: 'warning' },
}

export const NOVEL_GENRES: { labelKey: string; value: string }[] = [
  { labelKey: 'novel.genreSciFi', value: 'sci-fi' },
  { labelKey: 'novel.genreFantasy', value: 'fantasy' },
  { labelKey: 'novel.genreUrban', value: 'urban' },
  { labelKey: 'novel.genreHistory', value: 'history' },
  { labelKey: 'novel.genreRomance', value: 'romance' },
  { labelKey: 'novel.genreMystery', value: 'mystery' },
  { labelKey: 'novel.genreWuxia', value: 'wuxia' },
  { labelKey: 'novel.genreLight', value: 'light' },
]
