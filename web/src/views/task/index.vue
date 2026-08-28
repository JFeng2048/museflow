<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import {
  NGrid,
  NGi,
  NCard,
  NTag,
  NButton,
  NEmpty,
  NIcon,
  NProgress,
  NModal,
  NForm,
  NFormItem,
  NSelect,
  NInput,
  NText,
  useMessage,
} from 'naive-ui'
import {
  AddOutline,
  SparklesOutline,
  TimeOutline,
  PlayOutline,
  CheckmarkDoneOutline,
  AlertCircleOutline,
  RefreshOutline,
} from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import { useTaskStore } from '@/stores/task'
import { useNovelStore } from '@/stores/novel'

import TaskStatusTag from './components/TaskStatusTag.vue'
import { TASK_TYPE_OPTIONS, TASK_STATUS_FILTERS, TASK_STATUS_META } from './constants'
import { formatDateTime, formatWords } from '@/utils/format'
import type { TaskStatus, TaskType, GenerationTask } from '@/types'

const { t } = useI18n()
const message = useMessage()
const taskStore = useTaskStore()
const novelStore = useNovelStore()

const { tasks, loading } = storeToRefs(taskStore)
const { allNovels } = storeToRefs(novelStore)

const statusFilter = ref<TaskStatus | 'all'>('all')

// 每个状态的统计，做成顶部概览。
const counts = computed(() => {
  const c: Record<TaskStatus | 'all', number> = { all: tasks.value.length, pending: 0, running: 0, success: 0, failed: 0 }
  for (const tk of tasks.value) c[tk.status]++
  return c
})

const filtered = computed(() =>
  tasks.value.filter((t) => statusFilter.value === 'all' || t.status === statusFilter.value),
)

// 任务类型对应的图标，给每张卡一个签名元素。
const TYPE_ICON: Record<TaskType, any> = {
  outline: SparklesOutline,
  continue: PlayOutline,
  rewrite: RefreshOutline,
  expand: SparklesOutline,
  polish: SparklesOutline,
}

function typeAccent(type: TaskType): string {
  switch (type) {
    case 'outline': return 'var(--c-amber-deep)'
    case 'continue': return '#2f6fb3'
    case 'rewrite': return '#d9488a'
    case 'expand': return '#46b46a'
    case 'polish': return '#9b6bd6'
  }
}

const showCreate = ref(false)
const form = ref<{ novelId: string; type: TaskType; prompt: string }>({
  novelId: '',
  type: 'continue',
  prompt: '',
})
const submitting = ref(false)

const novelOptions = computed(() =>
  novelStore.byUser('u_demo').map((n) => ({ label: n.title, value: n.id })),
)

function taskTypeLabel(type: TaskType): string {
  return t(TASK_TYPE_OPTIONS.find((o) => o.value === type)?.labelKey || '')
}

const typeOptions = computed(() => TASK_TYPE_OPTIONS.map((o) => ({ label: t(o.labelKey), value: o.value })))

async function createTask() {
  if (!form.value.novelId) {
    message.warning(t('task.warnNovel'))
    return
  }
  if (!form.value.prompt.trim()) {
    message.warning(t('task.warnPrompt'))
    return
  }
  const novel = novelStore.byUser('u_demo').find((n) => n.id === form.value.novelId)
  submitting.value = true
  try {
    await taskStore.createTask({
      novelId: form.value.novelId,
      novelTitle: novel?.title ?? t('task.unnamed'),
      type: form.value.type,
      prompt: form.value.prompt.trim(),
    })
    message.success(t('task.created'))
    showCreate.value = false
    form.value = { novelId: '', type: 'continue', prompt: '' }
  } finally {
    submitting.value = false
  }
}

function retry(task: GenerationTask) {
  taskStore.retry(task.id)
  message.success(t('task.created'))
}

onMounted(() => {
  taskStore.loadTasks()
  if (!allNovels.value.length) novelStore.loadNovels()
})
</script>

