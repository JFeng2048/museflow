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
  accent: string
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
  { key: 'totalNovels', labelKey: 'stats.kpiNovels', icon: BookOutline, accent: '#5B8DEF' },
  { key: 'totalWords', labelKey: 'stats.kpiWords', icon: DocumentTextOutline, accent: '#7C6FF0' },
  { key: 'statusOngoing', labelKey: 'stats.kpiOngoing', icon: PulseOutline, accent: '#E0A458' },
  { key: 'statusCompleted', labelKey: 'stats.kpiCompleted', icon: CheckmarkDoneOutline, accent: '#3FA37A' },
  { key: 'statusDraft', labelKey: 'stats.kpiDraft', icon: PencilOutline, accent: '#C77D8B' },
]

/** 状态分布配置。 */
export const STATUS_DIST = [
  { key: 'statusOngoing', labelKey: 'stats.statusOngoing', color: '#E0A458' },
  { key: 'statusCompleted', labelKey: 'stats.statusCompleted', color: '#3FA37A' },
  { key: 'statusDraft', labelKey: 'stats.statusDraft', color: '#C77D8B' },
] as const

export { TrendingUpOutline }
