<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { NovelStatus } from '@/types'
import { NOVEL_STATUS_META } from '../constants'

const props = defineProps<{ status: NovelStatus }>()
const { t } = useI18n()

const meta = computed(() => NOVEL_STATUS_META[props.status])
</script>

<template>
  <n-tag :type="meta.type" size="small" round>{{ t(meta.labelKey) }}</n-tag>
</template>
