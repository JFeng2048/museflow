<script setup lang="ts">
import { ref, h, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NCard, NDataTable, NTag, NButton, NSwitch, NIcon, NModal, NForm, NFormItem,
  NInput, NSelect, useMessage, type DataTableColumns,
} from 'naive-ui'
import { CreateOutline } from '@vicons/ionicons5'
import { adminModels } from '@/mock/admin'
import type { AdminModel } from '@/types/admin'

const { t } = useI18n()
const message = useMessage()
const models = ref<AdminModel[]>([...adminModels])

const categoryOptions = [
  { label: '对话', value: '对话' },
  { label: '续写', value: '续写' },
  { label: '推理', value: '推理' },
  { label: '嵌入', value: '嵌入' },
  { label: '图像', value: '图像' },
]

function maskKey(key: string): string {
  return key.length > 6 ? key.slice(0, 3) + '****' + key.slice(-4) : '****'
}

// 编辑弹窗
const showEdit = ref(false)
const editing = reactive<AdminModel & { _ctx: string }>({
  id: '', name: '', provider: '', baseUrl: '', apiKey: '', category: '对话', contextWindow: 0, enabled: true, _ctx: '',
})

function openEdit(row: AdminModel) {
  Object.assign(editing, row, { _ctx: String(row.contextWindow || 0) })
  showEdit.value = true
}

function openAdd() {
  Object.assign(editing, {
    id: 'm-' + Math.random().toString(36).slice(2, 8),
    name: '', provider: '', baseUrl: '', apiKey: '', category: '对话', contextWindow: 0, enabled: true, _ctx: '0',
  })
  showEdit.value = true
}

function submitEdit() {
  if (!editing.name || !editing.provider || !editing.baseUrl) {
    message.warning(t('admin.models.namePh'))
    return
  }
  editing.contextWindow = Number(editing._ctx) || 0
  const idx = models.value.findIndex((m) => m.id === editing.id)
  if (idx >= 0) models.value[idx] = { ...editing }
  else models.value.unshift({ ...editing })
  showEdit.value = false
  message.success(t('admin.models.saved'))
}

const columns: DataTableColumns<AdminModel> = [
  { title: t('admin.models.name'), key: 'name', render: (row) => row.name },
  { title: t('admin.models.provider'), key: 'provider' },
  { title: t('admin.models.baseUrl'), key: 'baseUrl', render: (row) => row.baseUrl },
  {
    title: t('admin.models.apiKey'), key: 'apiKey',
    render: (row) => h('code', { class: 'mono' }, maskKey(row.apiKey)),
  },
  {
    title: t('admin.models.category'), key: 'category',
    render: (row) => h(NTag, { size: 'small', bordered: false }, { default: () => row.category }),
  },
  {
    title: t('admin.models.context'), key: 'contextWindow',
    render: (row) => (row.contextWindow ? row.contextWindow.toLocaleString() + ' tok' : '—'),
  },
  {
    title: t('admin.models.status'), key: 'enabled',
    render: (row) =>
      h(NSwitch, {
        size: 'small',
        value: row.enabled,
        onUpdateValue: (v: boolean) => { row.enabled = v },
      }),
  },
  {
    title: t('admin.models.edit'), key: 'op',
    render: (row) =>
      h(NButton, { size: 'small', tertiary: true, onClick: () => openEdit(row) }, { default: () => t('admin.models.edit') }),
  },
]
</script>

<template>
  <div class="admin-page">
    <header class="admin-head with-action">
      <div>
        <h1 class="title">{{ t('admin.models.title') }}</h1>
        <p class="subtitle">{{ t('admin.models.subtitle') }}</p>
      </div>
      <n-button type="primary" @click="openAdd">
        <template #icon><n-icon :component="CreateOutline" /></template>
        {{ t('admin.models.add') }}
      </n-button>
    </header>

    <n-card :bordered="false" class="table-card">
      <n-data-table :columns="columns" :data="models" :row-key="(r) => r.id" :pagination="false" />
    </n-card>

    <n-modal
      v-model:show="showEdit"
      :title="editing.id && models.some((m) => m.id === editing.id) ? t('admin.models.edit') : t('admin.models.add')"
      preset="card"
      style="width: 520px; max-width: 92vw"
    >
      <n-form label-placement="left" :label-width="96">
        <n-form-item :label="t('admin.models.formName')">
          <n-input v-model:value="editing.name" :placeholder="t('admin.models.namePh')" />
        </n-form-item>
        <n-form-item :label="t('admin.models.formProvider')">
          <n-input v-model:value="editing.provider" :placeholder="t('admin.models.providerPh')" />
        </n-form-item>
        <n-form-item :label="t('admin.models.formBaseUrl')">
          <n-input v-model:value="editing.baseUrl" :placeholder="t('admin.models.baseUrlPh')" />
        </n-form-item>
        <n-form-item :label="t('admin.models.formApiKey')">
          <n-input v-model:value="editing.apiKey" type="password" show-password-on="click" :placeholder="t('admin.models.apiKeyPh')" />
        </n-form-item>
        <n-form-item :label="t('admin.models.formCategory')">
          <n-select v-model:value="editing.category" :options="categoryOptions" :placeholder="t('admin.models.categoryPh')" />
        </n-form-item>
        <n-form-item :label="t('admin.models.formContext')">
          <n-input v-model:value="editing._ctx" type="text" inputmode="numeric" :placeholder="t('admin.models.contextPh')" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="modal-foot">
          <n-button @click="showEdit = false">{{ t('admin.announcements.statusDraft') }}</n-button>
          <n-button type="primary" @click="submitEdit">{{ t('admin.models.save') }}</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>
