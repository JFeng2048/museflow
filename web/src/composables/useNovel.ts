import { storeToRefs } from 'pinia'
import { useNovelStore } from '@/stores/novel'

export function useNovel() {
  const store = useNovelStore()
  const { novels, loading } = storeToRefs(store)
  return {
    novels,
    loading,
    loadNovels: store.loadNovels,
    createNovel: store.createNovel,
    findById: store.findById,
  }
}
