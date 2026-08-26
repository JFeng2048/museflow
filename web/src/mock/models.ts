import type { ModelProvider, AIModel } from '@/types/model'

export const systemProviders: ModelProvider[] = [
  {
    id: 'prov-museflow',
    system: true,
    name: 'MuseFlow 官方',
    protocol: 'openai',
    baseUrl: 'https://api.museflow.ai/v1',
    apiKey: '••••••••••••',
    createdAt: '2026-01-01T00:00:00Z',
  },
]

export const systemModels: AIModel[] = [
  {
    id: 'm-muse-pro',
    providerId: 'prov-museflow',
    name: 'MusePro 创作主力',
    apiModel: 'muse-pro',
    creditCost: 5,
    contextK: 128,
    system: true,
    enabled: true,
    description: '长文续写与润色，文风最贴近小说创作。',
  },
  {
    id: 'm-muse-lite',
    providerId: 'prov-museflow',
    name: 'MuseLite 轻量',
    apiModel: 'muse-lite',
    creditCost: 1,
    contextK: 32,
    system: true,
    enabled: true,
    description: '灵感、摘要、配角对白等轻量任务。',
  },
  {
    id: 'm-muse-vision',
    providerId: 'prov-museflow',
    name: 'MuseVision 视觉',
    apiModel: 'muse-vision',
    creditCost: 8,
    contextK: 64,
    system: true,
    enabled: false,
    description: '封面与分镜草图生成。',
  },
]

export const customProviders: ModelProvider[] = []
export const customModels: AIModel[] = []
