import type { TaskStatus, TaskType } from '@/types'

export const TASK_TYPE_OPTIONS: { labelKey: string; value: TaskType }[] = [
  { labelKey: 'task.typeOutline', value: 'outline' },
  { labelKey: 'task.typeContinue', value: 'continue' },
  { labelKey: 'task.typeRewrite', value: 'rewrite' },
  { labelKey: 'task.typeExpand', value: 'expand' },
  { labelKey: 'task.typePolish', value: 'polish' },
]

export const TASK_STATUS_FILTERS: { labelKey: string; value: TaskStatus | 'all' }[] = [
  { labelKey: 'task.filterAll', value: 'all' },
  { labelKey: 'task.statusPending', value: 'pending' },
  { labelKey: 'task.statusRunning', value: 'running' },
  { labelKey: 'task.statusSuccess', value: 'success' },
  { labelKey: 'task.statusFailed', value: 'failed' },
]

export const TASK_STATUS_META: Record<
  TaskStatus,
  { labelKey: string; type: 'default' | 'info' | 'success' | 'warning' | 'error' }
> = {
  pending: { labelKey: 'task.statusPending', type: 'default' },
  running: { labelKey: 'task.statusRunning', type: 'info' },
  success: { labelKey: 'task.statusSuccess', type: 'success' },
  failed: { labelKey: 'task.statusFailed', type: 'error' },
}
