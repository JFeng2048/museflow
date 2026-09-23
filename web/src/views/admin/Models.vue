<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NDataTable,
  NIcon,
  NPagination,
  NPopconfirm,
  NSwitch,
  NTabPane,
  NTabs,
  NTag,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import {
  CloudDownloadOutline,
  CreateOutline,
  PencilOutline,
  RefreshOutline,
  TrashOutline,
} from '@vicons/ionicons5'
import { useModelStore } from '@/stores/model'
import type { AIModel, ModelProvider } from '@/types/model'
import { MODEL_TYPES, modelTypeMeta } from '@/types/model'
import ProviderFormModal from '@/components/model/ProviderFormModal.vue'
import ModelFormModal from '@/components/model/ModelFormModal.vue'
import RemoteModelModal from '@/components/model/RemoteModelModal.vue'
import CapabilityIcons from '@/components/model/CapabilityIcons.vue'
import KeySlot from '@/components/model/KeySlot.vue'

/**
 * 管理端模型配置页。
 *
 * 两个页签对应两层配置：先维护「渠道」（厂商 + Base URL + API Key），
 * 再在渠道下维护「模型」（调用标识、上下文、能力、积分价）。
 * 之所以拆开而不是合成一张表：密钥属于渠道，一个渠道下的多个模型共用同一把钥匙，
 * 轮换密钥时不该挨个改模型。用户端可见的是这一页配好的、已上架的模型。
 */
const { t } = useI18n()
const message = useMessage()
const store = useModelStore()

const tab = ref<'providers' | 'models'>('providers')

onMounted(() => {
  store.fetchProviders()
  store.fetchModels()
})

// ---------- 渠道 ----------

const showProvider = ref(false)
const editingProvider = ref<ModelProvider | null>(null)

function openAddProvider() {
  editingProvider.value = null
  showProvider.value = true
}

function openEditProvider(row: ModelProvider) {
  editingProvider.value = row
  showProvider.value = true
}

async function submitProvider(payload: Record<string, unknown>) {
  try {
    if (editingProvider.value) {
      await store.editProvider(editingProvider.value.id, payload as never)
      message.success(t('model.savedProvider'))
    } else {
      await store.addProvider(payload as never)
      message.success(t('model.createdProvider'))
    }
    showProvider.value = false
  } catch {
    // 失败提示由请求层统一给出，这里只负责关弹窗的时机。
  }
}

function toggleProvider(row: ModelProvider, value: boolean) {
  store.toggleProvider(row.id, value)
}

function removeProvider(row: ModelProvider) {
  store.removeProvider(row.id)
}

const providerOptions = computed(() => store.providers.map((p) => ({ label: p.name, value: p.id })))

/** 目录弹窗的渠道下拉：带上渠道编码，用来派生平台模型编码。 */
const catalogProviders = computed(() =>
  store.providers.map((p) => ({ label: p.name, value: p.id, code: p.code })),
)

/** 模型类型筛选候选直接从类型元信息派生，新增类型不用改这里。 */
const modelTypeFilterOptions = computed(() => [
  { label: t('common.all'), value: '' },
  ...MODEL_TYPES.map((m) => ({ label: t(`model.type.${m.value}`), value: m.value })),
])

const providerColumns: DataTableColumns<ModelProvider> = [
  {
    title: t('model.name'),
    key: 'name',
    render: (row) =>
      h('div', { class: 'cell-stack' }, [
        h('span', { class: 'cell-main' }, row.name),
        h('span', { class: 'cell-sub mono' }, row.code),
      ]),
  },
  {
    title: t('model.protocol'),
    key: 'protocol',
    width: 120,
    render: (row) => h(NTag, { size: 'small', bordered: false }, { default: () => row.protocol }),
  },
  {
    title: t('model.baseUrl'),
    key: 'baseUrl',
    ellipsis: { tooltip: true },
    render: (row) => h('span', { class: 'mono cell-url' }, row.baseUrl),
  },
  {
    title: t('model.apiKey'),
    key: 'apiKey',
    width: 180,
    render: (row) => h(KeySlot, { hint: row.apiKeyHint, updatedAt: row.apiKeyUpdatedAt, compact: true }),
  },
  {
    title: t('model.organization'),
    key: 'organization',
    width: 120,
    render: (row) => row.organization || '—',
  },
  { title: t('model.sortOrder'), key: 'sortOrder', width: 88 },
  {
    title: t('admin.models.status'),
    key: 'isActive',
    width: 88,
    render: (row) =>
      h(NSwitch, {
        size: 'small',
        value: row.isActive,
        onUpdateValue: (v: boolean) => toggleProvider(row, v),
      }),
  },
  {
    title: t('admin.models.edit'),
    key: 'op',
    width: 140,
    render: (row) =>
      h('div', { class: 'cell-actions' }, [
        h(
          NButton,
          { size: 'small', tertiary: true, onClick: () => openEditProvider(row) },
          { default: () => t('common.edit'), icon: () => h(NIcon, { component: PencilOutline }) },
        ),
        h(
          NPopconfirm,
          { onPositiveClick: () => removeProvider(row) },
          {
            trigger: () =>
              h(
                NButton,
                { size: 'small', tertiary: true, type: 'error' },
                { default: () => t('common.delete'), icon: () => h(NIcon, { component: TrashOutline }) },
              ),
            default: () => t('model.confirmDeleteProvider'),
          },
        ),
      ]),
  },
]

