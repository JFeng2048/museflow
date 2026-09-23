<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import {
  NButton,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NModal,
  NSelect,
  NSwitch,
  useMessage,
} from 'naive-ui'
import { CreateOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import type { SystemSetting } from '@/types/model'
import { VALUE_TYPES } from '@/types/model'

/**
 * 系统配置写入弹窗。
 *
 * (config_group, key) 是配置的身份，创建后可读不可改；真正要小心的是机密项：
 *  - is_secret 打开时只填 secret_value，后端落加密列，明文只在保存响应里回显一次；
 *  - is_secret 关闭时只填 value，后端按 value_type 序列化进 jsonb 列；
 *  - 编辑已有机密项时留空，表示沿用旧密钥——改一句说明不必重填一遍密钥。
 * 弹窗因此把两个输入框做成互斥的一组，避免「两个都填」被后端拒绝才知道填错。
 *
 * submit 抛出三段：配置体（camelCase 领域字段）+ 分组 + 键名。
 * 分组与键名只在新增时由弹窗收集，编辑时调用方从行数据里就有——
 * 分开传是为了不让「身份」混进「值」，写入接口按 (config_group, key) 定位。
 */
const props = withDefaults(
  defineProps<{
    show: boolean
    /** 传入表示编辑，缺省表示新增。 */
    setting?: SystemSetting | null
  }>(),
  { setting: null },
)

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'submit', payload: Record<string, unknown>, group: string, key: string): void
}>()

const { t } = useI18n()
const message = useMessage()

const isEdit = () => !!props.setting
const valueTypeOptions = VALUE_TYPES.map((v) => ({ label: t(`model.valueType.${v}`), value: v }))

const form = reactive({
  configGroup: '',
  key: '',
  isSecret: false,
  value: '',
  /** 机密项只写不读：编辑时永远从空开始，留空即不改。 */
  secretValue: '',
  valueType: 'string',
  isPublic: false,
  description: '',
})

watch(
  () => props.show,
  (open) => {
    if (!open) return
    const s = props.setting
    Object.assign(form, {
      configGroup: s?.configGroup ?? '',
      key: s?.key ?? '',
      isSecret: s?.isSecret ?? false,
      value: s && !s.isSecret ? s.value : '',
      secretValue: '',
      valueType: s?.valueType ?? 'string',
      isPublic: s?.isPublic ?? false,
      description: s?.description ?? '',
    })
  },
)

const saving = ref(false)

function submit() {
  const group = form.configGroup.trim()
  const key = form.key.trim()
  if (!group || !key) {
    message.warning(t('model.validate.groupKeyRequired'))
    return
  }
  const payload: Record<string, unknown> = {
    isSecret: form.isSecret,
    valueType: form.valueType,
    isPublic: form.isPublic,
    description: form.description.trim(),
  }
  if (form.isSecret) {
    // 新增机密项必须给值；编辑时留空即沿用旧密钥（后端不会把空串写成新密钥）。
    if (!form.secretValue.trim() && !isEdit()) {
      message.warning(t('model.validate.secretRequired'))
      return
    }
    payload.secretValue = form.secretValue.trim()
  } else {
    if (!form.value.trim()) {
      message.warning(t('model.validate.valueRequired'))
      return
    }
    payload.value = form.value.trim()
  }
  saving.value = true
  emit('submit', payload, group, key)
  saving.value = false
}
</script>

<template>
  <n-modal
    :show="show"
    :title="isEdit() ? t('model.editSetting') : t('model.addSetting')"
    preset="card"
    style="width: 580px; max-width: 92vw"
    :bordered="false"
    @update:show="emit('update:show', $event)"
  >
    <n-form label-placement="top">
      <div class="form-pair">
        <n-form-item :label="t('model.configGroup')">
          <n-input
            :value="form.configGroup"
            :disabled="isEdit()"
            :placeholder="t('model.configGroupPh')"
            @update:value="form.configGroup = $event"
          />
        </n-form-item>
        <n-form-item :label="t('model.settingKey')">
          <n-input
            :value="form.key"
            :disabled="isEdit()"
            :placeholder="t('model.settingKeyPh')"
            @update:value="form.key = $event"
          />
        </n-form-item>
      </div>
      <n-form-item :label="t('model.isSecret')">
        <n-switch v-model:value="form.isSecret">
          <template #checked>{{ t('model.secretOn') }}</template>
          <template #unchecked>{{ t('model.secretOff') }}</template>
        </n-switch>
      </n-form-item>
      <n-form-item v-if="form.isSecret" :label="t('model.secretValue')">
        <n-input
          v-model:value="form.secretValue"
          type="password"
          show-password-on="click"
          :placeholder="isEdit() ? t('model.secretKeep') : t('model.secretValuePh')"
        />
        <template v-if="isEdit()" #feedback>{{ t('model.secretKeepHint') }}</template>
      </n-form-item>
      <template v-else>
        <n-form-item :label="t('model.valueType')">
          <n-select v-model:value="form.valueType" :options="valueTypeOptions" />
        </n-form-item>
        <n-form-item :label="t('model.value')">
          <n-input
            v-model:value="form.value"
            :type="form.valueType === 'json' ? 'textarea' : 'text'"
            :autosize="form.valueType === 'json' ? { minRows: 3, maxRows: 8 } : false"
            :placeholder="t('model.valuePh.' + form.valueType)"
          />
        </n-form-item>
      </template>
      <n-form-item :label="t('model.isPublic')">
        <n-switch v-model:value="form.isPublic">
          <template #checked>{{ t('model.publicOn') }}</template>
          <template #unchecked>{{ t('model.publicOff') }}</template>
        </n-switch>
      </n-form-item>
      <n-form-item :label="t('model.description')">
        <n-input
          v-model:value="form.description"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 4 }"
          :placeholder="t('model.descriptionPh')"
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
