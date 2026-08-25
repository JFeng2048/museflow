<script setup lang="ts">
import { onMounted, ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NButton, NIcon, NTag, NSpace, NInput, NEmpty, NCard, useMessage } from 'naive-ui'
import { ArrowBackOutline, SparklesOutline, SaveOutline } from '@vicons/ionicons5'
import { storeToRefs } from 'pinia'
import { useNovelStore } from '@/stores/novel'
import { useTaskStore } from '@/stores/task'
import ChapterTree from '@/components/novel/ChapterTree.vue'
import NovelStatusTag from '@/components/novel/NovelStatusTag.vue'
import { formatWords, formatDateTime } from '@/utils/format'
import type { Chapter } from '@/types'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const novelStore = useNovelStore()
const taskStore = useTaskStore()
const { novels } = storeToRefs(novelStore)

const novel = computed(() => novelStore.findById(route.params.id as string))
const activeChapter = ref<Chapter | null>(null)

// 编辑器内容按章节 id 暂存在本地（原型，不落库）
const drafts = ref<Record<string, string>>({})

const currentContent = computed({
  get: () => (activeChapter.value ? drafts.value[activeChapter.value.id] ?? '' : ''),
  set: (v: string) => {
    if (activeChapter.value) drafts.value[activeChapter.value.id] = v
  },
})

const currentWords = computed(() => (currentContent.value ? currentContent.value.length : 0))

const chapterStatusMap: Record<Chapter['status'], string> = {
  draft: '草稿',
  written: '已写',
  polished: '已润色',
}
function chapterStatusLabel(s: Chapter['status']) {
  return chapterStatusMap[s]
}

function selectChapter(ch: Chapter) {
  activeChapter.value = ch
}

function saveDraft() {
  message.success('草稿已保存（本地原型）')
}

async function aiContinue() {
  if (!novel.value || !activeChapter.value) return
  await taskStore.createTask({
    novelId: novel.value.id,
    novelTitle: novel.value.title,
    type: 'continue',
    prompt: `续写《${novel.value.title}》的「${activeChapter.value.title}」，保持当前文风与节奏。`,
  })
  message.success('已提交续写任务，可在「生成任务」查看进度')
}

async function ensureLoaded() {
  if (!novels.value.length) await novelStore.loadNovels()
  const first = novel.value?.chapters?.[0]
  if (first) selectChapter(first)
}

onMounted(ensureLoaded)
watch(() => route.params.id, ensureLoaded)
</script>

<template>
  <div v-if="novel" class="detail">
    <header class="detail__top">
      <div class="detail__title-row">
        <n-button quaternary circle @click="router.back()">
          <n-icon :component="ArrowBackOutline" />
        </n-button>
        <h1 class="detail__title">{{ novel.title }}</h1>
        <NovelStatusTag :status="novel.status" />
      </div>
      <p class="detail__desc">{{ novel.description }}</p>
      <div class="detail__meta">
        <span>{{ formatWords(novel.wordCount) }}</span>
        <span>· {{ novel.chapterCount }} 章</span>
        <span>· 更新于 {{ formatDateTime(novel.updatedAt) }}</span>
        <n-space :size="6" style="margin-left: 8px">
          <n-tag v-for="t in novel.tags" :key="t" size="tiny" :bordered="false">{{ t }}</n-tag>
        </n-space>
      </div>
    </header>

    <div class="detail__body">
      <n-card :bordered="false" class="detail__tree" title="章节">
        <ChapterTree
          v-if="novel.chapters.length"
          :chapters="novel.chapters"
          :active-id="activeChapter?.id"
          @select="selectChapter"
        />
        <n-empty v-else description="还没有章节" size="small" />
      </n-card>

      <n-card :bordered="false" class="detail__editor">
        <template v-if="activeChapter">
          <div class="editor__head">
            <div>
              <h2 class="editor__title">{{ activeChapter.title }}</h2>
              <n-space :size="8" align="center">
                <n-tag size="small" :bordered="false">{{ chapterStatusLabel(activeChapter.status) }}</n-tag>
                <span class="editor__words"
                  >{{ formatWords(activeChapter.words) }}（正文 {{ currentWords }} 字）</span
                >
              </n-space>
            </div>
            <n-space>
              <n-button secondary @click="saveDraft">
                <template #icon><n-icon :component="SaveOutline" /></template>
                保存草稿
              </n-button>
              <n-button type="primary" @click="aiContinue">
                <template #icon><n-icon :component="SparklesOutline" /></template>
                AI 续写
              </n-button>
            </n-space>
          </div>
          <n-input
            v-model:value="currentContent"
            type="textarea"
            placeholder="在这里书写这一章的内容，灵感来时尽情落笔…"
            class="editor__area"
            :autosize="{ minRows: 18, maxRows: 32 }"
          />
        </template>
        <n-empty v-else description="从左侧选择一章开始写作" />
      </n-card>
    </div>
  </div>
  <n-empty v-else description="找不到该项目，它可能已被删除" style="margin-top: 80px" />
</template>

<style scoped>
.detail {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.detail__top {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.detail__title-row {
  display: flex;
  align-items: center;
  gap: 10px;
}
.detail__title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--mf-text);
}
.detail__desc {
  color: var(--mf-text-3);
  margin: 6px 0 0;
}
.detail__meta {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--mf-text-3);
  font-size: 13px;
  flex-wrap: wrap;
}
.detail__body {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}
.detail__tree {
  width: 300px;
  flex: none;
}
.detail__editor {
  flex: 1;
  min-width: 0;
}
.editor__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}
.editor__title {
  margin: 0 0 6px;
  font-size: 16px;
  font-weight: 600;
  color: var(--mf-text);
}
.editor__words {
  color: var(--mf-text-3);
  font-size: 12px;
}
.editor__area {
  font-family: inherit;
}
@media (max-width: 900px) {
  .detail__body {
    flex-direction: column;
  }
  .detail__tree {
    width: 100%;
  }
}
</style>
