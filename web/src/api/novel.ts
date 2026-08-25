import request from '@/utils/request'
import { novels as mockNovels } from '@/mock'
import type { CreateNovelPayload, Novel } from '@/types'

let store: Novel[] = [...mockNovels]

export function fetchNovels(): Promise<Novel[]> {
  return request.get<Novel[]>('/novels').catch(() => store)
}

export function fetchNovel(id: string): Promise<Novel | undefined> {
  return request.get<Novel | undefined>(`/novels/${id}`).catch(() => store.find((n) => n.id === id))
}

export function createNovel(payload: CreateNovelPayload): Promise<Novel> {
  const novel: Novel = {
    id: `nv-${Date.now()}`,
    title: payload.title,
    description: payload.description,
    genre: payload.genre,
    status: 'draft',
    wordCount: 0,
    chapterCount: 0,
    chapters: [],
    tags: payload.tags,
    updatedAt: new Date().toISOString(),
    createdAt: new Date().toISOString(),
  }
  store = [novel, ...store]
  return request.post<Novel>('/novels', payload).catch(() => novel)
}
