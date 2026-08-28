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
