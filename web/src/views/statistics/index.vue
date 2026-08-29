<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NIcon, NStatistic, NEmpty, NGrid, NGi, NCard } from 'naive-ui'
import { TrendingUpOutline } from '@vicons/ionicons5'
import { useNovelStore } from '@/stores/novel'
import { useUserStore } from '@/stores/system/user'
import { STAT_KPIS, STATUS_DIST, WEEK_LABEL_KEYS } from './type'

const { t } = useI18n()
const novelStore = useNovelStore()
const userStore = useUserStore()

const userId = computed(() => userStore.user?.id || 'u_demo')

onMounted(async () => {
  if (!novelStore.novels.length) await novelStore.loadNovels()
})

// 防御性取值：避免某个 key 缺失时整页崩溃。
const stats = computed(() => {
  const raw = novelStore.userStats(userId.value) || {}
  return {
    totalNovels: raw.totalNovels ?? 0,
    totalWords: raw.totalWords ?? 0,
    statusOngoing: raw.statusOngoing ?? 0,
    statusCompleted: raw.statusCompleted ?? 0,
    statusDraft: raw.statusDraft ?? 0,
    last7Words: Array.isArray(raw.last7Words) ? raw.last7Words : [0, 0, 0, 0, 0, 0, 0],
  }
})

const kpi = computed(() =>
  STAT_KPIS.map((it) => ({ ...it, value: stats.value[it.key] ?? 0 })),
)

const statusSum = computed(() =>
  STATUS_DIST.reduce((s, it) => s + (stats.value[it.key] ?? 0), 0),
)

const dist = computed(() =>
  STATUS_DIST.map((it) => {
    const value = stats.value[it.key] ?? 0
    const total = statusSum.value || 1
    return { ...it, value, pct: Math.round((value / total) * 100) }
  }),
)

const week = computed(() => {
  const days = WEEK_LABEL_KEYS.map((k) => t(k))
  const base = stats.value.last7Words
  const max = Math.max(1, ...base)
  return days.map((d, i) => ({
    d,
    v: base[i] ?? 0,
    pct: Math.round(((base[i] ?? 0) / max) * 100),
  }))
})
</script>

<template>
  <div class="stats-page">
    <header class="stats-head">
      <div>
        <p class="stats-eyebrow">{{ t('stats.eyebrow') }}</p>
        <h2 class="stats-title">{{ t('stats.title') }}</h2>
        <p class="stats-sub">{{ t('stats.subtitle') }}</p>
      </div>
    </header>

    <n-grid cols="2 640:3 960:5" :x-gap="14" :y-gap="14" responsive="screen">
      <n-gi v-for="it in kpi" :key="it.key">
        <div class="stat-card" :style="{ '--accent': `var(${it.accentVar})` }">
          <div class="stat-icon">
            <n-icon :component="it.icon" />
          </div>
          <n-statistic :label="t(it.labelKey)" tabular-nums>
            <span class="stat-value">{{ it.value.toLocaleString() }}</span>
          </n-statistic>
        </div>
      </n-gi>
    </n-grid>

    <div class="stats-row">
      <n-card :bordered="false" class="stats-panel">
        <template #header>
          <span class="panel-title">{{ t('stats.statusDist') }}</span>
        </template>
        <div v-if="statusSum" class="dist">
          <div v-for="it in dist" :key="it.key" class="dist-row">
            <div class="dist-top">
              <span class="dist-dot" :style="{ background: `var(${it.colorVar})` }" />
              <span class="dist-label">{{ t(it.labelKey) }}</span>
              <span class="dist-pct">{{ it.pct }}%</span>
            </div>
            <div class="dist-bar">
              <div class="dist-fill" :style="{ width: it.pct + '%', background: `var(${it.colorVar})` }" />
            </div>
          </div>
        </div>
        <n-empty v-else :description="t('common.empty')" />
      </n-card>

      <n-card :bordered="false" class="stats-panel">
        <template #header>
          <span class="panel-title">
            <n-icon :component="TrendingUpOutline" class="panel-icon" />
            {{ t('stats.weekTrend') }}
          </span>
        </template>
        <div class="week">
          <div v-for="w in week" :key="w.d" class="week-col">
            <div class="week-bar-wrap">
              <div class="week-bar" :style="{ height: w.pct + '%' }">
                <span class="week-val">{{ w.v }}</span>
              </div>
            </div>
            <span class="week-label">{{ w.d }}</span>
          </div>
        </div>
      </n-card>
    </div>
  </div>
</template>
