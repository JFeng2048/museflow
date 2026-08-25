<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import {
  NGrid,
  NGi,
  NCard,
  NStatistic,
  NList,
  NListItem,
  NThing,
  NTag,
  NSpace,
  NButton,
  NProgress,
  NEmpty,
} from 'naive-ui'
import { storeToRefs } from 'pinia'
import { useNovelStore } from '@/stores/novel'
import { useTaskStore } from '@/stores/task'
import { useUserStore } from '@/stores/user'
import NovelCard from '@/components/novel/NovelCard.vue'
import TaskStatusTag from '@/components/task/TaskStatusTag.vue'
import { formatDate, formatWords } from '@/utils/format'
import { taskTypeLabel } from '@/utils/task'
import { loreCount } from '@/mock'

const router = useRouter()
const novelStore = useNovelStore()
const taskStore = useTaskStore()
const userStore = useUserStore()

const { novels } = storeToRefs(novelStore)
const { tasks } = storeToRefs(taskStore)

const nickname = computed(() => userStore.profile?.nickname ?? userStore.profile?.username ?? '创作者')

const totalWords = computed(() => novels.value.reduce((sum, n) => sum + (n.wordCount ?? 0), 0))
const activeTasks = computed(() =>
  tasks.value.filter((t) => t.status === 'running' || t.status === 'pending').length,
)
const recentNovels = computed(() => novels.value.slice(0, 4))
const recentTasks = computed(() => tasks.value.slice(0, 5))

onMounted(() => {
  novelStore.loadNovels()
  taskStore.loadTasks()
})

const goNovel = (id: string) => router.push(`/novels/${id}`)
</script>

<template>
  <div class="dashboard">
    <header class="dashboard__header">
      <div>
        <h1 class="dashboard__title">你好，{{ nickname }} 👋</h1>
        <p class="dashboard__subtitle">今天也来写点故事吧。</p>
      </div>
      <n-button type="primary" @click="router.push('/novels?create=1')">新建项目</n-button>
    </header>

    <n-grid :cols="4" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
      <n-gi span="4 m:1">
        <n-card :bordered="false">
          <n-statistic label="项目总数" :value="novels.length" />
        </n-card>
      </n-gi>
      <n-gi span="4 m:1">
        <n-card :bordered="false">
          <n-statistic label="累计字数" :value="formatWords(totalWords)" />
        </n-card>
      </n-gi>
      <n-gi span="4 m:1">
        <n-card :bordered="false">
          <n-statistic label="进行中任务" :value="activeTasks" />
        </n-card>
      </n-gi>
      <n-gi span="4 m:1">
        <n-card :bordered="false">
          <n-statistic label="设定条目" :value="loreCount" />
        </n-card>
      </n-gi>
    </n-grid>

    <section class="dashboard__section">
      <div class="dashboard__section-head">
        <h2 class="dashboard__section-title">最近项目</h2>
        <n-button text type="primary" @click="router.push('/novels')">查看全部</n-button>
      </div>
      <n-grid v-if="recentNovels.length" :cols="4" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
        <n-gi v-for="novel in recentNovels" :key="novel.id" span="4 s:2 l:1">
          <novel-card :novel="novel" @click="goNovel(novel.id)" />
        </n-gi>
      </n-grid>
      <n-empty v-else description="还没有项目，去创建一个吧" />
    </section>

    <section class="dashboard__section">
      <div class="dashboard__section-head">
        <h2 class="dashboard__section-title">最近任务</h2>
        <n-button text type="primary" @click="router.push('/tasks')">查看全部</n-button>
      </div>
      <n-card :bordered="false">
        <n-list v-if="recentTasks.length">
          <n-list-item v-for="task in recentTasks" :key="task.id">
            <n-thing :title="task.novelTitle" :description="formatDate(task.createdAt)">
              <template #header-extra>
                <n-space align="center" :size="8">
                  <n-tag size="small" :bordered="false" type="info">{{ taskTypeLabel(task.type) }}</n-tag>
                  <TaskStatusTag :status="task.status" />
                </n-space>
              </template>
              <p class="task-prompt">{{ task.prompt }}</p>
              <n-progress
                v-if="task.status === 'running'"
                type="line"
                :percentage="task.progress"
                :height="6"
                :show-indicator="false"
              />
            </n-thing>
          </n-list-item>
        </n-list>
        <n-empty v-else description="暂无任务" />
      </n-card>
    </section>
  </div>
</template>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 24px;
}
.dashboard__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.dashboard__title {
  margin: 0;
  font-size: 22px;
  font-weight: 600;
}
.dashboard__subtitle {
  margin: 4px 0 0;
  color: var(--mf-text-3);
}
.dashboard__section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}
.dashboard__section-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}
.task-prompt {
  margin: 6px 0 0;
  font-size: 13px;
  color: var(--mf-text-3);
}
</style>
