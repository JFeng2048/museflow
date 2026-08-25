import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchTasks, createTask as createTaskApi } from '@/api/generation'
import type { GenerationTask } from '@/types'

export const useTaskStore = defineStore('task', () => {
  const tasks = ref<GenerationTask[]>([])
  const loading = ref(false)

  async function loadTasks() {
    loading.value = true
    try {
      tasks.value = await fetchTasks()
    } finally {
      loading.value = false
    }
  }

  async function createTask(payload: {
    novelId: string
    novelTitle: string
    type: GenerationTask['type']
    prompt: string
  }) {
    const task = await createTaskApi(payload)
    tasks.value = [task, ...tasks.value]
    return task
  }

  return { tasks, loading, loadTasks, createTask }
})
