<script setup lang="ts">
import { computed } from 'vue'
import { NConfigProvider, NMessageProvider, NDialogProvider } from 'naive-ui'
import { useUiStore } from '@/stores/ui'

const ui = useUiStore()

// 跟随当前主题的 Naive UI 覆盖：直接读取 :root 上由 CSS 定义的语义变量，
// 保证 Naive UI 与 Tailwind 主题完全一致（单一色源）。
function cssVar(name: string, fallback = '#000000'): string {
  if (typeof window === 'undefined') return fallback
  const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  return v || fallback
}

const themeOverrides = computed(() => ({
  common: {
    primaryColor: cssVar('--c-amber-deep', '#b9853f'),
    primaryColorHover: cssVar('--c-amber', '#d4a05a'),
    primaryColorPressed: cssVar('--c-amber-deep', '#b9853f'),
    primaryColorSuppl: cssVar('--c-amber', '#d4a05a'),
    borderRadius: '10px',
    fontSize: '14px',
    bodyColor: cssVar('--c-warm', '#f8f5f0'),
    cardColor: cssVar('--c-paper', '#fffdf9'),
    textColorBase: cssVar('--c-ink', '#1a2332'),
  },
  Card: { borderRadius: '14px' },
  Tag: { borderRadius: '999px' },
}))
</script>

<template>
  <n-config-provider :theme-overrides="themeOverrides">
    <n-message-provider :max="3">
      <n-dialog-provider>
        <router-view />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>
