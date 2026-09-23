<script setup lang="ts">
import { computed } from 'vue'
import { NIcon, NTooltip } from 'naive-ui'
import { KeyOutline, LockClosedOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { maskApiKey } from '@/types/model'

/**
 * 密钥槽位（模型配置模块的签名元素）。
 *
 * 契约上 api_key 是只写字段：没有任何接口能回显完整密钥，只有末 4 位与轮换时间。
 * 因此这里把密钥呈现成一条「槽位」而不是一个可读的输入值——已配置时是填充的槽，
 * 未配置时是虚线空槽；紧凑形态（表格单元格）只显示掩码，完整形态附带轮换时间，
 * 提示密钥只能通过重新填写来更换。
 */
const props = withDefaults(
  defineProps<{
    /** api_key_hint：末 4 位；空串表示尚未配置密钥。 */
    hint?: string
    /** 最近一次写入密钥的时间（ISO）；空串表示从未配置。 */
    updatedAt?: string
    /** 紧凑形态用于表格单元格，只显示掩码。 */
    compact?: boolean
  }>(),
  { hint: '', updatedAt: '', compact: false },
)

const { t } = useI18n()

const configured = computed(() => !!props.hint)
const masked = computed(() => maskApiKey(props.hint))
const rotatedAt = computed(() => (props.updatedAt ? new Date(props.updatedAt).toLocaleString() : ''))

/** 槽位的说明文案：紧凑形态走 tooltip，完整形态直接显示在槽内。 */
const summary = computed(() =>
  configured.value
    ? rotatedAt.value
      ? t('model.keyRotatedAt', { at: rotatedAt.value })
      : t('model.keyConfigured')
    : t('model.keyNotConfigured'),
)
</script>

<template>
  <n-tooltip :disabled="compact" trigger="hover">
    <template #trigger>
      <div class="key-slot" :class="{ 'is-empty': !configured, 'is-compact': compact }">
        <n-icon :component="configured ? KeyOutline : LockClosedOutline" class="key-slot-ico" />
        <span v-if="configured" class="key-slot-val">{{ masked }}</span>
        <span v-else class="key-slot-val key-slot-none">{{ t('model.keyEmpty') }}</span>
        <span v-if="configured && !compact && rotatedAt" class="key-slot-time">{{ rotatedAt }}</span>
      </div>
    </template>
    {{ summary }}
  </n-tooltip>
</template>
