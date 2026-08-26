<script setup lang="ts">
import { onMounted, computed, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { NIcon, NStatistic, NSpace, NEmpty, NGrid, NGi, NCard } from 'naive-ui'
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
        <div class="stat-card" :style="{ '--accent': it.accent }">
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
              <span class="dist-dot" :style="{ background: it.color }" />
              <span class="dist-label">{{ t(it.labelKey) }}</span>
              <span class="dist-pct">{{ it.pct }}%</span>
            </div>
            <div class="dist-bar">
              <div class="dist-fill" :style="{ width: it.pct + '%', background: it.color }" />
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

<style scoped>
.stats-page {
  padding: 32px 40px 48px;
  max-width: 1080px;
  margin: 0 auto;
}
.stats-eyebrow {
  margin: 0;
  font-size: 13px;
  color: var(--brand, #5b8def);
  letter-spacing: 0.04em;
}
.stats-title {
  margin: 4px 0 2px;
  font-size: 27px;
  font-weight: 650;
  color: var(--ink, #2a2620);
}
.stats-sub {
  margin: 0;
  font-size: 14px;
  color: var(--ink-muted, #8b8478);
}
.stat-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 18px 18px;
  border-radius: 16px;
  background: var(--paper, #fff);
  border: 1px solid var(--line, #ece7df);
  box-shadow: 0 1px 2px rgba(40, 34, 24, 0.04);
}
.stat-icon {
  width: 42px;
  height: 42px;
  flex: none;
  display: grid;
  place-items: center;
  border-radius: 12px;
  color: var(--accent);
  background: color-mix(in srgb, var(--accent) 14%, transparent);
  font-size: 21px;
}
.stat-value {
  font-size: 24px;
  font-weight: 640;
  color: var(--ink, #2a2620);
}
.stats-row {
  display: grid;
  grid-template-columns: 1fr 1.4fr;
  gap: 16px;
  margin-top: 18px;
}
@media (max-width: 880px) {
  .stats-row {
    grid-template-columns: 1fr;
  }
}
.stats-panel :deep(.n-card-header) {
  padding-bottom: 10px;
}
.panel-title {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  font-weight: 620;
  color: var(--ink, #2a2620);
}
.panel-icon {
  font-size: 17px;
  color: var(--brand, #5b8def);
}
.dist {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 6px 2px;
}
.dist-top {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13.5px;
  color: var(--ink, #2a2620);
}
.dist-dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
}
.dist-pct {
  margin-left: auto;
  color: var(--ink-muted, #8b8478);
  font-variant-numeric: tabular-nums;
}
.dist-bar {
  margin-top: 6px;
  height: 8px;
  border-radius: 99px;
  background: var(--warm-2, #f4efe7);
  overflow: hidden;
}
.dist-fill {
  height: 100%;
  border-radius: 99px;
  transition: width 0.5s ease;
}
.week {
  display: flex;
  align-items: flex-end;
  gap: 10px;
  height: 200px;
  padding: 10px 4px 0;
}
.week-col {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 100%;
}
.week-bar-wrap {
  flex: 1;
  width: 100%;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}
.week-bar {
  position: relative;
  width: 60%;
  min-height: 4px;
  border-radius: 8px 8px 4px 4px;
  background: linear-gradient(180deg, var(--brand, #5b8def), color-mix(in srgb, var(--brand, #5b8def) 55%, #fff));
  transition: height 0.5s ease;
  display: flex;
  justify-content: center;
}
.week-val {
  position: absolute;
  top: -18px;
  font-size: 11px;
  color: var(--ink-muted, #8b8478);
  white-space: nowrap;
}
.week-label {
  margin-top: 8px;
  font-size: 12px;
  color: var(--ink-muted, #8b8478);
}
</style>
