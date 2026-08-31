<script setup lang="ts">
import { ref, h, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NCard, NDataTable, NTag, NButton, NInput, NEmpty, type DataTableColumns,
} from 'naive-ui'
import { listAuditLogs } from '@/api/admin'
import type { AdminAuditLog } from '@/types/admin'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()

const logs = ref<AdminAuditLog[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

/** 按操作类型过滤（后端 audit-logs 支持 action 过滤）。 */
const actionFilter = ref('')

async function load() {
  loading.value = true
  try {
    const res = await listAuditLogs({
      page: page.value,
      pageSize: pageSize.value,
      action: actionFilter.value || undefined,
    })
    logs.value = res.items
    total.value = res.total
  } catch {
    // 审计日志属于运营辅助信息，失败时保持空列表即可
    logs.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

onMounted(load)

function onSearch() {
  page.value = 1
  load()
}

/** 常见操作类型，用于快速过滤。 */
const quickActions = ['login', 'logout', 'change_password', 'assign_role']

const actionTagType: Record<string, 'default' | 'info' | 'success' | 'warning'> = {
  login: 'info',
  logout: 'default',
  change_password: 'warning',
  assign_role: 'success',
}

/** 操作人为空表示系统操作。 */
function actorLabel(row: AdminAuditLog): string {
  return row.actor || t('admin.logs.system')
}

const filtered = computed(() => logs.value)

const columns: DataTableColumns<AdminAuditLog> = [
  { title: t('admin.logs.time'), key: 'time', width: 170, render: (row) => formatDateTime(row.time) },
  {
    title: t('admin.logs.action'), key: 'action', width: 160,
    render: (row) =>
      h(NTag, { size: 'small', type: actionTagType[row.action] || 'default', bordered: false }, {
        default: () => row.action,
      }),
  },
  { title: t('admin.logs.service'), key: 'resource', width: 130 },
  { title: t('admin.logs.actor'), key: 'actor', width: 280, render: (row) => actorLabel(row) },
  { title: t('admin.logs.ip'), key: 'ip', width: 140 },
  { title: t('admin.logs.message'), key: 'detail', render: (row) => row.detail || '-' },
]
</script>

<template>
  <div class="admin-page">
    <header class="admin-head">
      <div>
        <h1 class="title">{{ t('admin.logs.title') }}</h1>
        <p class="subtitle">{{ t('admin.logs.subtitle') }}</p>
      </div>
      <div class="head-tools">
        <n-input
          v-model:value="actionFilter"
          :placeholder="t('admin.logs.actionPh')"
          clearable
          style="width: 220px"
          @keyup.enter="onSearch"
        />
        <n-button type="primary" :loading="loading" @click="onSearch">
          {{ t('admin.logs.search') }}
        </n-button>
      </div>
    </header>

    <div class="quick-row">
      <n-button
        v-for="a in quickActions"
        :key="a"
        size="small"
        :type="actionFilter === a ? 'primary' : 'default'"
        @click="actionFilter = actionFilter === a ? '' : a; onSearch()"
      >
        {{ a }}
      </n-button>
    </div>

    <n-card :bordered="false" class="table-card">
      <n-empty v-if="!loading && !filtered.length" :description="t('admin.logs.empty')" />
      <n-data-table
        v-else
        :columns="columns"
        :data="filtered"
        :loading="loading"
        :row-key="(r) => r.id"
        :remote="true"
        :pagination="{
          page: page,
          pageSize: pageSize,
          itemCount: total,
          showSizePicker: true,
          pageSizes: [10, 20, 50],
        }"
        @update:page="(p: number) => { page = p; load() }"
        @update:page-size="(s: number) => { pageSize = s; page = 1; load() }"
      />
    </n-card>
  </div>
</template>

<style scoped>
.head-tools {
  display: flex;
  align-items: center;
  gap: 10px;
}
.quick-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: -8px 0 14px;
}
</style>
