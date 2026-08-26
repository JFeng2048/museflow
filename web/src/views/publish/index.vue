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
  // 模拟提交
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

<style scoped>
.publish-page {
  max-width: 1180px;
  margin: 0 auto;
  padding: 8px 4px 96px;
}
.page-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 22px;
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
.head-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--c-ink-muted);
  background: var(--c-warm-2);
  border: 1px solid var(--c-line);
  padding: 6px 12px;
  border-radius: 999px;
  white-space: nowrap;
}
.section-label {
  font-size: 12px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--c-ink-muted);
  margin: 0 0 12px;
}
.publish-grid {
  display: grid;
  grid-template-columns: 300px 1fr;
  gap: 22px;
  align-items: start;
}

/* 作品来源 */
.source-card {
  border-radius: 16px;
  border: 1px solid var(--c-line);
}
.source-current {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 12px;
}
.source-cover {
  width: 52px;
  height: 52px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  color: #fff;
  font-family: var(--font-serif);
  font-size: 24px;
  font-weight: 700;
  flex: none;
}
.source-meta {
  min-width: 0;
}
.source-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--c-ink);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.source-sub {
  margin: 3px 0 0;
  font-size: 12px;
  color: var(--c-ink-muted);
}
.source-switch {
  width: 100%;
  justify-content: center;
  border: 1px dashed var(--c-line);
  border-radius: 10px;
  color: var(--c-ink-muted);
}
.source-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 260px;
  overflow-y: auto;
}
.source-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px;
  border: none;
  background: transparent;
  border-radius: 10px;
  cursor: pointer;
  text-align: left;
  color: var(--c-ink);
}
.source-item:hover {
  background: var(--c-warm-2);
}
.source-item.active {
  background: var(--c-amber-soft);
}
.source-item-cover {
  width: 30px;
  height: 30px;
  border-radius: 8px;
  display: grid;
  place-items: center;
  color: #fff;
  font-family: var(--font-serif);
  font-weight: 700;
  font-size: 14px;
  flex: none;
}
.source-item-name {
  flex: 1;
  font-size: 14px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.source-item-check {
  color: var(--c-amber-deep);
}

/* 渠道卡 */
.channel-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 16px;
}
.channel-card {
  position: relative;
  border-radius: 16px;
  border: 1px solid var(--c-line);
  overflow: hidden;
  transition: border-color 0.18s ease, box-shadow 0.18s ease, transform 0.18s ease;
}
.channel-card:hover {
  transform: translateY(-3px);
  box-shadow: 0 12px 30px rgba(26, 35, 50, 0.1);
}
.channel-card.is-on {
  border-color: rgba(185, 133, 63, 0.4);
}
.channel-accent {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 4px;
}
.channel-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 12px;
}
.channel-id {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.channel-logo {
  width: 40px;
  height: 40px;
  border-radius: 11px;
  display: grid;
  place-items: center;
  color: #fff;
  font-weight: 700;
  font-size: 18px;
  font-family: var(--font-serif);
  flex: none;
}
.channel-name {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--c-ink);
}
.channel-status {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: var(--c-ink-muted);
}
.channel-status .dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--c-line);
}
.channel-status.connected {
  color: var(--c-amber-deep);
}
.channel-status.connected .dot {
  background: #46b46a;
}
.channel-status.disconnected .dot {
  background: #c4c4c4;
}
.channel-desc {
  margin: 0 0 12px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--c-ink-soft);
  min-height: 40px;
}
.channel-foot {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 14px;
  padding-top: 12px;
  border-top: 1px solid var(--c-line-soft);
}
.channel-meta {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--c-ink-muted);
}
.channel-meta.muted {
  color: var(--c-ink-muted);
  opacity: 0.7;
}

/* 历史 */
.history {
  margin-top: 26px;
}
.history-card {
  border-radius: 16px;
  border: 1px solid var(--c-line);
}
.history-list {
  display: flex;
  flex-direction: column;
}
.history-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 11px 4px;
  border-bottom: 1px solid var(--c-line-soft);
}
.history-row:last-child {
  border-bottom: none;
}
.history-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex: none;
}
.history-dot.success {
  background: #46b46a;
}
.history-dot.pending {
  background: #e0a73a;
}
.history-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--c-ink);
  min-width: 120px;
}
.history-channel {
  font-size: 13px;
  color: var(--c-ink-soft);
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.history-status {
  font-size: 12px;
  padding: 2px 9px;
  border-radius: 999px;
  flex: none;
}
.history-status.success {
  background: rgba(70, 180, 106, 0.12);
  color: #2f8c4f;
}
.history-status.pending {
  background: rgba(224, 167, 58, 0.14);
  color: #b9842b;
}
.history-time {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--c-ink-muted);
  flex: none;
}

/* 底部发布栏 */
.publish-bar {
  position: sticky;
  bottom: 0;
  margin-top: 24px;
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 14px 18px;
  background: var(--c-paper);
  border: 1px solid var(--c-line);
  border-radius: 16px;
  box-shadow: 0 -4px 24px rgba(26, 35, 50, 0.06);
  z-index: 10;
}
.publish-bar-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.bar-novel {
  font-size: 14px;
  font-weight: 600;
  color: var(--c-ink);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 200px;
}
.bar-channels {
  font-size: 12px;
  color: var(--c-ink-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 320px;
}
.bar-note {
  flex: 1;
  max-width: 400px;
}
.empty {
  padding: 60px 0;
}

@media (max-width: 900px) {
  .publish-grid {
    grid-template-columns: 1fr;
  }
  .publish-bar {
    flex-wrap: wrap;
  }
  .bar-note {
    order: 3;
    max-width: 100%;
    flex: 1 1 100%;
  }
}
</style>
