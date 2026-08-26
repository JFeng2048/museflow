<script setup lang="ts">
import { ref, computed, h } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { NIcon } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import {
  BookOutline,
  CheckmarkDoneOutline,
  CreateOutline,
  FlashOutline,
  PencilOutline,
  SearchOutline,
  BulbOutline,
} from '@vicons/ionicons5'
import { useNovelStore } from '@/stores/novel'

import NovelStatusTag from './components/NovelStatusTag.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const message = useMessage()
const novelStore = useNovelStore()

const novel = computed(() => novelStore.getNovel(route.params.id as string))
const canEdit = computed(() => true)

const content = ref(
  '窗外的雨丝斜斜地打在玻璃上，咖啡馆里只有老式留声机在哼着走调的歌。\n\n他抬起头，"你确定要激活吗？"对面的声音很轻，却像石子落进湖心。\n\n他深吸一口气，按下了"是"。',
)
const selectionToolbar = ref(false)
const inlineInspiration = ref(true)
const currentChapter = ref('第一章 · 觉醒')
const savedAt = ref(t('novel.savedNow'))

const chapters = [
  { title: '第一章 · 觉醒', done: true },
  { title: '第二章 · 旧信', done: false },
  { title: '第三章 · 雨夜', done: false },
  { title: '第四章 · 抉择', done: false },
  { title: '第五章 · 归处', done: false },
]

const wordCount = computed(() => content.value.replace(/\s/g, '').length)

function onSelect() {
  const sel = (window.getSelection()?.toString() || '').trim()
  selectionToolbar.value = sel.length > 0
}
function aiAction(kind: 'continue' | 'rewrite' | 'expand') {
  selectionToolbar.value = false
  const map = { continue: 'novel.aiContinue', rewrite: 'novel.aiRewrite', expand: 'novel.aiExpand' }
  message.success(t(map[kind]))
}
function useInspiration() {
  inlineInspiration.value = false
  message.info(t('novel.inspirationSaved'))
}
function save() {
  savedAt.value = t('novel.savedNow')
  message.success(t('novel.savedToast'))
}
function publish() {
  message.success(t('novel.publishedToast'))
  router.push({ name: 'publish' })
}
</script>

<template>
  <div v-if="novel" class="editor-shell">
    <!-- 顶部条 -->
    <header class="editor-bar">
      <button class="editor-back" @click="router.push('/novels')">← {{ t('novel.backToLibrary') }}</button>
      <div class="editor-meta">
        <span class="editor-title">{{ novel.title }}</span>
        <NovelStatusTag :status="novel.status" />
        <span class="editor-sep">·</span>
        <span>{{ wordCount }} {{ t('novel.wordsUnit') }}</span>
      </div>
      <div class="editor-actions">
        <button class="editor-ghost" @click="save">{{ t('novel.save') }}</button>
        <button class="editor-primary" @click="publish">{{ t('novel.publish') }}</button>
      </div>
    </header>

    <div class="editor-work">
      <!-- 目录 -->
      <aside class="editor-toc">
        <p class="editor-toc-title flex items-center gap-2">
          <n-icon :component="BookOutline" class="text-[15px]" /> {{ t('novel.toc') }}
        </p>
        <button
          v-for="c in chapters"
          :key="c.title"
          class="editor-chap"
          :class="{ on: c.title === currentChapter }"
          @click="currentChapter = c.title"
        >
          <n-icon
            :component="c.done ? CheckmarkDoneOutline : CreateOutline"
            class="text-[15px]"
            :class="c.done ? 'text-emerald-600' : 'text-ink-muted'"
          />
          {{ c.title }}
        </button>
        <button class="editor-chap new">＋ {{ t('novel.newChapter') }}</button>
      </aside>

      <!-- 写作区 -->
      <main class="editor-page mf-scroll" @mouseup="onSelect">
        <h1 class="editor-chapter-title">{{ currentChapter }}</h1>

        <div class="editor-paper">
          <textarea
            v-model="content"
            :readonly="!canEdit"
            class="editor-write"
            :placeholder="t('novel.chapterPlaceholder')"
            @input="savedAt = t('novel.editing')"
          />

          <!-- 选中文字时的浮动 AI 工具条 -->
          <transition name="pop">
            <div v-if="selectionToolbar" class="float-ai">
              <button @click="aiAction('continue')"><n-icon :component="FlashOutline" class="text-[14px]" /> {{ t('novel.aiContinueLabel') }}</button>
              <button @click="aiAction('rewrite')"><n-icon :component="PencilOutline" class="text-[14px]" /> {{ t('novel.aiRewriteLabel') }}</button>
              <button @click="aiAction('expand')"><n-icon :component="SearchOutline" class="text-[14px]" /> {{ t('novel.aiExpandLabel') }}</button>
            </div>
          </transition>

          <!-- 行内灵感卡 -->
          <transition name="pop">
            <div v-if="inlineInspiration" class="editor-insp-card">
              <n-icon :component="BulbOutline" class="editor-i-ico" />
              <div class="editor-i-body">
                <p class="editor-i-title">{{ t('novel.inspFrom') }}：{{ t('novel.inspToday') }}</p>
                <p class="editor-i-desc">{{ t('novel.inspScene') }}：{{ t('novel.inspSceneDesc') }}</p>
              </div>
              <div class="editor-i-actions">
                <button class="editor-use" @click="useInspiration">{{ t('novel.editorUse') }}</button>
                <button class="editor-skip" @click="inlineInspiration = false">{{ t('novel.editorChange') }}</button>
              </div>
            </div>
          </transition>
        </div>
      </main>
    </div>

    <!-- 底部状态 -->
    <footer class="editor-status">
      <span class="editor-dot" :class="{ live: savedAt === t('novel.savedNow') }" />
      {{ t('novel.statusBar', { chapter: currentChapter, words: wordCount, tokens: '1.2k' }) }}
    </footer>
  </div>

  <div v-else class="editor-missing">
    <p>{{ t('novel.missing') }}</p>
    <button @click="router.push('/novels')">{{ t('novel.missingBack') }}</button>
  </div>
</template>

