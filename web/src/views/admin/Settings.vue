<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NDataTable,
  NIcon,
  NPopconfirm,
  NSelect,
  NTag,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import {
  CreateOutline,
  LockClosedOutline,
  LockOpenOutline,
  PencilOutline,
  RefreshOutline,
  TrashOutline,
} from '@vicons/ionicons5'
import { useModelStore } from '@/stores/model'
import type { SystemSetting } from '@/types/model'
import SettingFormModal from '@/components/model/SettingFormModal.vue'

/**
 * 管理端系统配置页。
 *
 * 这张表的定位是「平台侧可热改的配置收口处」：模型渠道之外的所有可调参数
 * （分组 + 键名 + 值）都落在这里，带分组的 key-value 让新增一项配置不需要改表。
 * 机密项（密钥 / 令牌）走加密列，界面上只能看到「已配置」——
 * 与渠道的 api_key 同一套约定：只写不读，回显一次。
 */
const { t } = useI18n()
const message = useMessage()
const store = useModelStore()

onMounted(() => store.fetchSettings())

/** 分组候选从当前列表里归纳，避免为了一个下拉框再存一张字典表。 */
const groupOptions = computed(() => {
  const groups = Array.from(new Set(store.settings.map((s) => s.configGroup))).sort()
  return [
    { label: t('common.all'), value: '' },
    ...groups.map((g) => ({ label: g, value: g })),
  ]
})

const showForm = ref(false)
const editing = ref<SystemSetting | null>(null)

function openAdd() {
  editing.value = null
  showForm.value = true
}

function openEdit(row: SystemSetting) {
  editing.value = row
  showForm.value = true
}

async function submit(payload: Record<string, unknown>, group: string, key: string) {
  const editingRow = editing.value
  try {
    if (editingRow) {
      await store.saveSetting(editingRow.configGroup, editingRow.key, payload as never)
      message.success(t('model.savedSetting'))
    } else {
      await store.saveSetting(group, key, payload as never)
      message.success(t('model.createdSetting'))
    }
    showForm.value = false
  } catch {
    // 失败提示由请求层统一给出。
  }
}

const columns: DataTableColumns<SystemSetting> = [
  {
    title: t('model.configGroup'),
    key: 'configGroup',
    width: 130,
    render: (row) =>
      h('div', { class: 'cell-stack' }, [
        h('span', { class: 'cell-main' }, row.configGroup),
        h('span', { class: 'cell-sub mono' }, row.key),
      ]),
  },
  {
    title: t('model.value'),
    key: 'value',
    ellipsis: { tooltip: true },
    render: (row) => {
      if (row.isSecret) {
        return h(
          'span',
          { class: 'secret-badge' },
          { default: () => [h(NIcon, { component: LockClosedOutline, size: 13 }), t('model.secretConfigured')] },
        )
      }
      return h('span', { class: 'mono cell-value' }, row.value || '—')
    },
  },
  {
    title: t('model.valueType'),
    key: 'valueType',
    width: 96,
    render: (row) => h(NTag, { size: 'small', bordered: false }, { default: () => row.valueType }),
  },
  {
    title: t('model.isPublic'),
    key: 'isPublic',
    width: 96,
    render: (row) =>
      row.isPublic
        ? h(
            NTag,
            { size: 'small', bordered: false, type: 'success' },
            { default: () => t('model.publicOn'), icon: () => h(NIcon, { component: LockOpenOutline, size: 13 }) },
          )
        : t('model.publicOff'),
  },
  {
    title: t('model.updatedAt'),
    key: 'updatedAt',
    width: 168,
    render: (row) => (row.updatedAt ? new Date(row.updatedAt).toLocaleString() : '—'),
  },
  {
    title: t('model.description'),
    key: 'description',
    ellipsis: { tooltip: true },
    render: (row) => row.description || '—',
  },
  {
    title: t('admin.models.edit'),
    key: 'op',
    width: 140,
    render: (row) =>
      h('div', { class: 'cell-actions' }, [
        h(
          NButton,
          { size: 'small', tertiary: true, onClick: () => openEdit(row) },
          { default: () => t('common.edit'), icon: () => h(NIcon, { component: PencilOutline }) },
        ),
        h(
          NPopconfirm,
          { onPositiveClick: () => store.removeSetting(row.configGroup, row.key) },
          {
            trigger: () =>
              h(
                NButton,
                { size: 'small', tertiary: true, type: 'error' },
                { default: () => t('common.delete'), icon: () => h(NIcon, { component: TrashOutline }) },
              ),
            default: () => t('model.confirmDeleteSetting'),
          },
        ),
      ]),
  },
]
</script>

<template>
  <div class="admin-page">
    <header class="admin-head with-action">
      <div>
        <h1 class="title">{{ t('admin.settings.title') }}</h1>
        <p class="subtitle">{{ t('admin.settings.subtitle') }}</p>
      </div>
      <n-button :loading="store.loadingSettings" @click="store.fetchSettings()">
        <template #icon><n-icon :component="RefreshOutline" /></template>
        {{ t('admin.services.refresh') }}
      </n-button>
    </header>

    <div class="toolbar">
      <div class="toolbar-filters">
        <n-select
          :value="store.settingFilter.configGroup"
          :options="groupOptions"
          style="width: 220px"
          @update:value="(v: string) => { store.settingFilter.configGroup = v; store.fetchSettings() }"
        />
        <n-switch v-model:value="store.settingFilter.onlyPublic" size="small" @update:value="store.fetchSettings()">
          <template #checked>{{ t('model.onlyPublic') }}</template>
          <template #unchecked>{{ t('common.all') }}</template>
        </n-switch>
      </div>
      <n-button type="primary" @click="openAdd">
        <template #icon><n-icon :component="CreateOutline" /></template>
        {{ t('model.addSetting') }}
      </n-button>
    </div>

    <n-card :bordered="false" class="table-card">
      <n-data-table
        :columns="columns"
        :data="store.settings"
        :row-key="(r: SystemSetting) => r.id"
        :loading="store.loadingSettings"
        :pagination="false"
        size="small"
      />
    </n-card>

    <setting-form-modal
      :show="showForm"
      :setting="editing"
      @update:show="showForm = $event"
      @submit="submit"
    />
  </div>
</template>
