<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NGrid, NGi, NButton, NInput, NSelect, NEmpty, NIcon } from 'naive-ui'
import { SearchOutline, AddOutline } from '@vicons/ionicons5'
import { storeToRefs } from 'pinia'
import { useNovelStore } from '@/stores/novel'
import NovelCard from '@/components/novel/NovelCard.vue'
import NovelCreate from './NovelCreate.vue'
import type { NovelStatus } from '@/types'

const route = useRoute()
const router = useRouter()
const novelStore = useNovelStore()
const { novels, loading } = storeToRefs(novelStore)

const keyword = ref('')
const statusFilter = ref<NovelStatus | 'all'>('all')
const showCreate = ref(false)

const statusOptions = [
  { label: '全部状态', value: 'all' },
  { label: '草稿', value: 'draft' },
  { label: '连载中', value: 'serializing' },
  { label: '已完结', value: 'completed' },
  { label: '已暂停', value: 'paused' },
]

const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  return novels.value.filter((n) => {
    const matchKw =
      !kw ||
      n.title.toLowerCase().includes(kw) ||
      n.description.toLowerCase().includes(kw) ||
      n.tags.some((t) => t.toLowerCase().includes(kw))
    const matchStatus = statusFilter.value === 'all' || n.status === statusFilter.value
    return matchKw && matchStatus
  })
})

onMounted(() => {
  novelStore.loadNovels()
  if (route.query.create === '1') {
    showCreate.value = true
    router.replace({ query: {} })
  }
})
</script>

<template>
  <div class="page">
    <header class="page__head">
      <div>
        <h1 class="page__title">我的项目</h1>
        <p class="page__sub">管理你的小说项目，从这里开始每一段故事。</p>
      </div>
      <n-button type="primary" @click="showCreate = true">
        <template #icon><n-icon :component="AddOutline" /></template>
        新建项目
      </n-button>
    </header>

    <div class="toolbar">
      <n-input
        v-model:value="keyword"
        placeholder="搜索标题、简介或标签"
        clearable
        style="max-width: 280px"
      >
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>
      <n-select v-model:value="statusFilter" :options="statusOptions" style="width: 160px" />
      <span class="count">共 {{ filtered.length }} 个项目</span>
    </div>

    <n-grid
      v-if="filtered.length"
      :cols="4"
      :x-gap="16"
      :y-gap="16"
      responsive="screen"
      item-responsive
    >
      <n-gi v-for="novel in filtered" :key="novel.id" span="4 s:2 l:1">
        <novel-card :novel="novel" />
      </n-gi>
    </n-grid>
    <n-empty v-else :description="loading ? '加载中…' : '没有匹配的项目'" />

    <NovelCreate v-model:show="showCreate" />
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
</style>
