export interface Character {
  id: string
  novelId: string
  name: string
  role: '主角' | '配角' | '反派' | '龙套'
  age?: string
  avatarColor: string
  summary: string
  traits: string[]
}

export interface WorldSetting {
  id: string
  novelId: string
  name: string
  category: '地理' | '规则' | '势力' | '历史' | '民俗'
  summary: string
  details: string
}

export type ForeshadowStatus = 'planted' | 'revealing' | 'resolved'

export interface Foreshadow {
  id: string
  novelId: string
  clue: string
  revealChapter: string
  status: ForeshadowStatus
  note: string
}