<template>
  <div class="task-page">
    <header class="page-head">
      <div>
        <h1 class="title">{{ t('task.title') }}</h1>
        <p class="subtitle">{{ t('task.subtitle') }}</p>
      </div>
      <n-button type="primary" @click="showCreate = true">
        <template #icon><n-icon :component="AddOutline" /></template>
        {{ t('task.newTask') }}
      </n-button>
    </header>

    <!-- 概览统计 -->
    <div class="task-stats">
      <button
        v-for="opt in TASK_STATUS_FILTERS"
        :key="opt.value"
        class="stat"
        :class="{ active: statusFilter === opt.value }"
        @click="statusFilter = opt.value as TaskStatus | 'all'"
      >
        <span class="stat-num">{{ counts[opt.value] }}</span>
        <span class="stat-label">{{ t(opt.labelKey) }}</span>
      </button>
    </div>

    <n-grid v-if="filtered.length" :cols="2" :x-gap="18" :y-gap="18" responsive="screen" item-responsive>
      <n-gi v-for="task in filtered" :key="task.id" span="2 m:1">
        <n-card :bordered="false" class="task-card" :class="['is-' + task.status]">
          <span class="task-accent" :style="{ background: typeAccent(task.type) }" />
          <div class="task__head">
            <div class="task__title-row">
              <span class="task__type-icon" :style="{ background: typeAccent(task.type) }">
                <n-icon :component="TYPE_ICON[task.type]" :size="15" />
              </span>
              <span class="task__novel">{{ task.novelTitle }}</span>
              <TaskStatusTag :status="task.status" />
            </div>
            <n-text depth="3" class="task__time">
              <n-icon :component="TimeOutline" :size="12" />{{ formatDateTime(task.createdAt) }}
            </n-text>
          </div>

          <div class="task__type-line">
            <n-tag size="small" :bordered="false" :color="{ color: typeAccent(task.type) + '1f', textColor: typeAccent(task.type) }">
              {{ taskTypeLabel(task.type) }}
            </n-tag>
          </div>

          <p class="task__prompt">{{ task.prompt }}</p>

          <n-progress
            v-if="task.status === 'running'"
            type="line"
            :percentage="task.progress"
            :height="6"
            :show-indicator="false"
            :color="typeAccent(task.type)"
            style="margin: 4px 0 10px"
          />

          <div class="task__meta">
            <span v-if="task.words"><n-icon :component="SparklesOutline" :size="13" />{{ t('task.output') }} {{ formatWords(task.words) }}</span>
            <span v-if="task.finishedAt"><n-icon :component="CheckmarkDoneOutline" :size="13" />{{ t('task.finishedAt') }} {{ formatDateTime(task.finishedAt) }}</span>
            <span v-if="task.status === 'failed'" class="task__failed">
              <n-icon :component="AlertCircleOutline" :size="13" />{{ t(TASK_STATUS_META.failed.labelKey) }}：{{ t('task.failedHint') }}
            </span>
          </div>

          <div v-if="task.status === 'failed'" class="task__actions">
            <n-button size="small" tertiary @click="retry(task)">
              <template #icon><n-icon :component="RefreshOutline" :size="14" /></template>
              {{ t('task.retry') }}
            </n-button>
          </div>
        </n-card>
      </n-gi>
    </n-grid>
    <n-empty v-else :description="loading ? t('common.loading') : t('task.empty')" class="empty" />

    <n-modal
      v-model:show="showCreate"
      preset="card"
      :title="t('task.newTask')"
      style="width: 520px; max-width: 92vw"
      :bordered="false"
    >
      <n-form label-placement="top">
        <n-form-item :label="t('task.relatedNovel')">
          <n-select
            v-model:value="form.novelId"
            :options="novelOptions"
            :placeholder="t('task.pickNovel')"
          />
        </n-form-item>
        <n-form-item :label="t('task.type')">
          <n-select v-model:value="form.type" :options="typeOptions" />
        </n-form-item>
        <n-form-item :label="t('task.prompt')">
          <n-input
            v-model:value="form.prompt"
            type="textarea"
            :placeholder="t('task.promptPlaceholder')"
            :autosize="{ minRows: 3, maxRows: 6 }"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="modal-footer">
          <n-button quaternary @click="showCreate = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="submitting" @click="createTask">{{ t('task.submit') }}</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>
