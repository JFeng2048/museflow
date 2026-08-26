import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { GenerationTask } from '@/types/generation'
import { tasks as mockTasks } from '@/mock'

export const useTaskStore = defineStore('task', () => {
  const tasks = ref<GenerationTask[]>([...mockTasks])
  const loading = ref(false)

  async function loadTasks() {
    await new Promise((r) => setTimeout(r, 150))
    tasks.value = [...mockTasks]
  }

  function createTask(input: {
    novelId: string
    novelTitle: string
    type: GenerationTask['type']
    prompt: string
  }): GenerationTask {
    const task: GenerationTask = {
      id: 'tk_' + Math.random().toString(36).slice(2, 8),
      ...input,
      status: 'running',
      progress: 12,
      words: 0,
      createdAt: new Date().toISOString(),
    }
    tasks.value.unshift(task)
    return task
  }

  function retry(id: string) {
    const task = tasks.value.find((t) => t.id === id)
    if (task) {
      task.status = 'running'
      task.progress = 12
      task.finishedAt = undefined
    }
  }

  return { tasks, loading, loadTasks, createTask, retry }
})
