export type TaskStatus = 'pending' | 'running' | 'success' | 'failed'
export type TaskType = 'outline' | 'continue' | 'rewrite' | 'expand' | 'polish'

export interface GenerationTask {
  id: string
  novelId: string
  novelTitle: string
  type: TaskType
  prompt: string
  status: TaskStatus
  progress: number
  words: number
  createdAt: string
  finishedAt?: string
}
