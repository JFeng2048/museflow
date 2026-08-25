import request from '@/utils/request'

export interface PublishChannel {
  id: string
  name: string
  enabled: boolean
  status: 'connected' | 'disconnected'
}

const mockChannels: PublishChannel[] = [
  { id: 'tomato', name: '番茄小说', enabled: true, status: 'connected' },
  { id: 'qidian', name: '起点读书', enabled: false, status: 'disconnected' },
  { id: 'feilu', name: '飞卢小说', enabled: false, status: 'disconnected' },
]

export function fetchChannels(): Promise<PublishChannel[]> {
  return request.get<PublishChannel[]>('/publish/channels').catch(() => mockChannels)
}