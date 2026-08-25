import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchNovels, createNovel as createNovelApi } from '@/api/novel'
import type { CreateNovelPayload, Novel } from '@/types'

export const useNovelStore = defineStore('novel', () => {
  const novels = ref<Novel[]>([])
  const loading = ref(false)

  async function loadNovels() {
    loading.value = true
    try {
      novels.value = await fetchNovels()
    } finally {
      loading.value = false
    }
  }

  async function createNovel(payload: CreateNovelPayload) {
    const novel = await createNovelApi(payload)
    novels.value = [novel, ...novels.value]
    return novel
  }

  function findById(id: string) {
    return novels.value.find((n) => n.id === id)
  }

  return { novels, loading, loadNovels, createNovel, findById }
})
