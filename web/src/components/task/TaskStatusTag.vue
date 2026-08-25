<script setup lang="ts">
import { computed } from 'vue'
import type { TaskStatus } from '@/types'
import { taskStatusMeta } from '@/utils/task'

const props = defineProps<{ status: TaskStatus }>()
const meta = computed(() => taskStatusMeta(props.status))
</script>

<template>
  <n-tag :type="meta.type" size="small" round>{{ meta.label }}</n-tag>
</template>
