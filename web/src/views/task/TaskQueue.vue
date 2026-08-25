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
  NSpace,
  NText,
  useMessage,
} from 'naive-ui'
import { AddOutline } from '@vicons/ionicons5'
import { storeToRefs } from 'pinia'
import { useTaskStore } from '@/stores/task'
import { useNovelStore } from '@/stores/novel'
import TaskStatusTag from '@/components/task/TaskStatusTag.vue'
import { TASK_TYPE_OPTIONS, taskTypeLabel, taskStatusMeta } from '@/utils/task'
import { formatDateTime, formatWords } from '@/utils/format'
import type { TaskStatus, TaskType } from '@/types'

const message = useMessage()
const taskStore = useTaskStore()
const novelStore = useNovelStore()
const { tasks, loading } = storeToRefs(taskStore)
const { novels } = storeToRefs(novelStore)

const statusFilter = ref<TaskStatus | 'all'>('all')
const statusOptions = [
  { label: '全部', value: 'all' },
  { label: '排队中', value: 'pending' },
  { label: '进行中', value: 'running' },
  { label: '已完成', value: 'success' },
  { label: '失败', value: 'failed' },
]

const filtered = computed(() =>
  tasks.value.filter((t) => statusFilter.value === 'all' || t.status === statusFilter.value),
)

const showCreate = ref(false)
const form = ref<{ novelId: string; type: TaskType; prompt: string }>({
  novelId: '',
  type: 'continue',
  prompt: '',
})
const submitting = ref(false)

const novelOptions = computed(() => novels.value.map((n) => ({ label: n.title, value: n.id })))

async function createTask() {
  if (!form.value.novelId) {
    message.warning('请选择关联项目')
    return
  }
  if (!form.value.prompt.trim()) {
    message.warning('请填写生成指令')
    return
  }
  const novel = novels.value.find((n) => n.id === form.value.novelId)
  submitting.value = true
  try {
    await taskStore.createTask({
      novelId: form.value.novelId,
      novelTitle: novel?.title ?? '未命名项目',
      type: form.value.type,
      prompt: form.value.prompt.trim(),
    })
    message.success('任务已提交')
    showCreate.value = false
    form.value = { novelId: '', type: 'continue', prompt: '' }
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  taskStore.loadTasks()
  if (!novels.value.length) novelStore.loadNovels()
})
</script>

<template>
  <div class="page">
    <header class="page__head">
      <div>
        <h1 class="page__title">生成任务</h1>
        <p class="page__sub">AI 正在替你把灵感变成文字，进度一目了然。</p>
      </div>
      <n-button type="primary" @click="showCreate = true">
        <template #icon><n-icon :component="AddOutline" /></template>
        新建生成任务
      </n-button>
    </header>

    <n-space class="filters">
      <n-tag
        v-for="opt in statusOptions"
        :key="opt.value"
        :type="statusFilter === opt.value ? 'primary' : 'default'"
        :bordered="false"
        round
        class="filter-tag"
        @click="statusFilter = opt.value as TaskStatus | 'all'"
      >
        {{ opt.label }}
      </n-tag>
    </n-space>

    <n-grid v-if="filtered.length" :cols="1" :y-gap="12">
      <n-gi v-for="task in filtered" :key="task.id">
        <n-card :bordered="false" class="task-card">
          <div class="task__head">
            <div class="task__title-row">
              <span class="task__novel">{{ task.novelTitle }}</span>
              <n-tag size="small" :bordered="false" type="info">{{ taskTypeLabel(task.type) }}</n-tag>
              <TaskStatusTag :status="task.status" />
            </div>
            <n-text depth="3" style="font-size: 12px">{{ formatDateTime(task.createdAt) }}</n-text>
          </div>
          <p class="task__prompt">{{ task.prompt }}</p>
          <n-progress
            v-if="task.status === 'running'"
            type="line"
            :percentage="task.progress"
            :height="6"
            :show-indicator="false"
            style="margin: 6px 0"
          />
          <div class="task__meta">
            <span v-if="task.words">产出 {{ formatWords(task.words) }}</span>
            <span v-if="task.finishedAt">完成于 {{ formatDateTime(task.finishedAt) }}</span>
            <span v-if="task.status === 'failed'" class="task__failed">
              {{ taskStatusMeta('failed').label }}：生成服务超时，请重试
            </span>
          </div>
        </n-card>
      </n-gi>
    </n-grid>
    <n-empty v-else :description="loading ? '加载中…' : '暂无任务'" />

    <n-modal
      v-model:show="showCreate"
      preset="card"
      title="新建生成任务"
      style="width: 520px; max-width: 92vw"
      :bordered="false"
    >
      <n-form label-placement="top">
        <n-form-item label="关联项目">
          <n-select
            v-model:value="form.novelId"
            :options="novelOptions"
            placeholder="选择要操作的小说"
          />
        </n-form-item>
        <n-form-item label="任务类型">
          <n-select v-model:value="form.type" :options="TASK_TYPE_OPTIONS" />
        </n-form-item>
        <n-form-item label="生成指令">
          <n-input
            v-model:value="form.prompt"
            type="textarea"
            placeholder="描述你希望 AI 完成的工作，越具体越好"
            :autosize="{ minRows: 3, maxRows: 6 }"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="modal-footer">
          <n-button quaternary @click="showCreate = false">取消</n-button>
          <n-button type="primary" :loading="submitting" @click="createTask">提交任务</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.page__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
}
.page__title {
  margin: 0;
  font-size: 22px;
  font-weight: 600;
}
.page__sub {
  margin: 4px 0 0;
  color: var(--mf-text-3);
}
.filters {
  flex-wrap: wrap;
}
.filter-tag {
  cursor: pointer;
}
.task-card {
  width: 100%;
}
.task__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}
.task__title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.task__novel {
  font-weight: 600;
  font-size: 15px;
  color: var(--mf-text);
}
.task__prompt {
  color: var(--mf-text-2);
  font-size: 13px;
  line-height: 1.7;
  margin: 0 0 8px;
}
.task__meta {
  display: flex;
  align-items: center;
  gap: 14px;
  color: var(--mf-text-3);
  font-size: 12px;
}
.task__failed {
  color: #d64545;
}
.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
