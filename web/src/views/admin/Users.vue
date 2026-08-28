<script setup lang="ts">
import { ref, h, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NCard, NDataTable, NTag, NButton, NIcon, NSwitch, NModal, NForm, NFormItem,
  NInput, NSelect, useMessage, type DataTableColumns,
} from 'naive-ui'
import { AddOutline } from '@vicons/ionicons5'
import { adminUsers, createAdminUser } from '@/mock/admin'
import type { AdminUser } from '@/types/admin'
import type { UserRole as Role } from '@/types/system/auth'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const message = useMessage()
const users = ref<AdminUser[]>([...adminUsers])

function toggleStatus(row: AdminUser) {
  row.status = row.status === 'active' ? 'disabled' : 'active'
  message.success(row.status === 'active' ? t('admin.users.enabled') : t('admin.users.disabled'))
}

function setRole(row: AdminUser) {
  if (row.role === 'admin') return
  row.role = row.role === 'writer' ? 'admin' : 'writer'
  message.success(t('admin.users.roleUpdated'))
}

const roleOptions = [
  { label: t('admin.roleWriter'), value: 'writer' as Role },
  { label: t('admin.roleAdmin'), value: 'admin' as Role },
]

const columns: DataTableColumns<AdminUser> = [
  { title: t('admin.users.name'), key: 'name', render: (row) => row.name },
  { title: t('admin.users.email'), key: 'email' },
  {
    title: t('admin.users.role'), key: 'role',
    render: (row) =>
      h(
        NButton,
        { quaternary: true, size: 'small', disabled: row.role === 'admin', onClick: () => setRole(row) },
        { default: () => (row.role === 'admin' ? t('admin.roleAdmin') : t('admin.roleWriter')) },
      ),
  },
  {
    title: t('admin.users.status'), key: 'status',
    render: (row) =>
      h(NSwitch, {
        size: 'small',
        value: row.status === 'active',
        onUpdateValue: () => toggleStatus(row),
      }),
  },
  { title: t('admin.users.novels'), key: 'novelCount' },
  { title: t('admin.users.words'), key: 'totalWords', render: (row) => row.totalWords.toLocaleString() },
  { title: t('admin.users.credits'), key: 'credits' },
  { title: t('admin.users.lastActive'), key: 'lastActiveAt', render: (row) => formatDateTime(row.lastActiveAt) },
]

// 创建用户弹窗
const showCreate = ref(false)
const form = reactive({ name: '', email: '', password: '', role: 'writer' as Role })

function openCreate() {
  form.name = ''
  form.email = ''
  form.password = ''
  form.role = 'writer'
  showCreate.value = true
}

function submitCreate() {
  if (!form.name || !form.email || !form.password) {
    message.warning(t('admin.users.namePh'))
    return
  }
  createAdminUser({ ...form })
  users.value = [...adminUsers]
  showCreate.value = false
  message.success(t('admin.users.created'))
}
</script>

<template>
  <div class="admin-page">
    <header class="admin-head with-action">
      <div>
        <h1 class="title">{{ t('admin.users.title') }}</h1>
        <p class="subtitle">{{ t('admin.users.subtitle') }}</p>
      </div>
      <n-button type="primary" @click="openCreate">
        <template #icon><n-icon :component="AddOutline" /></template>
        {{ t('admin.users.create') }}
      </n-button>
    </header>

    <n-card :bordered="false" class="table-card">
      <n-data-table :columns="columns" :data="users" :row-key="(r) => r.id" :pagination="{ pageSize: 8 }" />
    </n-card>

    <n-modal
      v-model:show="showCreate"
      :title="t('admin.users.create')"
      preset="card"
      style="width: 440px; max-width: 92vw"
    >
      <n-form label-placement="left" :label-width="84">
        <n-form-item :label="t('admin.users.formName')">
          <n-input v-model:value="form.name" :placeholder="t('admin.users.namePh')" />
        </n-form-item>
        <n-form-item :label="t('admin.users.formEmail')">
          <n-input v-model:value="form.email" :placeholder="t('admin.users.emailPh')" />
        </n-form-item>
        <n-form-item :label="t('admin.users.formPassword')">
          <n-input v-model:value="form.password" type="password" :placeholder="t('admin.users.pwdPh')" />
        </n-form-item>
        <n-form-item :label="t('admin.users.formRole')">
          <n-select v-model:value="form.role" :options="roleOptions" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="modal-foot">
          <n-button @click="showCreate = false">{{ t('admin.announcements.statusDraft') }}</n-button>
          <n-button type="primary" @click="submitCreate">{{ t('admin.users.create') }}</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>
