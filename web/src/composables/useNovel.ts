import { storeToRefs } from 'pinia'
import { useNovelStore } from '@/stores/novel'

export function useNovel() {
  const store = useNovelStore()
  const { allNovels: novels, loading } = storeToRefs(store)
  return {
    novels,
    loading,
    loadNovels: store.loadNovels,
    createNovel: store.createNovel,
    getNovel: store.getNovel,
  }
}
