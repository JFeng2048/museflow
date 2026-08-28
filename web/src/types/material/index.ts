export type MaterialType = 'character' | 'plot' | 'world' | 'quote' | 'image'

export interface Material {
  id: string
  title: string
  type: MaterialType
  source: string
  content: string
  tags: string[]
  imported: boolean
  createdAt: string
}
