<script setup lang="ts">
import { NIcon } from 'naive-ui'
import { FlaskOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'

/**
 * 演示数据提示条。
 *
 * 渲染本地示例数据的页面必须在首屏挂上它：这些数字没有后端来源，
 * 用户不该把它们当成自己账号里的真实状态。
 */
defineProps<{
  /** 覆盖默认说明，点明这页具体缺哪一段后端。 */
  note?: string
}>()

const { t } = useI18n()
</script>

<template>
  <p class="demo-note">
    <n-icon :component="FlaskOutline" :size="14" class="demo-note-ico" />
    <span class="demo-note-label">{{ t('demo.badge') }}</span>
    <span class="demo-note-text">{{ note || t('demo.note') }}</span>
  </p>
</template>
