<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NEmpty, useMessage, NTabs, NTabPane } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import { useNovelStore } from '@/stores/novel'
import NovelCard from './components/NovelCard.vue'
import NovelCreate from './NovelCreate.vue'
import { NOVEL_STATUS_TABS } from './constants'
import type { Novel } from '@/types/novel'

const router = useRouter()
const novelStore = useNovelStore()
const message = useMessage()
const { t } = useI18n()

const { novels } = storeToRefs(novelStore)
const tab = ref<string>('all')
const showCreate = ref(false)

onMounted(() => {
  novelStore.loadNovels()
})

// 单用户演示：直接展示全部 mock 作品，按状态标签本地筛选，避免依赖 user.id 匹配。
const list = computed<Novel[]>(() =>
  tab.value === 'all' ? novels.value : novels.value.filter((n) => n.status === tab.value),
)

function openNovel(id: string) {
  router.push({ name: 'novel-detail', params: { id } })
}

function onCreated() {
  message.success(t('novel.createSuccess'))
}
</script>

<template>
  <div class="novel-list">
    <header class="page-head">
      <div>
        <h1 class="title">{{ t('novel.listTitle') }}</h1>
        <p class="subtitle">{{ t('novel.listSub') }}</p>
      </div>
      <NButton type="primary" @click="showCreate = true">{{ t('novel.newWorkAlt') }}</NButton>
    </header>

    <n-tabs v-model:value="tab" type="line" class="status-tabs">
      <n-tab-pane v-for="item in NOVEL_STATUS_TABS" :key="item.value" :name="item.value" :tab="t(item.labelKey)" />
    </n-tabs>

    <div v-if="list.length" class="grid">
      <NovelCard v-for="n in list" :key="n.id" :novel="n" @click="openNovel(n.id)" />
    </div>
    <NEmpty v-else :description="t('novel.listEmpty')" class="empty" />

    <NovelCreate v-model:show="showCreate" @created="onCreated" />
  </div>
</template>

<style scoped>
.novel-list {
  max-width: 1180px;
  margin: 0 auto;
  padding: 8px 4px 60px;
}
.page-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}
.title {
  font-size: 26px;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--c-ink);
  margin: 0;
}
.subtitle {
  margin: 6px 0 0;
  font-size: 14px;
  color: var(--c-ink-muted);
}
.status-tabs {
  margin-bottom: 20px;
}
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 20px;
}
.empty {
  padding: 80px 0;
}
@media (max-width: 640px) {
  .grid {
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 14px;
  }
}
</style>
