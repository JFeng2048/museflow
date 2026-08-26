<script setup lang="ts">
import { NDropdown } from 'naive-ui'
import { NIcon } from 'naive-ui'
import { LanguageOutline } from '@vicons/ionicons5'
import { useUiStore } from '@/stores/ui'
import type { AppLocale } from '@/i18n'

const ui = useUiStore()

const options = [
  { label: '中文', key: 'zh' },
  { label: 'English', key: 'en' },
]

function onSelect(key: string) {
  ui.setLang(key as AppLocale)
}
</script>

<template>
  <NDropdown :options="options" @select="onSelect" trigger="click">
    <button class="lang-trigger" :title="ui.currentLang === 'zh' ? '语言：中文' : 'Language: English'">
      <n-icon :component="LanguageOutline" class="text-[15px]" />
      <span>{{ ui.currentLang === 'zh' ? '中文' : 'English' }}</span>
    </button>
  </NDropdown>
</template>
