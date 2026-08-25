export type NovelStatus = 'draft' | 'serializing' | 'completed' | 'paused'

export interface Chapter {
  id: string
  title: string
  words: number
  status: 'draft' | 'written' | 'polished'
  updatedAt: string
  children?: Chapter[]
}

export interface Novel {
  id: string
  title: string
  cover?: string
  description: string
  genre: string
  status: NovelStatus
  wordCount: number
  chapterCount: number
  chapters: Chapter[]
  tags: string[]
  updatedAt: string
  createdAt: string
}

export interface CreateNovelPayload {
  title: string
  description: string
  genre: string
  tags: string[]
}
