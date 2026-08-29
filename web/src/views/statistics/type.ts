import type { Component } from 'vue'
import {
  BookOutline,
  DocumentTextOutline,
  PulseOutline,
  CheckmarkDoneOutline,
  PencilOutline,
  TrendingUpOutline,
} from '@vicons/ionicons5'

/** 统计卡片项。 */
export interface StatItem {
  key: keyof Statistics
  labelKey: string
  icon: Component
  /** 主题色 token 名（--c-accent-*），不写硬编码 hex。 */
  accentVar: string
}

/** 统计数据模型。 */
export interface Statistics {
  totalNovels: number
  totalWords: number
  statusOngoing: number
  statusCompleted: number
  statusDraft: number
  last7Words: number[]
}

/** 周维度标签（i18n 在视图中按顺序解析）。 */
export const WEEK_LABEL_KEYS = [
  'stats.dayMon',
  'stats.dayTue',
  'stats.dayWed',
  'stats.dayThu',
  'stats.dayFri',
  'stats.daySat',
  'stats.daySun',
] as const

/** 顶部 KPI 卡片配置。 */
export const STAT_KPIS: StatItem[] = [
  { key: 'totalNovels',    labelKey: 'stats.kpiNovels',    icon: BookOutline,            accentVar: '--c-accent-blue'   },
  { key: 'totalWords',     labelKey: 'stats.kpiWords',     icon: DocumentTextOutline,    accentVar: '--c-accent-violet' },
  { key: 'statusOngoing',  labelKey: 'stats.kpiOngoing',   icon: PulseOutline,           accentVar: '--c-accent-amber'  },
  { key: 'statusCompleted',labelKey: 'stats.kpiCompleted', icon: CheckmarkDoneOutline,   accentVar: '--c-accent-green'  },
  { key: 'statusDraft',    labelKey: 'stats.kpiDraft',     icon: PencilOutline,          accentVar: '--c-accent-rose'   },
]

/** 状态分布配置。 */
export type StatusKey = 'statusOngoing' | 'statusCompleted' | 'statusDraft'

export interface DistItem {
  key: StatusKey
  labelKey: string
  /** 主题色 token 名（--c-accent-*），不写硬编码 hex。 */
  colorVar: string
}

/** 状态分布配置。 */
export const STATUS_DIST: DistItem[] = [
  { key: 'statusOngoing',   labelKey: 'stats.statusOngoing',   colorVar: '--c-accent-amber' },
  { key: 'statusCompleted', labelKey: 'stats.statusCompleted', colorVar: '--c-accent-green' },
  { key: 'statusDraft',     labelKey: 'stats.statusDraft',     colorVar: '--c-accent-rose'  },
]

export { TrendingUpOutline }
