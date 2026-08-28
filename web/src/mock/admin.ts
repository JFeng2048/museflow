import type {
  AdminUser,
  AdminUserCreate,
  AdminMetrics,
  AdminModel,
  AdminAnnouncement,
  AdminLog,
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

export const adminUsers: AdminUser[] = [
  {
    id: 'u-1', name: '林知秋', email: 'demo@museflow.com', role: 'writer',
    status: 'active', activeToday: true, novelCount: 7, totalWords: 165000, credits: 1280,
    createdAt: daysAgo(90), lastActiveAt: hoursAgo(2),
  },
  {
    id: 'u-2', name: '沈砚', email: 'shen@museflow.com', role: 'writer',
    status: 'active', activeToday: false, novelCount: 3, totalWords: 42000, credits: 320,
    createdAt: daysAgo(45), lastActiveAt: daysAgo(2),
  },
  {
    id: 'u-3', name: 'Aki', email: 'aki@museflow.com', role: 'writer',
    status: 'disabled', activeToday: false, novelCount: 1, totalWords: 8000, credits: 0,
    createdAt: daysAgo(20), lastActiveAt: daysAgo(8),
  },
  {
    id: 'u-admin', name: '管理员', email: 'admin@museflow.com', role: 'admin',
    status: 'active', activeToday: true, novelCount: 0, totalWords: 0, credits: 99999,
    createdAt: daysAgo(365), lastActiveAt: hoursAgo(1),
  },
  {
    id: 'u-4', name: '墨小白', email: 'mobai@museflow.com', role: 'writer',
    status: 'active', activeToday: true, novelCount: 12, totalWords: 298000, credits: 540,
    createdAt: daysAgo(150), lastActiveAt: hoursAgo(5),
  },
  {
    id: 'u-5', name: '清风', email: 'qingfeng@museflow.com', role: 'writer',
    status: 'active', activeToday: false, novelCount: 5, totalWords: 88000, credits: 210,
    createdAt: daysAgo(60), lastActiveAt: daysAgo(1),
  },
]

/** 创建用户（mock：返回新用户对象）。 */
export function createAdminUser(input: AdminUserCreate): AdminUser {
  const user: AdminUser = {
    id: 'u-' + Math.random().toString(36).slice(2, 8),
    name: input.name,
    email: input.email,
    role: input.role,
    status: 'active',
    activeToday: false,
    novelCount: 0,
    totalWords: 0,
    credits: 200,
    createdAt: new Date().toISOString(),
    lastActiveAt: daysAgo(999),
  }
  adminUsers.unshift(user)
  return user
}

export const adminModels: AdminModel[] = [
  { id: 'm-chat', name: 'GPT-4o', provider: 'OpenAI', baseUrl: 'https://api.openai.com/v1', apiKey: 'sk-****3a9f', category: '对话', contextWindow: 128000, enabled: true },
  { id: 'm-write', name: 'GLM-4-Plus', provider: '智谱', baseUrl: 'https://open.bigmodel.cn/api/paas/v4', apiKey: 'pk-****7c21', category: '续写', contextWindow: 128000, enabled: true },
  { id: 'm-reason', name: 'Claude-3.5-Sonnet', provider: 'Anthropic', baseUrl: 'https://api.anthropic.com', apiKey: 'sk-ant-****b8e0', category: '推理', contextWindow: 200000, enabled: true },
  { id: 'm-embed', name: 'text-embedding-3-large', provider: 'OpenAI', baseUrl: 'https://api.openai.com/v1', apiKey: 'sk-****3a9f', category: '嵌入', contextWindow: 8191, enabled: true },
  { id: 'm-img', name: 'DALL·E-3', provider: 'OpenAI', baseUrl: 'https://api.openai.com/v1', apiKey: 'sk-****3a9f', category: '图像', contextWindow: 0, enabled: true },
  { id: 'm-write-2', name: 'Qwen-Max', provider: '阿里云', baseUrl: 'https://dashscope.aliyuncs.com/compatible-mode/v1', apiKey: 'sk-****d4aa', category: '续写', contextWindow: 32000, enabled: false },
]

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

export const adminLogs: AdminLog[] = [
  { id: 'l-1', time: minsAgo(2), level: 'error', service: 'user-service', actor: 'system', message: '数据库连接池耗尽，已触发自动扩容' },
  { id: 'l-2', time: minsAgo(9), level: 'warn', service: 'api-gateway', actor: 'system', message: '上游 user-service P99 延迟超过 800ms' },
  { id: 'l-3', time: minsAgo(15), level: 'info', service: 'api-gateway', actor: 'admin@museflow.com', message: '管理员登录成功' },
  { id: 'l-4', time: minsAgo(28), level: 'info', service: 'generation-svc', actor: 'demo@museflow.com', message: '发起续写任务 task-9921，耗时 4.2s' },
  { id: 'l-5', time: minsAgo(41), level: 'warn', service: 'generation-svc', actor: 'system', message: '模型 GLM-4-Plus 触发限流（每分钟 60 次）' },
  { id: 'l-6', time: hoursAgo(1), level: 'info', service: 'crawl4ai-svc', actor: 'mobai@museflow.com', message: '素材采集完成，入库 12 条' },
  { id: 'l-7', time: hoursAgo(2), level: 'error', service: 'generation-svc', actor: 'aki@museflow.com', message: '生成任务失败：上下文超出窗口上限' },
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
