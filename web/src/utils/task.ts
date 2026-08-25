import type { TaskStatus, TaskType } from '@/types'

export const TASK_TYPE_OPTIONS: { label: string; value: TaskType }[] = [
  { label: '大纲生成', value: 'outline' },
  { label: '续写', value: 'continue' },
  { label: '改写', value: 'rewrite' },
  { label: '扩写', value: 'expand' },
  { label: '润色', value: 'polish' },
]

export function taskTypeLabel(type: TaskType): string {
  return TASK_TYPE_OPTIONS.find((o) => o.value === type)?.label ?? type
}

export function taskStatusMeta(
  status: TaskStatus,
): { label: string; type: 'default' | 'info' | 'success' | 'warning' | 'error' } {
  switch (status) {
    case 'pending':
      return { label: '排队中', type: 'default' }
    case 'running':
      return { label: '进行中', type: 'info' }
    case 'success':
      return { label: '已完成', type: 'success' }
    case 'failed':
      return { label: '失败', type: 'error' }
  }
}
