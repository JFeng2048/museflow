<script setup lang="ts">
import { computed } from 'vue'
import type { NovelStatus } from '@/types'

const props = defineProps<{ status: NovelStatus }>()

const map: Record<NovelStatus, { label: string; type: 'default' | 'success' | 'warning' | 'error' | 'info' }> = {
  draft: { label: '草稿', type: 'default' },
  serializing: { label: '连载中', type: 'info' },
  completed: { label: '已完结', type: 'success' },
  paused: { label: '已暂停', type: 'warning' },
}

const meta = computed(() => map[props.status])
</script>

<template>
  <n-tag :type="meta.type" size="small" round>{{ meta.label }}</n-tag>
</template>
