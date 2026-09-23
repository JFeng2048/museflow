<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import {
  NButton,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NInputNumber,
  NModal,
  NSelect,
  NSwitch,
  useMessage,
} from 'naive-ui'
import { CreateOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import type { ModelProtocol, ModelProvider, UserProvider } from '@/types/model'
import { PROTOCOLS } from '@/types/model'

/**
 * 渠道表单弹窗：平台渠道（管理端）与我的自定义渠道（用户端）共用。
 *
 * 两者字段几乎一致，差别只有三处，都由 scope 决定：
 *  - platform 多一个渠道编码（创建时可填、创建后不可改）与排序权重；
 *  - user     多一个启用开关，且编码不存在；
 *  - api_key  一律是「一次性凭据」：新建时填写，编辑时留空表示不轮换。
 *
 * 弹窗只负责表单与校验，保存动作交给调用方（store 归属在父级）；
 * submit 抛出的是 camelCase 领域字段，与 @/api/model 的入参形状一致，
 * 父级可以直接转发给 store。
 */
const props = withDefaults(
  defineProps<{
    show: boolean
    /** 传入表示编辑，缺省表示新增。 */
    provider?: ModelProvider | UserProvider | null
    scope?: 'platform' | 'user'
  }>(),
  { provider: null, scope: 'platform' },
)

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'submit', payload: Record<string, unknown>): void
}>()

const { t } = useI18n()
const message = useMessage()

const isEdit = () => !!props.provider
const protocolOptions = PROTOCOLS.map((p) => ({ label: p.label, value: p.value }))

const form = reactive({
  code: '',
  name: '',
  protocol: 'openai' as ModelProtocol,
  baseUrl: '',
  /** 编辑时留空即「不轮换」，因此不能回填任何值。 */
  apiKey: '',
  organization: '',
  sortOrder: 0,
  isActive: true,
})

// 打开弹窗时按「编辑 / 新增」重置表单；密钥永远从空开始。
watch(
  () => props.show,
  (open) => {
    if (!open) return
    const p = props.provider
    Object.assign(form, {
      code: p && 'code' in p ? p.code : '',
      name: p?.name ?? '',
      protocol: p?.protocol ?? 'openai',
      baseUrl: p?.baseUrl ?? '',
      apiKey: '',
      organization: p?.organization ?? '',
      sortOrder: 'sortOrder' in (p ?? {}) ? (p as ModelProvider).sortOrder : 0,
      isActive: p ? p.isActive : true,
    })
  },
)

const saving = ref(false)

function submit() {
  if (!form.name.trim()) {
    message.warning(t('model.validate.nameRequired'))
    return
  }
  if (!form.baseUrl.trim()) {
    message.warning(t('model.validate.baseUrlRequired'))
    return
  }
  if (!isEdit() && props.scope === 'platform' && !form.code.trim()) {
    message.warning(t('model.validate.codeRequired'))
    return
  }
  saving.value = true
  const payload: Record<string, unknown> = {
    name: form.name.trim(),
    protocol: form.protocol,
    baseUrl: form.baseUrl.trim(),
    // 空串交给后端判断：新增时等于不配密钥，编辑时等于不轮换。
    apiKey: form.apiKey.trim(),
    organization: form.organization.trim(),
  }
  if (props.scope === 'platform') {
    payload.sortOrder = form.sortOrder
    if (!isEdit()) payload.code = form.code.trim()
  } else {
    payload.isActive = form.isActive
  }
  emit('submit', payload)
  saving.value = false
}
</script>

<template>
  <n-modal
    :show="show"
    :title="
      isEdit()
        ? scope === 'platform'
          ? t('model.editProvider')
          : t('model.editMyProvider')
        : scope === 'platform'
          ? t('model.addProvider')
          : t('model.addMyProvider')
    "
    preset="card"
    style="width: 560px; max-width: 92vw"
    :bordered="false"
    @update:show="emit('update:show', $event)"
  >
    <n-form label-placement="top">
      <n-form-item v-if="scope === 'platform'" :label="t('model.code')">
        <n-input
          :value="form.code"
          :disabled="isEdit()"
          :placeholder="t('model.codePh')"
          @update:value="form.code = $event"
        />
        <template v-if="isEdit()" #feedback>{{ t('model.codeImmutable') }}</template>
      </n-form-item>
      <n-form-item :label="t('model.name')">
        <n-input
          :value="form.name"
          :placeholder="t('model.namePlaceholder')"
          @update:value="form.name = $event"
        />
      </n-form-item>
      <n-form-item :label="t('model.protocol')">
        <n-select v-model:value="form.protocol" :options="protocolOptions" />
      </n-form-item>
      <n-form-item :label="t('model.baseUrl')">
        <n-input
          :value="form.baseUrl"
          :placeholder="PROTOCOLS.find((p) => p.value === form.protocol)?.baseUrlHint"
          @update:value="form.baseUrl = $event"
        />
      </n-form-item>
      <n-form-item :label="t('model.apiKey')">
        <n-input
          :value="form.apiKey"
          type="password"
          show-password-on="click"
          :placeholder="isEdit() ? t('model.apiKeyKeep') : t('model.apiKeyPh')"
          @update:value="form.apiKey = $event"
        />
        <template v-if="isEdit() && provider && provider.apiKeyHint" #feedback>
          <span class="mono">{{ provider.apiKeyHint }}</span>
          · {{ t('model.apiKeyKeepHint') }}
        </template>
      </n-form-item>
      <n-form-item :label="t('model.organization')">
        <n-input
          :value="form.organization"
          :placeholder="t('model.organizationPh')"
          @update:value="form.organization = $event"
        />
      </n-form-item>
      <n-form-item v-if="scope === 'platform'" :label="t('model.sortOrder')">
        <n-input-number v-model:value="form.sortOrder" :min="0" :max="9999" style="width: 100%" />
      </n-form-item>
      <n-form-item v-else :label="t('model.isActive')">
        <n-switch v-model:value="form.isActive">
          <template #checked>{{ t('model.enabled') }}</template>
          <template #unchecked>{{ t('model.disabled') }}</template>
        </n-switch>
      </n-form-item>
    </n-form>
    <template #footer>
      <div class="modal-foot">
        <n-button quaternary @click="emit('update:show', false)">{{ t('common.cancel') }}</n-button>
        <n-button type="primary" :loading="saving" @click="submit">
          <template #icon><n-icon :component="CreateOutline" /></template>
          {{ t('common.save') }}
        </n-button>
      </div>
    </template>
  </n-modal>
</template>
