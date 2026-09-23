<script setup lang="ts">
import { computed, h, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NDataTable,
  NEmpty,
  NFormItem,
  NIcon,
  NModal,
  NSelect,
  NSpace,
  NTag,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import { CloudDownloadOutline } from '@vicons/ionicons5'
import { useModelStore } from '@/stores/model'
import type { CreateModelPayload, UserModelPayload } from '@/api/model'
import type { ModelCapabilities, ModelType, RemoteModel } from '@/types/model'
import { MODEL_TYPES } from '@/types/model'

/**
 * 上游模型目录弹窗：按已保存渠道的 base_url + api_key 调上游 /models，
 * 勾选后逐条登记成平台模型（管理端）或我的自定义模型（用户端）。
 *
 * 两个设计取舍值得说明：
 *  1. 只按「渠道 ID」拉取，密钥由服务端解密后出站，浏览器全程不接触明文——
 *     一旦密钥进了前端就收不回来，因此这里不提供「临时填密钥拉一次」的入口。
 *  2. 上下文窗口、能力、定价这些上游目录不返回的字段一律不猜，只让用户先定一个
 *     模型类型，其余留默认值，登记后仍可在表单里补。
 *     平台模型的 code 由「渠道 code + 上游标识」派生：code 全局唯一且创建后不可改，
 *     只取上游标识容易和别的渠道撞上，带上渠道前缀才稳定——派生结果就摆在列表里，
 *     用户登记前能看见自己将要得到什么。
 */
const props = withDefaults(
  defineProps<{
    show: boolean
    scope?: 'platform' | 'user'
    /** 渠道下拉项：平台渠道来自管理端，自定义渠道来自当前用户。 */
    providers?: { label: string; value: number; code?: string }[]
    /** 已登记的 api_model，用于把目录里重复的条目标成不可选。 */
    existingApiModels?: string[]
  }>(),
  { scope: 'platform', providers: () => [], existingApiModels: () => [] },
)

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'registered', count: number): void
}>()

const { t } = useI18n()
const message = useMessage()
const store = useModelStore()

const providerId = ref(0)
const checked = ref<string[]>([])
const modelType = ref<ModelType>('chat')
const submitting = ref(false)

const items = computed(() => store.remoteModels)
const typeOptions = MODEL_TYPES.map((m) => ({ label: t(`model.type.${m.value}`), value: m.value }))
const hasProviders = computed(() => props.providers.length > 0)

function currentProvider() {
  return props.providers.find((p) => p.value === providerId.value)
}

function isRegistered(m: RemoteModel) {
  return props.existingApiModels.includes(m.id)
}

