import { storeToRefs } from 'pinia'
import { useTaskStore } from '@/stores/task'

export function useTask() {
  const store = useTaskStore()
  const { tasks, loading } = storeToRefs(store)
  return {
    tasks,
    loading,
    loadTasks: store.loadTasks,
    createTask: store.createTask,
  }
}
