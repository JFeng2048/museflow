<script setup lang="ts">
import { ref, h, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NCard, NDataTable, NTag, NButton, NSpace, NEmpty, type DataTableColumns,
} from 'naive-ui'
import { adminLogs } from '@/mock/admin'
import type { AdminLog } from '@/types/admin'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const logs = ref<AdminLog[]>([...adminLogs])

const levelFilter = ref<'all' | 'info' | 'warn' | 'error'>('all')
const levelTag: Record<string, { type: 'info' | 'warning' | 'error'; key: string }> = {
  info: { type: 'info', key: 'admin.logs.info' },
  warn: { type: 'warning', key: 'admin.logs.warn' },
  error: { type: 'error', key: 'admin.logs.error' },
}

const filtered = computed(() =>
  levelFilter.value === 'all'
    ? logs.value
    : logs.value.filter((l) => l.level === levelFilter.value),
)

const levelOptions = [
  { label: t('admin.logs.filterAll'), value: 'all' },
  { label: t('admin.logs.info'), value: 'info' },
  { label: t('admin.logs.warn'), value: 'warn' },
  { label: t('admin.logs.error'), value: 'error' },
]

const columns: DataTableColumns<AdminLog> = [
  { title: t('admin.logs.time'), key: 'time', width: 170, render: (row) => formatDateTime(row.time) },
  {
    title: t('admin.logs.level'), key: 'level', width: 90,
    render: (row) => h(NTag, { size: 'small', type: levelTag[row.level].type, bordered: false }, { default: () => t(levelTag[row.level].key) }),
  },
  { title: t('admin.logs.service'), key: 'service', width: 150 },
  { title: t('admin.logs.actor'), key: 'actor', width: 160 },
  { title: t('admin.logs.message'), key: 'message', render: (row) => row.message },
]
</script>

<template>
  <div class="admin-page">
    <header class="admin-head">
      <div>
        <h1 class="title">{{ t('admin.logs.title') }}</h1>
        <p class="subtitle">{{ t('admin.logs.subtitle') }}</p>
      </div>
      <n-space :size="8">
        <n-button
          v-for="opt in levelOptions"
          :key="opt.value"
          size="small"
          :type="levelFilter === opt.value ? 'primary' : 'default'"
          @click="levelFilter = opt.value as any"
        >
          {{ opt.label }}
        </n-button>
      </n-space>
    </header>

    <n-card :bordered="false" class="table-card">
      <n-empty v-if="!filtered.length" :description="t('admin.logs.empty')" />
      <n-data-table v-else :columns="columns" :data="filtered" :row-key="(r) => r.id" :pagination="{ pageSize: 10 }" />
    </n-card>
  </div>
</template>
