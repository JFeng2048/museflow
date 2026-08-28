<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import {
  NCard,
  NButton,
  NEmpty,
  NSwitch,
  NInput,
  NIcon,
  NPopover,
  useMessage,
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { SendOutline, CheckmarkDoneOutline, TimeOutline, PersonOutline } from '@vicons/ionicons5'
import { fetchChannels } from '@/api/publish'
import { useNovelStore } from '@/stores/novel'
import { storeToRefs } from 'pinia'
import { formatWords } from '@/utils/format'
import type { PublishChannel } from '@/api/publish'

const { t } = useI18n()
const message = useMessage()
const novelStore = useNovelStore()
const { allNovels } = storeToRefs(novelStore)

const channels = ref<PublishChannel[]>([])
const selectedNovelId = ref<string>('')
const selectedChannels = ref<string[]>([])
const note = ref('')
const publishing = ref(false)

// 各平台品牌色，让每张卡有独立个性。
const PLATFORM_COLOR: Record<string, string> = {
  tomato: '#ff5a4d',
  qidian: '#2f6fb3',
  feilu: '#f0883e',
  jjwxc: '#d9488a',
}

const novels = computed(() => novelStore.byUser('u_demo'))
const selectedNovel = computed(() => novels.value.find((n) => n.id === selectedNovelId.value) || null)
const enabledChannels = computed(() => channels.value.filter((c) => c.enabled))

// 模拟发布历史
const history = ref([
  { id: 'h1', title: '星海拾遗者', channel: '番茄小说', status: 'success', at: '2026-08-24 09:12' },
  { id: 'h2', title: '雾港旧事', channel: '起点读书', status: 'success', at: '2026-08-22 20:40' },
  { id: 'h3', title: '墨色人间', channel: '晋江文学城', status: 'pending', at: '2026-08-21 14:05' },
])

onMounted(async () => {
  channels.value = await fetchChannels()
  if (!allNovels.value.length) await novelStore.loadNovels()
  const ws = novels.value
  if (ws.length) selectedNovelId.value = ws[0].id
  selectedChannels.value = channels.value.filter((c) => c.enabled).map((c) => c.id)
})

function toggle(id: string, on: boolean) {
  const ch = channels.value.find((c) => c.id === id)
  if (ch) ch.enabled = on
  if (on && !selectedChannels.value.includes(id)) selectedChannels.value.push(id)
  if (!on) selectedChannels.value = selectedChannels.value.filter((x) => x !== id)
}

function publish() {
  if (!selectedNovel.value) {
    message.warning(t('publish.warnNovel'))
    return
  }
  if (!selectedChannels.value.length) {
    message.warning(t('publish.warnChannel'))
    return
  }
  publishing.value = true
  setTimeout(() => {
    const names = channels.value
      .filter((c) => selectedChannels.value.includes(c.id))
      .map((c) => c.name)
      .join('、')
    message.success(t('publish.publishedDone', { title: selectedNovel.value!.title }))
    history.value.unshift({
      id: 'h' + Date.now(),
      title: selectedNovel.value!.title,
      channel: names,
      status: 'success',
      at: new Date().toLocaleString('zh-CN', { hour12: false }),
    })
    publishing.value = false
  }, 700)
}
</script>

<template>
  <div class="publish-page">
    <header class="page-head">
      <div>
        <h1 class="title">{{ t('publish.title') }}</h1>
        <p class="subtitle">{{ t('publish.subtitle') }}</p>
      </div>
      <span class="head-badge">
        <n-icon :component="CheckmarkDoneOutline" :size="15" />
        {{ enabledChannels.length }} / {{ channels.length }} {{ t('publish.platform') }}
      </span>
    </header>

    <div class="publish-grid">
      <!-- 左侧：作品来源 -->
      <section class="col-source">
        <p class="section-label">{{ t('publish.selectedNovel') }}</p>
        <n-card :bordered="false" class="source-card">
          <div v-if="selectedNovel" class="source-current">
            <div class="source-cover" :style="{ background: selectedNovel.coverColor }">
              {{ selectedNovel.title.slice(0, 1) }}
            </div>
            <div class="source-meta">
              <p class="source-title">{{ selectedNovel.title }}</p>
              <p class="source-sub">
                {{ formatWords(selectedNovel.wordCount) }} · {{ selectedNovel.chapterCount }} 章
              </p>
            </div>
          </div>
          <n-empty v-else :description="t('publish.warnNovel')" size="small" />

          <n-popover trigger="click" placement="bottom-start" :width="320">
            <template #trigger>
              <n-button quaternary size="small" class="source-switch">{{ t('publish.selectedNovelPlaceholder') }}</n-button>
            </template>
            <div class="source-list">
              <button
                v-for="n in novels"
                :key="n.id"
                class="source-item"
                :class="{ active: n.id === selectedNovelId }"
                @click="selectedNovelId = n.id"
              >
                <span class="source-item-cover" :style="{ background: n.coverColor }">{{ n.title.slice(0, 1) }}</span>
                <span class="source-item-name">{{ n.title }}</span>
                <n-icon v-if="n.id === selectedNovelId" :component="CheckmarkDoneOutline" :size="16" class="source-item-check" />
              </button>
            </div>
          </n-popover>
        </n-card>
      </section>

      <!-- 右侧：发布渠道 -->
      <section class="col-channels">
        <p class="section-label">{{ t('publish.channelCardTitle') }}</p>
        <div v-if="channels.length" class="channel-grid">
          <n-card
            v-for="ch in channels"
            :key="ch.id"
            :bordered="false"
            class="channel-card"
            :class="{ 'is-on': ch.enabled }"
          >
            <span class="channel-accent" :style="{ background: PLATFORM_COLOR[ch.id] || 'var(--c-amber)' }" />
            <div class="channel-top">
              <div class="channel-id">
                <span class="channel-logo" :style="{ background: PLATFORM_COLOR[ch.id] || 'var(--c-amber)' }">
                  {{ ch.name.slice(0, 1) }}
                </span>
                <div>
                  <p class="channel-name">{{ ch.name }}</p>
                  <span class="channel-status" :class="ch.status">
                    <i class="dot" />
                    {{ ch.status === 'connected' ? t('publish.connected') : t('publish.disconnected') }}
                  </span>
                </div>
              </div>
              <n-switch :value="ch.enabled" @update:value="(v: boolean) => toggle(ch.id, v)" />
            </div>

            <p class="channel-desc">{{ ch.desc }}</p>

            <div class="channel-foot">
              <span v-if="ch.penName" class="channel-meta">
                <n-icon :component="PersonOutline" :size="14" />{{ ch.penName }}
              </span>
              <span v-else class="channel-meta muted">{{ t('publish.unbound') }}</span>
              <span v-if="ch.account" class="channel-meta">{{ ch.account }}</span>
            </div>
          </n-card>
        </div>
        <n-empty v-else :description="t('publish.emptyChannels')" class="empty" />
      </section>
    </div>

    <!-- 发布历史 -->
    <section class="history">
      <p class="section-label">{{ t('publish.history') }}</p>
      <n-card :bordered="false" class="history-card">
        <div v-if="history.length" class="history-list">
          <div v-for="h in history" :key="h.id" class="history-row">
            <span class="history-dot" :class="h.status" />
            <span class="history-title">{{ h.title }}</span>
            <span class="history-channel">{{ h.channel }}</span>
            <span class="history-status" :class="h.status">
              {{ h.status === 'success' ? t('publish.success') : t('publish.pending') }}
            </span>
            <span class="history-time"><n-icon :component="TimeOutline" :size="13" />{{ h.at }}</span>
          </div>
        </div>
        <n-empty v-else :description="t('publish.emptyChannels')" size="small" />
      </n-card>
    </section>

    <!-- 底部发布栏 -->
    <footer class="publish-bar">
      <div class="publish-bar-info">
        <span class="bar-novel">{{ selectedNovel?.title || t('publish.warnNovel') }}</span>
        <span class="bar-channels">
          →
          <template v-if="selectedChannels.length">
            {{ channels.filter((c) => selectedChannels.includes(c.id)).map((c) => c.name).join('、') }}
          </template>
          <template v-else>{{ t('publish.warnChannel') }}</template>
        </span>
      </div>
      <n-input
        v-model:value="note"
        type="text"
        :placeholder="t('publish.notePlaceholder')"
        class="bar-note"
        :maxlength="60"
      />
      <n-button
        type="primary"
        size="large"
        :loading="publishing"
        :disabled="!selectedNovel || !selectedChannels.length"
        @click="publish"
      >
        <template #icon><n-icon :component="SendOutline" /></template>
        {{ t('publish.publishBtn') }}
      </n-button>
    </footer>
  </div>
</template>
