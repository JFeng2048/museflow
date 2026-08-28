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
