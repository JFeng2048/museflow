<script setup lang="ts">
import { computed } from 'vue'
import { NConfigProvider, NMessageProvider, NDialogProvider, darkTheme } from 'naive-ui'
import { useTheme } from '@/composables/useTheme'

const { isDark } = useTheme()

const theme = computed(() => (isDark.value ? darkTheme : null))

// 轻/暗共用一套主题微调，主色取自品牌靛蓝。
const themeOverrides = {
  common: {
    primaryColor: '#3b5bdb',
    primaryColorHover: '#4c6ef5',
    primaryColorPressed: '#364fc7',
    primaryColorSuppl: '#4c6ef5',
    borderRadius: '8px',
    fontSize: '14px',
  },
  Card: { borderRadius: '12px' },
  Tag: { borderRadius: '999px' },
}
</script>

<template>
  <n-config-provider :theme="theme" :theme-overrides="themeOverrides">
    <n-message-provider :max="3">
      <n-dialog-provider>
        <router-view />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>
