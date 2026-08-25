import request from '@/utils/request'
import { tasks as mockTasks } from '@/mock'
import type { GenerationTask } from '@/types'

let store: GenerationTask[] = [...mockTasks]

export function fetchTasks(): Promise<GenerationTask[]> {
  return request.get<GenerationTask[]>('/tasks').catch(() => store)
}

export function createTask(payload: {
  novelId: string
  novelTitle: string
  type: GenerationTask['type']
  prompt: string
}): Promise<GenerationTask> {
  const task: GenerationTask = {
    id: `tk-${Date.now()}`,
    ...payload,
    status: 'pending',
    progress: 0,
    words: 0,
    createdAt: new Date().toISOString(),
  }
  store = [task, ...store]
  return request.post<GenerationTask>('/tasks', task).catch(() => task)
}
