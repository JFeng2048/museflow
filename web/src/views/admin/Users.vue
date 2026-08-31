<script setup lang="ts">
import { ref, h, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NCard, NDataTable, NTag, NButton, NIcon, NSwitch, NInput, NSelect,
  useMessage, type DataTableColumns,
} from 'naive-ui'
import { SearchOutline } from '@vicons/ionicons5'
import { listUsers, updateUserStatus, assignRole, listRoles } from '@/api/admin'
import type { AdminUser, AdminUserStatusValue, AdminRole } from '@/types/admin'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const message = useMessage()

const users = ref<AdminUser[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

/** 搜索关键字与状态筛选。 */
const keyword = ref('')
const statusFilter = ref<AdminUserStatusValue | null>(null)

/** 可选角色：由后端角色列表提供，默认回落到内置两种。 */
const roleOptions = ref<{ label: string; value: string }[]>([])
const roleCodeByValue = ref<Record<string, string>>({})

const statusOptions = [
  { label: t('admin.users.statusNormal'), value: 1 },
  { label: t('admin.users.statusFrozen'), value: 2 },
  { label: t('admin.users.statusDeleted'), value: 3 },
  { label: t('admin.users.statusPending'), value: 4 },
]

const statusTagType: Record<number, 'success' | 'warning' | 'error' | 'info'> = {
  1: 'success',
  2: 'warning',
  3: 'error',
  4: 'info',
}
const statusLabelKey: Record<number, string> = {
  1: 'admin.users.statusNormal',
  2: 'admin.users.statusFrozen',
  3: 'admin.users.statusDeleted',
  4: 'admin.users.statusPending',
}

async function loadRoles() {
  try {
    const roles = await listRoles()
    roleOptions.value = roles.map((r: AdminRole) => ({ label: r.name, value: r.code }))
    roleCodeByValue.value = Object.fromEntries(roles.map((r) => [r.code, r.name]))
  } catch {
    // 角色列表拉取失败不阻塞用户列表，退化成显示原始编码
    roleOptions.value = []
  }
}

async function load() {
  loading.value = true
  try {
    const res = await listUsers({
      page: page.value,
      pageSize: pageSize.value,
      keyword: keyword.value || undefined,
      status: statusFilter.value ?? undefined,
      orderBy: 'created_at',
      desc: true,
    })
    users.value = res.items
    total.value = res.total
  } catch (e: any) {
    message.error(e?.message || t('admin.users.loadFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadRoles()
  load()
})

function onSearch() {
  page.value = 1
  load()
}

/** 冻结 / 解冻：后端只有「正常」与「冻结」两种可逆状态。 */
async function toggleStatus(row: AdminUser) {
  const next: AdminUserStatusValue = row.status === 2 ? 1 : 2
  try {
    await updateUserStatus(row.id, next)
    row.status = next
    message.success(next === 1 ? t('admin.users.enabled') : t('admin.users.disabled'))
  } catch (e: any) {
    message.error(e?.message || t('admin.users.updateFailed'))
  }
}

async function onAssignRole(row: AdminUser, code: string) {
  try {
    await assignRole(row.id, code)
    row.roles = [code]
    message.success(t('admin.users.roleUpdated'))
  } catch (e: any) {
    message.error(e?.message || t('admin.users.updateFailed'))
  }
}

function primaryRole(row: AdminUser): string {
  return row.roles[0] || ''
}

const columns: DataTableColumns<AdminUser> = [
  { title: t('admin.users.name'), key: 'name' },
  { title: t('admin.users.email'), key: 'email' },
  {
    title: t('admin.users.role'), key: 'role', width: 150,
    render: (row) =>
      h(NSelect, {
        size: 'small',
        value: primaryRole(row) || null,
        options: roleOptions.value,
        consistentMenuWidth: false,
        onUpdateValue: (v: string) => onAssignRole(row, v),
      }),
  },
  {
    title: t('admin.users.status'), key: 'status', width: 120,
    render: (row) =>
      h(NSwitch, {
        size: 'small',
        // 已注销 / 待审核不可通过开关切换，仅展示状态
        disabled: row.status === 3 || row.status === 4,
        value: row.status === 1,
        onUpdateValue: () => toggleStatus(row),
      }),
  },
  {
    title: t('admin.users.state'), key: 'state', width: 100,
    render: (row) =>
      h(NTag, { size: 'small', type: statusTagType[row.status] || 'info', bordered: false }, {
        default: () => t(statusLabelKey[row.status] || 'admin.users.statusNormal'),
      }),
  },
  {
    title: t('admin.users.createdAt'), key: 'createdAt', width: 170,
    render: (row) => formatDateTime(row.createdAt),
  },
  {
    title: t('admin.users.lastActive'), key: 'updatedAt', width: 170,
    render: (row) => formatDateTime(row.updatedAt),
  },
]
</script>

<template>
  <div class="admin-page">
    <header class="admin-head with-action">
      <div>
        <h1 class="title">{{ t('admin.users.title') }}</h1>
        <p class="subtitle">{{ t('admin.users.subtitle') }}</p>
      </div>
      <div class="head-tools">
        <n-input
          v-model:value="keyword"
          :placeholder="t('admin.users.searchPh')"
          clearable
          style="width: 220px"
          @keyup.enter="onSearch"
        >
          <template #prefix><n-icon :component="SearchOutline" /></template>
        </n-input>
        <n-select
          v-model:value="statusFilter"
          :options="statusOptions"
          :placeholder="t('admin.users.status')"
          clearable
          style="width: 140px"
          @update:value="onSearch"
        />
        <n-button type="primary" :loading="loading" @click="onSearch">
          {{ t('admin.users.search') }}
        </n-button>
      </div>
    </header>

    <n-card :bordered="false" class="table-card">
      <n-data-table
        :columns="columns"
        :data="users"
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
</style>
