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
import { SearchOutline, DownloadOutline } from '@vicons/ionicons5'
import { fetchMaterials, importMaterial } from '@/api/material'
import type { Material, MaterialType } from '@/types'

const message = useMessage()
const materials = ref<Material[]>([])
const keyword = ref('')
const typeFilter = ref<MaterialType | 'all'>('all')

const typeOptions = [
  { label: '全部类型', value: 'all' },
  { label: '语录', value: 'quote' },
  { label: '世界观', value: 'world' },
  { label: '情节', value: 'plot' },
  { label: '角色', value: 'character' },
  { label: '图片', value: 'image' },
]

const typeLabelMap: Record<MaterialType, string> = {
  quote: '语录',
  world: '世界观',
  plot: '情节',
  character: '角色',
  image: '图片',
}

const typeColorMap: Record<MaterialType, 'default' | 'info' | 'success' | 'warning' | 'error'> = {
  quote: 'default',
  world: 'success',
  plot: 'warning',
  character: 'info',
  image: 'error',
}

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
  if (updated) message.success(`已将「${m.title}」导入设定集`)
}
</script>

<template>
  <div class="page">
    <header class="page__head">
      <div>
        <h1 class="page__title">素材库</h1>
        <p class="page__sub">收集灵感碎片，随时导入你的设定集。</p>
      </div>
    </header>

    <div class="toolbar">
      <n-input
        v-model:value="keyword"
        placeholder="搜索素材标题、内容或标签"
        clearable
        style="max-width: 300px"
      >
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>
      <n-select v-model:value="typeFilter" :options="typeOptions" style="width: 160px" />
      <span class="count">共 {{ filtered.length }} 条素材</span>
    </div>

    <n-grid v-if="filtered.length" :cols="3" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
      <n-gi v-for="m in filtered" :key="m.id" span="3 s:2 l:1">
        <n-card :bordered="false" class="material-card">
          <div class="material__head">
            <span class="material__title">{{ m.title }}</span>
            <n-tag size="small" :bordered="false" :type="typeColorMap[m.type]">{{
              typeLabelMap[m.type]
            }}</n-tag>
          </div>
          <p class="material__source">来源：{{ m.source }}</p>
          <p class="material__content">{{ m.content }}</p>
          <n-space :size="6" class="material__tags">
            <n-tag v-for="t in m.tags" :key="t" size="tiny" :bordered="false">{{ t }}</n-tag>
          </n-space>
          <div class="material__foot">
            <span class="material__status" :class="{ on: m.imported }">
              {{ m.imported ? '已导入设定' : '未导入' }}
            </span>
            <n-button
              size="small"
              :type="m.imported ? 'default' : 'primary'"
              :disabled="m.imported"
              @click="onImport(m)"
            >
              <template #icon><n-icon :component="DownloadOutline" /></template>
              {{ m.imported ? '已导入' : '导入设定' }}
            </n-button>
          </div>
        </n-card>
      </n-gi>
    </n-grid>
    <n-empty v-else description="没有匹配的素材" />
  </div>
</template>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.page__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
}
.page__title {
  margin: 0;
  font-size: 22px;
  font-weight: 600;
}
.page__sub {
  margin: 4px 0 0;
  color: var(--mf-text-3);
}
.toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
}
.count {
  color: var(--mf-text-3);
  font-size: 13px;
  margin-left: auto;
}
.material-card {
  height: 100%;
  display: flex;
  flex-direction: column;
}
.material__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 6px;
}
.material__title {
  font-weight: 600;
  font-size: 15px;
  color: var(--mf-text);
}
.material__source {
  color: var(--mf-text-3);
  font-size: 12px;
  margin: 0 0 8px;
}
.material__content {
  color: var(--mf-text-2);
  font-size: 13px;
  line-height: 1.7;
  margin: 0 0 10px;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.material__tags {
  margin-bottom: 12px;
}
.material__foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: auto;
}
.material__status {
  font-size: 12px;
  color: var(--mf-text-3);
}
.material__status.on {
  color: #2e9e6b;
}
</style>
