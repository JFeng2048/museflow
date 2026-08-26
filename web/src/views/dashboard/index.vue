<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NIcon } from 'naive-ui'
import { BookOutline, BulbOutline, FlashOutline } from '@vicons/ionicons5'
import { useNovelStore } from '@/stores/novel'
import { useUserStore } from '@/stores/system/user'

const { t } = useI18n()
const router = useRouter()
const novelStore = useNovelStore()
const userStore = useUserStore()

const userId = computed(() => userStore.user?.id || 'u_demo')

onMounted(async () => {
  if (!novelStore.novels.length) await novelStore.loadNovels()
})

const novels = computed(() => novelStore.byUser(userId.value))
const stats = computed(() => novelStore.userStats(userId.value) || {})
const recent = computed(() => [...novels.value].sort((a, b) => +new Date(b.updatedAt) - +new Date(a.updatedAt)).slice(0, 4))

const greetingKey = (() => {
  const h = new Date().getHours()
  if (h < 6) return 'dashboard.greetingNight'
  if (h < 11) return 'dashboard.greetingMorning'
  if (h < 14) return 'dashboard.greetingNoon'
  if (h < 18) return 'dashboard.greetingAfternoon'
  return 'dashboard.greetingEvening'
})()

const quickActions = [
  { to: '/novels', labelKey: 'dashboard.continueWrite', icon: BookOutline },
  { to: '/inspiration', labelKey: 'dashboard.findInspiration', icon: BulbOutline },
  { to: '/task', labelKey: 'dashboard.aiTools', icon: FlashOutline },
]
</script>

<template>
  <div class="px-10 py-9 max-w-[1080px] mx-auto">
    <header class="dash-hero">
      <p class="dash-eyebrow">{{ t('dashboard.eyebrow') }}</p>
      <h1>{{ t(greetingKey) }}</h1>
      <p class="dash-sub">{{ t('dashboard.sub') }}</p>
      <div class="dash-quick">
        <button v-for="q in quickActions" :key="q.to" class="dash-q" @click="router.push(q.to)">
          <n-icon :component="q.icon" class="text-[15px]" />
          {{ t(q.labelKey) }}
        </button>
      </div>
    </header>

    <div class="grid grid-cols-4 gap-4 my-6">
      <div class="dash-kpi">
        <p class="dash-kpi-n">{{ stats.totalNovels ?? 0 }}</p>
        <p class="dash-kpi-l">{{ t('dashboard.works') }}</p>
      </div>
      <div class="dash-kpi">
        <p class="dash-kpi-n">{{ Math.round((stats.totalWords ?? 0) / 100) / 10 }}k</p>
        <p class="dash-kpi-l">{{ t('dashboard.totalWords') }}</p>
      </div>
      <div class="dash-kpi">
        <p class="dash-kpi-n">{{ stats.statusOngoing ?? 0 }}</p>
        <p class="dash-kpi-l">{{ t('dashboard.ongoing') }}</p>
      </div>
      <div class="dash-kpi">
        <p class="dash-kpi-n">{{ stats.statusCompleted ?? 0 }}</p>
        <p class="dash-kpi-l">{{ t('dashboard.completed') }}</p>
      </div>
    </div>

    <section class="dash-recent">
      <div class="flex items-center justify-between mb-4">
        <h3 class="text-[18px] m-0">{{ t('dashboard.recent') }}</h3>
        <button class="dash-more" @click="router.push('/novels')">{{ t('dashboard.viewAll') }} →</button>
      </div>
      <div v-if="recent.length" class="grid grid-cols-[repeat(auto-fill,minmax(200px,1fr))] gap-3.5">
        <button
          v-for="n in recent"
          :key="n.id"
          class="dash-rc"
          @click="router.push({ name: 'novel-detail', params: { id: n.id } })"
        >
          <span class="dash-cov" :style="{ background: n.coverColor }">{{ n.title.slice(0, 1) }}</span>
          <span class="font-semibold text-[15px]">{{ n.title }}</span>
          <span class="text-[12px] text-ink-muted">{{ n.chapterCount }} 章 · {{ Math.round(n.wordCount / 100) / 10 }}k {{ t('dashboard.wordsUnit') }}</span>
        </button>
      </div>
      <div v-else class="text-center text-ink-muted py-5 text-[14px]">
        <p>{{ t('dashboard.emptyNovel') }}<button class="dash-link" @click="router.push('/novels')">{{ t('dashboard.emptyLink') }} →</button></p>
      </div>
    </section>
  </div>
</template>

