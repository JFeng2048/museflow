<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import {
  NGrid,
  NGi,
  NCard,
  NTag,
  NInput,
  NSelect,
  NButton,
  NEmpty,
  NIcon,
  useMessage,
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { SearchOutline, DownloadOutline } from '@vicons/ionicons5'
import { fetchMaterials, importMaterial } from '@/api/material'
import type { Material, MaterialType } from '@/types'
import { MATERIAL_TYPE_OPTIONS, MATERIAL_TYPE_LABEL_KEYS, MATERIAL_TYPE_COLORS } from './constants'

const { t } = useI18n()
const message = useMessage()
const materials = ref<Material[]>([])
const keyword = ref('')
const typeFilter = ref<MaterialType | 'all'>('all')

const typeOptions = computed(() =>
  MATERIAL_TYPE_OPTIONS.map((o) => ({ label: t(o.labelKey), value: o.value })),
)
const typeColorMap = MATERIAL_TYPE_COLORS

const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  return materials.value.filter((m) => {
    const matchKw =
      !kw ||
      m.title.toLowerCase().includes(kw) ||
      m.content.toLowerCase().includes(kw) ||
      m.tags.some((t) => t.toLowerCase().includes(kw))
    const matchType = typeFilter.value === 'all' || m.type === typeFilter.value
    return matchKw && matchType
  })
})

onMounted(async () => {
  materials.value = await fetchMaterials()
})

async function onImport(m: Material) {
  const updated = await importMaterial(m.id)
  const target = materials.value.find((x) => x.id === m.id)
  if (target) target.imported = true
  if (updated) message.success(t('material.importedToast', { title: m.title }))
}
</script>

<template>
  <div class="flex flex-col gap-5">
    <header class="flex items-start justify-between">
      <div>
        <h1 class="text-[22px] font-semibold m-0">{{ t('material.title') }}</h1>
        <p class="text-ink-muted mt-1 mb-0">{{ t('material.subtitle') }}</p>
      </div>
    </header>

    <div class="flex items-center gap-3">
      <n-input
        v-model:value="keyword"
        :placeholder="t('material.search')"
        clearable
        class="max-w-[300px]"
      >
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>
      <n-select v-model:value="typeFilter" :options="typeOptions" class="w-[160px]" />
      <span class="text-ink-muted text-[13px] ml-auto">{{ t('material.count', { n: filtered.length }) }}</span>
    </div>

    <n-grid v-if="filtered.length" :cols="3" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
      <n-gi v-for="m in filtered" :key="m.id" span="3 s:2 l:1">
        <n-card :bordered="false" class="h-full flex flex-col">
          <div class="flex items-center justify-between gap-2 mb-1.5">
            <span class="font-semibold text-[15px] text-ink">{{ m.title }}</span>
            <n-tag size="small" :bordered="false" :type="typeColorMap[m.type]">{{
              t(MATERIAL_TYPE_LABEL_KEYS[m.type])
            }}</n-tag>
          </div>
          <p class="text-ink-muted text-[12px] m-0 mb-2">{{ t('material.source') }}：{{ m.source }}</p>
          <p class="material-content">{{ m.content }}</p>
          <n-space :size="6" class="mb-3">
            <n-tag v-for="t in m.tags" :key="t" size="tiny" :bordered="false">{{ t }}</n-tag>
          </n-space>
          <div class="flex items-center justify-between mt-auto">
            <span class="material-status" :class="{ on: m.imported }">
              {{ m.imported ? t('material.imported') : t('material.notImported') }}
            </span>
            <n-button
              size="small"
              :type="m.imported ? 'default' : 'primary'"
              :disabled="m.imported"
              @click="onImport(m)"
            >
              <template #icon><n-icon :component="DownloadOutline" /></template>
              {{ m.imported ? t('material.importedBtn') : t('material.importBtn') }}
            </n-button>
          </div>
        </n-card>
      </n-gi>
    </n-grid>
    <n-empty v-else :description="t('material.noMatch')" />
  </div>
</template>

