<script setup lang="ts">
import { NIcon, NTooltip } from 'naive-ui'
import type { ModelCapabilities } from '@/types/model'
import { CAPABILITIES } from '@/types/model'
import { useI18n } from 'vue-i18n'

/**
 * 模型能力刻度行。
 *
 * 四个能力（流式 / 工具 / JSON / 图像）用等宽图标呈现：开启的填充琥珀色，
 * 未开启的留空线框。表格里四列文字太占宽度，图标行能在不损失信息的前提下
 * 把密度提上来，每格都带 tooltip 说明是哪项能力。
 */
defineProps<{
  capabilities: ModelCapabilities
}>()

const { t } = useI18n()
</script>

<template>
  <div class="cap-row">
    <n-tooltip v-for="cap in CAPABILITIES" :key="cap.key" trigger="hover">
      <template #trigger>
        <span class="cap-cell" :class="{ on: capabilities[cap.key] }">
          <n-icon :component="cap.icon" :size="14" />
        </span>
      </template>
      {{ t('model.capability.' + cap.key) }}
      {{ capabilities[cap.key] ? t('model.capabilityOn') : t('model.capabilityOff') }}
    </n-tooltip>
  </div>
</template>
