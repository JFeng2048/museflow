import request from '@/utils/request'
import type { PublishChannel } from './type'

const mockChannels: PublishChannel[] = [
  {
    id: 'tomato',
    name: '番茄小说',
    enabled: true,
    status: 'connected',
    account: 'writer@museflow.app',
    penName: '知秋',
    requiresLogin: true,
    desc: '字节系免费阅读平台，适合快节奏爽文。',
  },
  {
    id: 'qidian',
    name: '起点读书',
    enabled: false,
    status: 'disconnected',
    requiresLogin: true,
    desc: '阅文旗下付费头部平台，适合长篇精品。',
  },
  {
    id: 'feilu',
    name: '飞卢小说',
    enabled: false,
    status: 'disconnected',
    requiresLogin: true,
    desc: '同人 / 快节奏题材友好，分成模式灵活。',
  },
  {
    id: 'jjwxc',
    name: '晋江文学城',
    enabled: false,
    status: 'disconnected',
    requiresLogin: true,
    desc: '女性向头部平台，适合言情 / 耽美。',
  },
]

export function fetchChannels(): Promise<PublishChannel[]> {
  return request.get<PublishChannel[]>('/publish/channels').catch(() => mockChannels)
}

export type { PublishChannel }
