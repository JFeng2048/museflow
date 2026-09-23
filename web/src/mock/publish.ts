import type { PublishChannel } from '@/types/publish'

/**
 * 发布渠道演示数据。
 *
 * 后端尚无 publish-service，这些平台连接都是本地假数据；
 * 页面必须显式标注「演示数据」，不要让用户误以为已经接好了发布通道。
 */
export const channels: PublishChannel[] = [
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
