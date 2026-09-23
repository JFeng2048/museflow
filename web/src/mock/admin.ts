import type {
  AdminMetrics,
  AdminAnnouncement,
  AdminService,
} from '@/types/admin'

function daysAgo(n: number): string {
  return new Date(Date.now() - n * 86400000).toISOString()
}
function hoursAgo(n: number): string {
  return new Date(Date.now() - n * 3600000).toISOString()
}
function minsAgo(n: number): string {
  return new Date(Date.now() - n * 60000).toISOString()
}

/**
 * 注意：用户管理（/admin/users）与审计日志（/admin/audit-logs）已对接真实接口，
 * 这里不再提供 adminUsers / adminLogs mock 数据。
 * 以下 mock 仅服务于后端尚未实现的页面（概览指标 / 公告 / 服务监控）。
 */

export const adminAnnouncements: AdminAnnouncement[] = [
  {
    id: 'a-1', title: 'MuseFlow 2.0 创作引擎升级公告',
    content: '我们已对续写与推理模型进行全面升级，生成质量与速度显著提升。\n旧版草稿不受影响，欢迎体验全新世界观一致性校验。',
    level: 'important', pinned: true, publishedAt: hoursAgo(6), status: 'published',
  },
  {
    id: 'a-2', title: '例行维护通知（本周日 02:00-04:00）',
    content: '系统将于本周日进行数据库扩容维护，期间服务短暂只读，敬请提前保存作品。',
    level: 'maintenance', pinned: false, publishedAt: daysAgo(1), status: 'published',
  },
  {
    id: 'a-3', title: '积分体系优化说明',
    content: '新注册用户赠送积分已从 100 调整为 200，详见系统设置。',
    level: 'normal', pinned: false, publishedAt: daysAgo(3), status: 'draft',
  },
]

export const adminServices: AdminService[] = [
  { id: 's-gateway', name: 'API Gateway', kind: 'Gin / :5001', status: 'healthy', instances: 2, latency: 42, cpu: 23, memory: 38, checkedAt: minsAgo(1), endpoint: 'http://gateway:5001' },
  { id: 's-user', name: 'User Service', kind: 'gRPC / :5002', status: 'healthy', instances: 2, latency: 28, cpu: 18, memory: 31, checkedAt: minsAgo(1), endpoint: 'grpc://user-svc:5002' },
  { id: 's-gen', name: 'Generation Service', kind: 'Go / :5003', status: 'degraded', instances: 3, latency: 820, cpu: 71, memory: 64, checkedAt: minsAgo(1), endpoint: 'http://gen-svc:5003' },
  { id: 's-crawl', name: 'Crawl4AI Service', kind: 'Python / :8000', status: 'healthy', instances: 1, latency: 410, cpu: 44, memory: 52, checkedAt: minsAgo(2), endpoint: 'http://crawl-svc:8000' },
  { id: 's-redis', name: 'Redis', kind: 'Cache / :6379', status: 'healthy', instances: 1, latency: 3, cpu: 9, memory: 27, checkedAt: minsAgo(1), endpoint: 'redis://cache:6379' },
  { id: 's-pg', name: 'PostgreSQL', kind: 'DB / :5432', status: 'down', instances: 1, latency: 0, cpu: 0, memory: 0, checkedAt: minsAgo(3), endpoint: 'postgres://db:5432' },
]

export const adminMetrics: AdminMetrics = {
  totalUsers: 1284,
  newToday: 19,
  new7: [12, 15, 9, 21, 17, 14, 19],
  totalNovels: 5632,
  totalWords: 184000000,
  genToday: 372,
  servicesOnline: 5,
  servicesTotal: 6,
}
