<script setup lang="ts">
import { useRouter } from 'vue-router'
import { formatWords, formatRelative } from '@/utils/format'
import NovelStatusTag from './NovelStatusTag.vue'
import type { Novel } from '@/types'

const props = defineProps<{ novel: Novel }>()
const router = useRouter()

function open() {
  router.push({ name: 'novel-detail', params: { id: props.novel.id } })
}
</script>

<template>
  <n-card hoverable class="novel-card" @click="open">
    <div class="cover">{{ novel.title.slice(0, 1) }}</div>
    <div class="body">
      <div class="head">
        <span class="title">{{ novel.title }}</span>
        <NovelStatusTag :status="novel.status" />
      </div>
      <p class="desc">{{ novel.description }}</p>
      <div class="meta">
        <span>{{ formatWords(novel.wordCount) }}</span>
        <span>· {{ novel.chapterCount }} 章</span>
        <span>· {{ formatRelative(novel.updatedAt) }}</span>
      </div>
      <div class="tags">
        <n-tag v-for="t in novel.tags" :key="t" size="tiny" :bordered="false" type="info">{{
          t
        }}</n-tag>
      </div>
    </div>
  </n-card>
</template>

<style scoped>
.novel-card {
  cursor: pointer;
  border-radius: 12px;
}
.cover {
  height: 96px;
  border-radius: 10px;
  background: linear-gradient(135deg, var(--mf-accent), var(--mf-accent-hover));
  color: #fff;
  font-size: 40px;
  font-weight: 700;
  display: grid;
  place-items: center;
  margin-bottom: 12px;
}
.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.title {
  font-weight: 600;
  font-size: 15px;
  color: var(--mf-text);
}
.desc {
  color: var(--mf-text-3);
  font-size: 13px;
  line-height: 1.6;
  margin: 8px 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.meta {
  display: flex;
  gap: 6px;
  color: var(--mf-text-3);
  font-size: 12px;
}
.tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 10px;
}
</style>