function slug(s: string): string {
  return s
    .toLowerCase()
    .replace(/[^a-z0-9._-]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

/** 模型类型决定默认能力：对话类开流式与视觉，其余类型先全关，登记后再按需开。 */
function defaultCapabilities(type: ModelType): ModelCapabilities {
  return { stream: type === 'chat', toolCall: false, jsonMode: false, vision: type === 'vision' }
}

const columns: DataTableColumns<RemoteModel> = [
  { type: 'selection', disabled: (row) => isRegistered(row) },
  {
    title: t('model.remote.columnId'),
    key: 'id',
    render: (row) =>
      h('div', { class: 'cell-stack' }, [
        h('div', { class: 'remote-id-line' }, [
          h('span', { class: 'remote-id-main' }, row.id),
          isRegistered(row)
            ? h(
                NTag,
                { size: 'small', bordered: false, type: 'warning' },
                { default: () => t('model.remote.registeredAlready') },
              )
            : null,
        ]),
        // 只有平台模型有编码：把将要生成的 code 提前亮出来，它是创建后不可改的。
        props.scope === 'platform'
          ? h('span', { class: 'cell-sub remote-id-sub' }, toModelCode(row.id))
          : null,
      ]),
  },
  {
    title: t('model.remote.columnOwner'),
    key: 'ownedBy',
    width: 130,
    render: (row) => row.ownedBy || '—',
  },
  {
    title: t('model.remote.columnObject'),
    key: 'object',
    width: 104,
    render: (row) =>
      row.object
        ? h(NTag, { size: 'small', bordered: false }, { default: () => row.object })
        : '—',
  },
]

/** 平台模型编码：唯一且创建后不可改，因此从「渠道 code + 上游标识」派生稳定值。 */
function toModelCode(id: string): string {
  const code = [slug(currentProvider()?.code ?? ''), slug(id)].filter(Boolean).join('-').slice(0, 90)
  return code || 'model'
}

async function load() {
  if (!providerId.value) {
    message.warning(t('model.validate.providerRequired'))
    return
  }
  try {
    if (props.scope === 'platform') {
      await store.fetchAdminCatalog(providerId.value)
    } else {
      await store.fetchUserCatalog(providerId.value)
    }
    // 已登记的条目不默认勾选，避免重复登记撞唯一键。
    checked.value = items.value.filter((m) => !isRegistered(m)).map((m) => m.id)
  } catch {
    // 失败提示由请求层给出，这里只把勾选状态清干净。
    checked.value = []
  }
}

async function register() {
  const picked = items.value.filter((m) => checked.value.includes(m.id) && !isRegistered(m))
  if (!picked.length) {
    message.warning(t('model.remote.selectFirst'))
    return
  }
  submitting.value = true
  const payloads: (CreateModelPayload | UserModelPayload)[] = picked.map((m) => {
    const base: UserModelPayload = {
      providerId: providerId.value,
      name: m.id,
      modelType: modelType.value,
      apiModel: m.id,
      capabilities: defaultCapabilities(modelType.value),
    }
    return props.scope === 'platform' ? { ...base, code: toModelCode(m.id) } : base
  })
  const { ok, failed } = await store.registerCatalogModels(props.scope, payloads)
  submitting.value = false

  if (!failed.length) {
    message.success(t('model.remote.registered', { n: ok }))
    emit('registered', ok)
    emit('update:show', false)
    return
  }
  // 部分成功时保留弹窗与勾选，失败条目仍在列表里，改完类型可以直接重试。
  checked.value = failed
  message.error(t('model.remote.partial', { ok, failed: failed.length }))
}

// 打开弹窗即定位到首个渠道并拉一次目录：绝大多数情况用户就是来拉模型的。
watch(
  () => props.show,
  (open) => {
    if (!open) return
    checked.value = []
    store.remoteModels = []
    providerId.value = props.providers[0]?.value ?? 0
    if (providerId.value) load()
  },
)
</script>

<template>
  <n-modal
    :show="show"
    :title="t('model.remote.title')"
    preset="card"
    style="width: 680px; max-width: 94vw"
    :bordered="false"
    @update:show="emit('update:show', $event)"
  >
    <n-empty v-if="!hasProviders" :description="t('model.remote.noProviders')" />
    <template v-else>
      <div class="remote-bar">
        <n-form-item :label="t('model.remote.provider')" :show-feedback="false" style="flex: 1">
          <n-select
            v-model:value="providerId"
            :options="providers"
            :placeholder="t('model.providerPh')"
            @update:value="checked = []"
          />
        </n-form-item>
        <n-form-item :label="t('model.remote.registerAs')" :show-feedback="false" style="width: 150px">
          <n-select v-model:value="modelType" :options="typeOptions" />
        </n-form-item>
        <n-button
          class="remote-fetch"
          :loading="store.loadingRemote"
          :disabled="!providerId"
          @click="load"
        >
          <template #icon><n-icon :component="CloudDownloadOutline" /></template>
          {{ t('model.remote.fetch') }}
        </n-button>
      </div>

      <p class="remote-hint">{{ t('model.remote.hint') }}</p>

      <n-empty v-if="!store.loadingRemote && !items.length" :description="t('model.remote.empty')">
        <template #extra>
          <n-button size="small" :disabled="!providerId" @click="load">
            {{ t('model.remote.fetch') }}
          </n-button>
        </template>
      </n-empty>
      <n-data-table
        v-else
        :columns="columns"
        :data="items"
        :row-key="(r: RemoteModel) => r.id"
        :checked-row-keys="checked"
        :row-props="(row: RemoteModel) => ({ class: isRegistered(row) ? 'is-registered' : '' })"
        :loading="store.loadingRemote"
        :pagination="false"
        size="small"
        max-height="320"
        @update:checked-row-keys="(keys: Array<string | number>) => (checked = keys as string[])"
      />
    </template>

    <template #footer>
      <div class="remote-foot">
        <span class="remote-count">{{ t('model.remote.selected', { n: checked.length }) }}</span>
        <n-space>
          <n-button quaternary :disabled="submitting" @click="emit('update:show', false)">
            {{ t('common.cancel') }}
          </n-button>
          <n-button type="primary" :loading="submitting" :disabled="!checked.length" @click="register">
            <template #icon><n-icon :component="CloudDownloadOutline" /></template>
            {{ t('model.remote.register') }}
          </n-button>
        </n-space>
      </div>
    </template>
  </n-modal>
</template>
