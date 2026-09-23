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
import type { AIModel, ModelCapabilities, ModelType, UserModel } from '@/types/model'
import { CAPABILITIES, EMPTY_CAPABILITIES, MODEL_TYPES } from '@/types/model'

/**
 * 模型表单弹窗：平台模型（管理端）与我的自定义模型（用户端）共用。
 *
 * 与渠道弹窗一样用 scope 区分两侧，差别集中在三处：
 *  - platform 多一个模型编码（创建后可读不可改）、积分单价与排序权重；
 *  - user     多一个启用开关，没有编码与定价（费用由用户直接付给上游）；
 *  - 归属渠道一律从现有渠道里选，创建后不可改——换渠道等于换凭证。
 *
 * 能力开关用四个 NSwitch 平铺，而不是四列图标：表单里要点，图标不够明确。
 */
const props = withDefaults(
  defineProps<{
    show: boolean
    /** 传入表示编辑，缺省表示新增。 */
    model?: AIModel | UserModel | null
    scope?: 'platform' | 'user'
    /** 可选渠道下拉项：平台渠道来自管理端，自定义渠道来自当前用户。 */
    providers?: { label: string; value: number }[]
  }>(),
  { model: null, scope: 'platform', providers: () => [] },
)

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'submit', payload: Record<string, unknown>): void
}>()

const { t } = useI18n()
const message = useMessage()

const isEdit = () => !!props.model
const modelTypeOptions = MODEL_TYPES.map((m) => ({ label: t(`model.type.${m.value}`), value: m.value }))

const form = reactive({
  providerId: 0,
  code: '',
  name: '',
  modelType: 'chat' as ModelType,
  apiModel: '',
  contextWindow: 0,
  maxOutputTokens: 0,
  capabilities: { ...EMPTY_CAPABILITIES } as ModelCapabilities,
  creditCost: 0,
  description: '',
  sortOrder: 0,
  isActive: true,
})

// 打开弹窗时按「编辑 / 新增」重置表单；编码与渠道只读，不做回填。
watch(
  () => props.show,
  (open) => {
    if (!open) return
    const m = props.model
    Object.assign(form, {
      providerId: m?.providerId ?? props.providers[0]?.value ?? 0,
      code: m && 'code' in m ? m.code : '',
      name: m?.name ?? '',
      modelType: m?.modelType ?? 'chat',
      apiModel: m?.apiModel ?? '',
      contextWindow: m?.contextWindow ?? 0,
      maxOutputTokens: m?.maxOutputTokens ?? 0,
      capabilities: m?.capabilities ? { ...m.capabilities } : { ...EMPTY_CAPABILITIES },
      creditCost: m && 'creditCost' in m ? m.creditCost : 0,
      description: m?.description ?? '',
      sortOrder: m && 'sortOrder' in m ? m.sortOrder : 0,
      isActive: m ? m.isActive : true,
    })
  },
)

const saving = ref(false)

function submit() {
  if (!form.name.trim()) {
    message.warning(t('model.validate.nameRequired'))
    return
  }
  if (!form.apiModel.trim()) {
    message.warning(t('model.validate.apiModelRequired'))
    return
  }
  if (!form.providerId) {
    message.warning(t('model.validate.providerRequired'))
    return
  }
  if (!isEdit() && props.scope === 'platform' && !form.code.trim()) {
    message.warning(t('model.validate.codeRequired'))
    return
  }
  saving.value = true
  const payload: Record<string, unknown> = {
    providerId: form.providerId,
    name: form.name.trim(),
    modelType: form.modelType,
    apiModel: form.apiModel.trim(),
    contextWindow: form.contextWindow,
    maxOutputTokens: form.maxOutputTokens,
    capabilities: { ...form.capabilities },
    description: form.description.trim(),
  }
  if (props.scope === 'platform') {
    payload.creditCost = form.creditCost
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
          ? t('model.editModel')
          : t('model.editMyModel')
        : scope === 'platform'
          ? t('model.addModel')
          : t('model.addMyModel')
    "
    preset="card"
    style="width: 620px; max-width: 92vw"
    :bordered="false"
    @update:show="emit('update:show', $event)"
  >
    <n-form label-placement="top">
      <n-form-item :label="t('model.provider')">
        <n-select
          :value="form.providerId"
          :options="providers"
          :disabled="isEdit()"
          :placeholder="t('model.providerPh')"
          @update:value="form.providerId = $event"
        />
        <template v-if="isEdit()" #feedback>{{ t('model.providerImmutable') }}</template>
      </n-form-item>
      <n-form-item v-if="scope === 'platform'" :label="t('model.modelCode')">
        <n-input
          :value="form.code"
          :disabled="isEdit()"
          :placeholder="t('model.modelCodePh')"
          @update:value="form.code = $event"
        />
        <template v-if="isEdit()" #feedback>{{ t('model.codeImmutable') }}</template>
      </n-form-item>
      <n-form-item :label="t('model.modelName')">
        <n-input
          :value="form.name"
          :placeholder="t('model.modelNamePlaceholder')"
          @update:value="form.name = $event"
        />
      </n-form-item>
      <n-form-item :label="t('model.modelType')">
        <n-select v-model:value="form.modelType" :options="modelTypeOptions" />
      </n-form-item>
      <n-form-item :label="t('model.apiModel')">
        <n-input
          :value="form.apiModel"
          :placeholder="t('model.apiModelPlaceholder')"
          @update:value="form.apiModel = $event"
        />
      </n-form-item>
      <div class="form-pair">
        <n-form-item :label="t('model.contextWindow')">
          <n-input-number v-model:value="form.contextWindow" :min="0" :max="10000000" style="width: 100%">
            <template #suffix>tok</template>
          </n-input-number>
        </n-form-item>
        <n-form-item :label="t('model.maxOutputTokens')">
          <n-input-number v-model:value="form.maxOutputTokens" :min="0" :max="1000000" style="width: 100%">
            <template #suffix>tok</template>
          </n-input-number>
        </n-form-item>
      </div>
      <n-form-item :label="t('model.capabilitiesTitle')">
        <div class="cap-form">
          <n-switch v-for="cap in CAPABILITIES" :key="cap.key" v-model:value="form.capabilities[cap.key]" size="small">
            <template #checked>{{ t('model.capability.' + cap.key) }}</template>
            <template #unchecked>{{ t('model.capability.' + cap.key) }}</template>
          </n-switch>
        </div>
      </n-form-item>
      <div class="form-pair">
        <n-form-item v-if="scope === 'platform'" :label="t('model.creditCost')">
          <n-input-number v-model:value="form.creditCost" :min="0" :max="999999" style="width: 100%" />
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
      </div>
      <n-form-item :label="t('model.description')">
        <n-input
          :value="form.description"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 5 }"
          :placeholder="t('model.descriptionPh')"
          @update:value="form.description = $event"
        />
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