// ---------- 模型 ----------

const showModel = ref(false)
const editingModel = ref<AIModel | null>(null)
const showCatalog = ref(false)

function openAddModel() {
  if (!store.providers.length) {
    message.warning(t('model.needProviderFirst'))
    return
  }
  editingModel.value = null
  showModel.value = true
}

function openEditModel(row: AIModel) {
  editingModel.value = row
  showModel.value = true
}

async function submitModel(payload: Record<string, unknown>) {
  try {
    if (editingModel.value) {
      await store.editModel(editingModel.value.id, payload as never)
      message.success(t('model.savedModel'))
    } else {
      await store.addModel(payload as never)
      message.success(t('model.createdModel'))
    }
    showModel.value = false
  } catch {
    // 同上：错误提示走请求层。
  }
}

function toggleModel(row: AIModel, value: boolean) {
  store.toggleModel(row.id, value)
}

const modelColumns: DataTableColumns<AIModel> = [
  {
    title: t('model.name'),
    key: 'name',
    render: (row) =>
      h('div', { class: 'cell-stack' }, [
        h('span', { class: 'cell-main' }, row.name),
        h('span', { class: 'cell-sub mono' }, row.code),
      ]),
  },
  {
    title: t('model.provider'),
    key: 'providerId',
    width: 140,
    render: (row) => store.providerName(row.providerId),
  },
  {
    title: t('model.modelType'),
    key: 'modelType',
    width: 108,
    render: (row) => {
      const meta = modelTypeMeta(row.modelType)
      return h(
        NTag,
        { size: 'small', bordered: false },
        {
          default: () => t(`model.type.${row.modelType}`),
          icon: () => h(NIcon, { component: meta.icon, size: 13 }),
        },
      )
    },
  },
  {
    title: t('model.apiModel'),
    key: 'apiModel',
    width: 190,
    ellipsis: { tooltip: true },
    render: (row) => h('span', { class: 'mono' }, row.apiModel),
  },
  {
    title: t('model.contextWindow'),
    key: 'contextWindow',
    width: 130,
    render: (row) => (row.contextWindow ? `${row.contextWindow.toLocaleString()} / ${row.maxOutputTokens.toLocaleString()}` : '—'),
  },
  {
    title: t('model.capabilitiesTitle'),
    key: 'capabilities',
    width: 116,
    render: (row) => h(CapabilityIcons, { capabilities: row.capabilities }),
  },
  {
    title: t('model.creditCost'),
    key: 'creditCost',
    width: 96,
    render: (row) => (row.creditCost ? String(row.creditCost) : '0'),
  },
  { title: t('model.sortOrder'), key: 'sortOrder', width: 88 },
  {
    title: t('admin.models.status'),
    key: 'isActive',
    width: 88,
    render: (row) =>
      h(NSwitch, {
        size: 'small',
        value: row.isActive,
        onUpdateValue: (v: boolean) => toggleModel(row, v),
      }),
  },
  {
    title: t('admin.models.edit'),
    key: 'op',
    width: 140,
    render: (row) =>
      h('div', { class: 'cell-actions' }, [
        h(
          NButton,
          { size: 'small', tertiary: true, onClick: () => openEditModel(row) },
          { default: () => t('common.edit'), icon: () => h(NIcon, { component: PencilOutline }) },
        ),
        h(
          NPopconfirm,
          { onPositiveClick: () => store.removeModel(row.id) },
          {
            trigger: () =>
              h(
                NButton,
                { size: 'small', tertiary: true, type: 'error' },
                { default: () => t('common.delete'), icon: () => h(NIcon, { component: TrashOutline }) },
              ),
            default: () => t('model.confirmDeleteModel'),
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
        <h1 class="title">{{ t('admin.models.title') }}</h1>
        <p class="subtitle">{{ t('admin.models.subtitle') }}</p>
      </div>
      <n-button :loading="store.loadingProviders || store.loadingModels" @click="store.fetchProviders(); store.fetchModels()">
        <template #icon><n-icon :component="RefreshOutline" /></template>
        {{ t('admin.services.refresh') }}
      </n-button>
    </header>

    <n-tabs v-model:value="tab" type="line" animated>
      <n-tab-pane name="providers" :tab="t('model.providers')">
        <div class="toolbar">
          <div class="toolbar-filters">
            <n-input
              :value="store.providerFilter.keyword"
              :placeholder="t('model.filterProviderPh')"
              clearable
              style="width: 240px"
              @update:value="(v: string) => { store.providerFilter.keyword = v; store.providerFilter.page = 1; store.fetchProviders() }"
            />
            <n-switch v-model:value="store.providerFilter.onlyActive" size="small" @update:value="store.providerFilter.page = 1; store.fetchProviders()">
              <template #checked>{{ t('model.onlyActive') }}</template>
              <template #unchecked>{{ t('common.all') }}</template>
            </n-switch>
          </div>
          <n-button type="primary" @click="openAddProvider">
            <template #icon><n-icon :component="CreateOutline" /></template>
            {{ t('model.addProvider') }}
          </n-button>
        </div>

        <n-card :bordered="false" class="table-card">
          <n-data-table
            :columns="providerColumns"
            :data="store.providers"
            :row-key="(r: ModelProvider) => r.id"
            :loading="store.loadingProviders"
            :pagination="false"
            size="small"
          />
          <div class="table-foot">
            <n-pagination
              :page="store.providerFilter.page"
              :page-size="store.providerFilter.pageSize"
              :item-count="store.providersTotal"
              :page-slot="7"
              size="small"
              @update:page="(p: number) => { store.providerFilter.page = p; store.fetchProviders() }"
            />
          </div>
        </n-card>
      </n-tab-pane>

      <n-tab-pane name="models" :tab="t('model.models')">
        <div class="toolbar">
          <div class="toolbar-filters">
            <n-input
              :value="store.modelFilter.keyword"
              :placeholder="t('model.filterModelPh')"
              clearable
              style="width: 240px"
              @update:value="(v: string) => { store.modelFilter.keyword = v; store.modelFilter.page = 1; store.fetchModels() }"
            />
            <n-select
              :value="store.modelFilter.providerId"
              :options="[{ label: t('common.all'), value: 0 }, ...providerOptions]"
              style="width: 180px"
              @update:value="(v: number) => { store.modelFilter.providerId = v; store.modelFilter.page = 1; store.fetchModels() }"
            />
            <n-select
              :value="store.modelFilter.modelType"
              :options="modelTypeFilterOptions"
              style="width: 150px"
              @update:value="(v: string) => { store.modelFilter.modelType = v as never; store.modelFilter.page = 1; store.fetchModels() }"
            />
            <n-switch v-model:value="store.modelFilter.onlyActive" size="small" @update:value="store.modelFilter.page = 1; store.fetchModels()">
              <template #checked>{{ t('model.onlyActive') }}</template>
              <template #unchecked>{{ t('common.all') }}</template>
            </n-switch>
          </div>
          <n-button :disabled="!store.providers.length" @click="showCatalog = true">
            <template #icon><n-icon :component="CloudDownloadOutline" /></template>
            {{ t('model.remote.importFrom') }}
          </n-button>
          <n-button type="primary" @click="openAddModel">
            <template #icon><n-icon :component="CreateOutline" /></template>
            {{ t('model.addModel') }}
          </n-button>
        </div>

        <n-card :bordered="false" class="table-card">
          <n-data-table
            :columns="modelColumns"
            :data="store.models"
            :row-key="(r: AIModel) => r.id"
            :loading="store.loadingModels"
            :pagination="false"
            size="small"
          />
          <div class="table-foot">
            <n-pagination
              :page="store.modelFilter.page"
              :page-size="store.modelFilter.pageSize"
              :item-count="store.modelsTotal"
              :page-slot="7"
              size="small"
              @update:page="(p: number) => { store.modelFilter.page = p; store.fetchModels() }"
            />
          </div>
        </n-card>
      </n-tab-pane>
    </n-tabs>

    <provider-form-modal
      :show="showProvider"
      :provider="editingProvider"
      scope="platform"
      @update:show="showProvider = $event"
      @submit="submitProvider"
    />
    <model-form-modal
      :show="showModel"
      :model="editingModel"
      scope="platform"
      :providers="providerOptions"
      @update:show="showModel = $event"
      @submit="submitModel"
    />
    <!-- 目录弹窗：已登记清单只覆盖当前页，翻页后的重复项由后端唯一约束兜底 -->
    <remote-model-modal
      :show="showCatalog"
      scope="platform"
      :providers="catalogProviders"
      :existing-api-models="store.models.map((m) => m.apiModel)"
      @update:show="showCatalog = $event"
    />
  </div>
</template>
