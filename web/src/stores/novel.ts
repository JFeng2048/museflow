import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Novel, NovelStatus, CreateNovelPayload, NovelStats } from '@/types/novel'
import { novels as mockNovels } from '@/mock'

const sampleNovels: Novel[] = mockNovels

export const useNovelStore = defineStore('novel', () => {
  const novels = ref<Novel[]>([])
  const loading = ref(false)
  const loaded = ref(false)

  async function loadNovels() {
    if (loaded.value) return
    loading.value = true
    await new Promise((r) => setTimeout(r, 200))
    novels.value = sampleNovels
    loaded.value = true
    loading.value = false
  }

  // 个人空间：作品按用户隔离（team/workspace 概念已移除）。
  const allNovels = novels

  const byUser = (userId: string) => novels.value.filter((n) => n.userId === userId)
  const getNovel = (id: string) => novels.value.find((n) => n.id === id)

  async function createNovel(payload: CreateNovelPayload, userId: string) {
    await new Promise((r) => setTimeout(r, 250))
    const colors = ['#d4a05a', '#1a2332', '#b9853f', '#5b7b9b', '#7faa5a']
    const novel: Novel = {
      id: 'n_' + Math.random().toString(36).slice(2, 8),
      userId,
      title: payload.title,
      premise: payload.description || '',
      genre: payload.genre || '',
      tags: payload.tags || [],
      coverColor: payload.coverColor || colors[Math.floor(Math.random() * colors.length)],
      status: 'draft' as NovelStatus,
      wordCount: 0,
      chapterCount: 0,
      wordGoal: payload.wordGoal || 300000,
      updatedAt: new Date().toISOString(),
      createdAt: new Date().toISOString(),
    }
    novels.value.unshift(novel)
    return novel
  }

  function updateNovel(id: string, patch: Partial<Novel>) {
    const n = novels.value.find((x) => x.id === id)
    if (n) Object.assign(n, patch, { updatedAt: new Date().toISOString() })
  }

  function deleteNovel(id: string) {
    novels.value = novels.value.filter((n) => n.id !== id)
  }

  function userStats(userId: string): NovelStats {
    const list = byUser(userId)
    const totalWords = list.reduce((s, n) => s + n.wordCount, 0)
    const statusOngoing = list.filter((n) => n.status === 'ongoing' || n.status === 'serializing').length
    const statusCompleted = list.filter((n) => n.status === 'completed').length
    const statusDraft = list.filter((n) => n.status === 'draft').length
    // 近 7 日字数：没有真实的按日统计，这里用作品字数的平滑分布做演示。
    const last7Words = Array.from({ length: 7 }, (_, i) => {
      const base = Math.round(totalWords / 14)
      return Math.max(0, Math.round(base * (0.4 + 0.6 * Math.sin((i + 1) / 2))))
    })
    return {
      totalNovels: list.length,
      totalWords,
      statusOngoing,
      statusCompleted,
      statusDraft,
      last7Words,
      completionRate: list.length ? Math.round((statusCompleted / list.length) * 100) : 0,
    }
  }

  return {
    novels,
    allNovels,
    loading,
    loaded,
    loadNovels,
    byUser,
    getNovel,
    createNovel,
    updateNovel,
    deleteNovel,
    userStats,
  }
})
