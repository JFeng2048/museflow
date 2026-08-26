<script setup lang="ts">
import { computed } from 'vue'
import { NDropdown } from 'naive-ui'
import { NIcon } from 'naive-ui'
import { ColorPaletteOutline } from '@vicons/ionicons5'
import { useUiStore } from '@/stores/ui'
import { themes } from '@/themes'

const ui = useUiStore()

const options = themes.map((t) => ({
  label: t.name,
  key: t.id,
  props: { title: t.description },
}))

const label = computed(() => ui.currentTheme.name)

function onSelect(key: string) {
  ui.setTheme(key)
}
</script>

<template>
  <NDropdown :options="options" @select="onSelect" trigger="click">
    <button class="theme-trigger" :title="`主题：${ui.currentTheme.name}`">
      <n-icon :component="ColorPaletteOutline" class="text-[15px]" />
      <span>{{ label }}</span>
    </button>
  </NDropdown>
</template>
