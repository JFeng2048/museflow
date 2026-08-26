import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ModelProvider, AIModel, ModelProtocol } from '@/types/model'
import {
  systemProviders,
  systemModels,
  customProviders as seedProviders,
  customModels as seedModels,
} from '@/mock/models'

export const useModelStore = defineStore('model', () => {
  const providers = ref<ModelProvider[]>([...systemProviders, ...seedProviders])
  const models = ref<AIModel[]>([...systemModels, ...seedModels])

  const systemModelsList = () => models.value.filter((m) => m.system)
  const customModelsList = () => models.value.filter((m) => !m.system)
  const enabledModels = () => models.value.filter((m) => m.enabled)
  const getProvider = (id: string) => providers.value.find((p) => p.id === id)
  const getModel = (id: string) => models.value.find((m) => m.id === id)

  // —— 自定义供应商 ——
  function addProvider(input: Omit<ModelProvider, 'id' | 'createdAt' | 'system'>) {
    const p: ModelProvider = {
      ...input,
      id: 'prov_' + Math.random().toString(36).slice(2, 8),
      system: false,
      createdAt: new Date().toISOString(),
    }
    providers.value.push(p)
    return p
  }
  function updateProvider(id: string, patch: Partial<ModelProvider>) {
    const p = providers.value.find((x) => x.id === id)
    if (p && !p.system) Object.assign(p, patch)
  }
  function removeProvider(id: string) {
    const p = providers.value.find((x) => x.id === id)
    if (p?.system) return
    providers.value = providers.value.filter((x) => x.id !== id)
    models.value = models.value.filter((m) => m.providerId !== id)
  }

  // —— 自定义模型 ——
  function addModel(input: Omit<AIModel, 'id' | 'system' | 'creditCost' | 'enabled'>) {
    const m: AIModel = {
      ...input,
      id: 'm_' + Math.random().toString(36).slice(2, 8),
      system: false,
      creditCost: 0, // 自定义模型免费
      enabled: true,
    }
    models.value.push(m)
    return m
  }
  function updateModel(id: string, patch: Partial<AIModel>) {
    const m = models.value.find((x) => x.id === id)
    if (m && !m.system) Object.assign(m, patch, { creditCost: 0 })
  }
  function toggleModel(id: string, enabled: boolean) {
    const m = models.value.find((x) => x.id === id)
    if (m) m.enabled = enabled
  }
  function removeModel(id: string) {
    const m = models.value.find((x) => x.id === id)
    if (m?.system) return
    models.value = models.value.filter((x) => x.id !== id)
  }

  return {
    providers,
    models,
    systemModelsList,
    customModelsList,
    enabledModels,
    getProvider,
    getModel,
    addProvider,
    updateProvider,
    removeProvider,
    addModel,
    updateModel,
    toggleModel,
    removeModel,
  }
})
