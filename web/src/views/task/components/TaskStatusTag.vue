<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { TaskStatus } from '@/types'
import { TASK_STATUS_META } from '../constants'

const props = defineProps<{ status: TaskStatus }>()
const { t } = useI18n()
const meta = computed(() => TASK_STATUS_META[props.status])
</script>

<template>
  <n-tag :type="meta.type" size="small" round>{{ t(meta.labelKey) }}</n-tag>
</template>
