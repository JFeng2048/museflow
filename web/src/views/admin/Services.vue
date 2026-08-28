<script setup lang="ts">
import { ref, h, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NCard, NGrid, NGi, NIcon, NTag, NButton, NProgress, NSpace, useMessage,
} from 'naive-ui'
import { ReloadOutline, ServerOutline } from '@vicons/ionicons5'
import { adminServices } from '@/mock/admin'
import type { AdminService } from '@/types/admin'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const message = useMessage()
const services = ref<AdminService[]>([...adminServices])

const statusTag: Record<string, { type: 'success' | 'warning' | 'error'; key: string }> = {
  healthy: { type: 'success', key: 'admin.services.healthy' },
  degraded: { type: 'warning', key: 'admin.services.degraded' },
  down: { type: 'error', key: 'admin.services.down' },
}

const online = computed(() => services.value.filter((s) => s.status !== 'down').length)

function refresh() {
  // mock：刷新检测时间
  services.value.forEach((s) => (s.checkedAt = new Date().toISOString()))
  message.success(t('admin.services.refreshed'))
}
</script>

<template>
  <div class="admin-page">
    <header class="admin-head with-action">
      <div>
        <h1 class="title">{{ t('admin.services.title') }}</h1>
        <p class="subtitle">{{ t('admin.services.subtitle') }}</p>
      </div>
      <n-space :size="12">
        <span class="svc-summary">
          {{ t('admin.services.online') }}: <strong>{{ online }}</strong> / {{ t('admin.services.total') }}: {{ services.length }}
        </span>
        <n-button @click="refresh">
          <template #icon><n-icon :component="ReloadOutline" /></template>
          {{ t('admin.services.refresh') }}
        </n-button>
      </n-space>
    </header>

    <n-grid :cols="2" :x-gap="18" :y-gap="18" responsive="screen" item-responsive>
      <n-gi v-for="s in services" :key="s.id" span="2 s:1">
        <n-card :bordered="false" class="svc-card">
          <div class="svc-top">
            <span class="svc-ico"><n-icon :component="ServerOutline" :size="18" /></span>
            <div class="svc-meta">
              <strong>{{ s.name }}</strong>
              <small>{{ s.kind }}</small>
            </div>
            <n-tag size="small" :type="statusTag[s.status].type" :bordered="false">
              {{ t(statusTag[s.status].key) }}
            </n-tag>
          </div>

          <div class="svc-metrics">
            <div class="svc-metric">
              <span class="svc-metric-label">{{ t('admin.services.instances') }}</span>
              <span class="svc-metric-val">{{ s.instances }}</span>
            </div>
            <div class="svc-metric">
              <span class="svc-metric-label">{{ t('admin.services.latency') }}</span>
              <span class="svc-metric-val">{{ s.status === 'down' ? '—' : s.latency + ' ' + t('admin.services.ms') }}</span>
            </div>
          </div>

          <div class="svc-bars">
            <div class="svc-bar">
              <div class="svc-bar-head">
                <span>{{ t('admin.services.cpu') }}</span><span>{{ s.cpu }}%</span>
              </div>
              <n-progress
                type="line"
                :percentage="s.cpu"
                :height="6"
                :show-indicator="false"
                :color="s.cpu > 70 ? '#e0883e' : '#2f9e8f'"
              />
            </div>
            <div class="svc-bar">
              <div class="svc-bar-head">
                <span>{{ t('admin.services.memory') }}</span><span>{{ s.memory }}%</span>
              </div>
              <n-progress
                type="line"
                :percentage="s.memory"
                :height="6"
                :show-indicator="false"
                :color="s.memory > 70 ? '#e0883e' : '#2f9e8f'"
              />
            </div>
          </div>

          <div class="svc-foot">
            <span class="svc-endpoint" v-if="s.endpoint">{{ s.endpoint }}</span>
            <span class="svc-checked">{{ t('admin.services.checkedAt') }}: {{ formatDateTime(s.checkedAt) }}</span>
          </div>
        </n-card>
      </n-gi>
    </n-grid>
  </div>
</template>
