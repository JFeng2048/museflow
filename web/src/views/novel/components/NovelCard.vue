<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { formatWords, formatRelative, shade, readableOn } from '@/utils/format'
import NovelStatusTag from './NovelStatusTag.vue'
import { useI18n } from 'vue-i18n'
import type { Novel } from '@/types'
const { t } = useI18n()
const props = defineProps<{ novel: Novel }>()
const router = useRouter()

const baseColor = computed(() => props.novel.coverColor || '#d4a05a')
const coverGradient = computed(() => `linear-gradient(140deg, ${baseColor.value}, ${shade(baseColor.value, -22)})`)
const initial = computed(() => props.novel.title.slice(0, 1))
const inkOn = computed(() => readableOn(baseColor.value))

const progress = computed(() => {
  const goal = props.novel.wordGoal || 0
  if (!goal) return 0
  return Math.min(100, Math.round((props.novel.wordCount / goal) * 100))
})

function open() {
  router.push({ name: 'novel-detail', params: { id: props.novel.id } })
}
</script>

<template>
  <n-card hoverable class="novel-card" @click="open">
    <div class="cover" :style="{ background: coverGradient, color: inkOn }">
      <span class="cover-watermark">{{ initial }}</span>
      <span class="cover-genre">{{ novel.genre }}</span>
      <div class="cover-status"><NovelStatusTag :status="novel.status" /></div>
    </div>

    <div class="body">
      <h3 class="title">{{ novel.title }}</h3>
      <p class="premise">{{ novel.premise || novel.description }}</p>

      <div class="meta">
        <span>{{ formatWords(novel.wordCount) }}</span>
        <span class="dot">·</span>
        <span>{{ novel.chapterCount }} 章</span>
        <span class="dot">·</span>
        <span>{{ formatRelative(novel.updatedAt) }}</span>
      </div>

      <div v-if="novel.wordGoal" class="progress">
        <div class="progress-track"><div class="progress-fill" :style="{ width: progress + '%' }" /></div>
        <span class="progress-label">{{ progress }}%</span>
      </div>

      <div class="tags">
        <n-tag v-for="tg in novel.tags" :key="tg" size="tiny" :bordered="false" type="info">{{ tg }}</n-tag>
      </div>
    </div>
  </n-card>
</template>

<style scoped>
.novel-card {
  border-radius: 16px;
  overflow: hidden;
  transition: transform 0.18s ease, box-shadow 0.18s ease;
  cursor: pointer;
}
.novel-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 14px 34px rgba(26, 35, 50, 0.14);
}
.cover {
  position: relative;
  height: 128px;
  border-radius: 12px;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 14px;
}
.cover-watermark {
  font-family: var(--font-serif);
  font-size: 64px;
  font-weight: 700;
  line-height: 1;
  opacity: 0.9;
  user-select: none;
}
.cover-genre {
  position: absolute;
  left: 12px;
  bottom: 10px;
  font-size: 12px;
  letter-spacing: 0.08em;
  opacity: 0.85;
}
.cover-status {
  position: absolute;
  right: 10px;
  top: 10px;
  filter: drop-shadow(0 1px 2px rgba(0, 0, 0, 0.25));
}
.body {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--c-ink);
  font-family: var(--font-sans);
}
.premise {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--c-ink-soft);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  min-height: 40px;
}
.meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
  font-size: 12px;
  color: var(--c-ink-muted);
}
.meta .dot {
  opacity: 0.5;
}
.progress {
  display: flex;
  align-items: center;
  gap: 8px;
}
.progress-track {
  flex: 1;
  height: 5px;
  border-radius: 999px;
  background: var(--c-line);
  overflow: hidden;
}
.progress-fill {
  height: 100%;
  border-radius: 999px;
  background: linear-gradient(90deg, var(--c-amber), var(--c-amber-deep));
  transition: width 0.3s ease;
}
.progress-label {
  font-size: 11px;
  color: var(--c-ink-muted);
  font-variant-numeric: tabular-nums;
}
.tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
</style>
