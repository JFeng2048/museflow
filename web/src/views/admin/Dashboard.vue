<script setup lang="ts">
import { ref } from 'vue'
import { NCard, NGrid, NGi, NIcon, NSpace, NTag } from 'naive-ui'
import {
  PeopleOutline,
  BookOutline,
  DocumentTextOutline,
  FlashOutline,
  TrendingUpOutline,
  ServerOutline,
} from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { adminMetrics, adminServices } from '@/mock/admin'
import { formatWords } from '@/utils/format'

const { t } = useI18n()
const metrics = ref(adminMetrics)
const services = ref(adminServices)

const cards = [
  { key: 'totalUsers', label: t('admin.dash.totalUsers'), value: metrics.value.totalUsers, icon: PeopleOutline, color: '#2f6fb3' },
  { key: 'totalNovels', label: t('admin.dash.totalNovels'), value: metrics.value.totalNovels, icon: BookOutline, color: '#b9853f' },
  { key: 'totalWords', label: t('admin.dash.totalWords'), value: formatWords(metrics.value.totalWords), icon: DocumentTextOutline, color: '#46b46a' },
  { key: 'genToday', label: t('admin.dash.genToday'), value: metrics.value.genToday, icon: FlashOutline, color: '#9b6bd6' },
  { key: 'newToday', label: t('admin.dash.newToday'), value: metrics.value.newToday, icon: TrendingUpOutline, color: '#e0883e' },
  { key: 'services', label: t('admin.dash.services'), value: `${metrics.value.servicesOnline}/${metrics.value.servicesTotal}`, icon: ServerOutline, color: '#2f9e8f' },
] as const

const serviceTag: Record<string, { type: 'success' | 'warning' | 'error'; key: string }> = {
  healthy: { type: 'success', key: 'admin.services.healthy' },
  degraded: { type: 'warning', key: 'admin.services.degraded' },
  down: { type: 'error', key: 'admin.services.down' },
}
</script>

<template>
  <div class="admin-page">
    <header class="admin-head">
      <div>
        <h1 class="title">{{ t('admin.dash.title') }}</h1>
        <p class="subtitle">{{ t('admin.dash.subtitle') }}</p>
      </div>
    </header>

    <n-grid :cols="3" :x-gap="18" :y-gap="18" responsive="screen" item-responsive>
      <n-gi v-for="c in cards" :key="c.key" span="3 s:1">
        <n-card :bordered="false" class="metric-card">
          <span class="metric-ico" :style="{ background: c.color }">
            <n-icon :component="c.icon" :size="20" />
          </span>
          <div class="metric-body">
            <span class="metric-val">{{ c.value }}</span>
            <span class="metric-label">{{ c.label }}</span>
          </div>
        </n-card>
      </n-gi>
    </n-grid>

    <div class="admin-grid">
      <n-card :bordered="false" class="trend-card">
        <template #header>{{ t('admin.dash.trend7') }}</template>
        <div class="bars">
          <div v-for="(v, i) in metrics.new7" :key="i" class="bar-col">
            <span class="bar" :style="{ height: (v / Math.max(...metrics.new7)) * 100 + '%' }" />
            <span class="bar-x">D{{ i + 1 }}</span>
          </div>
        </div>
      </n-card>

      <n-card :bordered="false" class="svc-card">
        <template #header>{{ t('admin.dash.serviceHealth') }}</template>
        <n-space vertical :size="10">
          <div v-for="s in services" :key="s.id" class="svc-row">
            <div class="svc-info">
              <strong>{{ s.name }}</strong>
              <small>{{ s.kind }}</small>
            </div>
            <n-tag size="small" :type="serviceTag[s.status].type" :bordered="false">
              {{ t(serviceTag[s.status].key) }}
            </n-tag>
          </div>
          <p v-if="!services.length" class="muted">{{ t('admin.none') }}</p>
        </n-space>
      </n-card>
    </div>
  </div>
</template>
