import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'
import type {
  ModelProvider,
  AIModel,
  ModelType,
  UserProvider,
  UserModel,
  AvailableModel,
  RemoteModel,
  SystemSetting,
} from '@/types/model'
import {
  listProviders,
  createProvider,
  updateProvider,
  setProviderActive,
  deleteProvider,
  listModels,
  createModel,
  updateModel,
  setModelActive,
  deleteModel,
  listUserProviders,
  createUserProvider,
  updateUserProvider,
  deleteUserProvider,
  listUserModels,
  createUserModel,
  updateUserModel,
  deleteUserModel,
  fetchAdminRemoteModels,
  fetchUserRemoteModels,
  listAvailableModels,
  listPublicSettings,
  listSettings,
  upsertSetting,
  deleteSetting,
} from '@/api/model'

import type { CreateModelPayload, UserModelPayload } from '@/api/model'

/**
 * 模型目录与系统配置 store。
 *
 * 平台侧（providers / models）与用户侧（userProviders / userModels / available）
 * 都是真实接口的镜像：列表连同筛选条件一起缓存在这里，写操作成功后按当前筛选刷新，
 * 保证页面上的停用开关、新增渠道立刻反映到用户端可见性。
 */
export const useModelStore = defineStore('model', () => {
  // ---------- 平台渠道 ----------
  const providers = ref<ModelProvider[]>([])
  const providersTotal = ref(0)
  const loadingProviders = ref(false)
  const providerFilter = reactive({
    page: 1,
    pageSize: 20,
    keyword: '',
    onlyActive: false,
  })

  // ---------- 平台模型 ----------
  const models = ref<AIModel[]>([])
  const modelsTotal = ref(0)
  const loadingModels = ref(false)
  const modelFilter = reactive({
    page: 1,
    pageSize: 20,
    providerId: 0,
    modelType: '' as ModelType | '',
    keyword: '',
    onlyActive: false,
  })

  // ---------- 用户侧 ----------
  const userProviders = ref<UserProvider[]>([])
  const userModels = ref<UserModel[]>([])
  const available = ref<AvailableModel[]>([])
  const availableType = ref<ModelType | ''>('')
  const publicSettings = ref<SystemSetting[]>([])
  const loadingUser = ref(false)

  // ---------- 上游模型目录 ----------
  // 与列表数据分开存：目录是「还没登记」的候选，勾选后才落库，
  // 混进 models / userModels 会让页面出现实际不存在的条目。
  const remoteModels = ref<RemoteModel[]>([])
  const loadingRemote = ref(false)

  // ---------- 系统配置（管理端）----------
  const settings = ref<SystemSetting[]>([])
  const settingFilter = reactive({
    configGroup: '',
    onlyPublic: false,
  })
  const loadingSettings = ref(false)

  async function fetchProviders() {
    loadingProviders.value = true
    try {
      const page = await listProviders(providerFilter)
      providers.value = page.items
      providersTotal.value = page.total
    } finally {
      loadingProviders.value = false
    }
  }

  async function fetchModels() {
    loadingModels.value = true
    try {
      const page = await listModels(modelFilter)
      models.value = page.items
      modelsTotal.value = page.total
    } finally {
      loadingModels.value = false
    }
  }

  async function fetchUserData() {
    loadingUser.value = true
    try {
      const [p, m, a] = await Promise.all([
        listUserProviders({ pageSize: 100 }),
        listUserModels({ pageSize: 100 }),
        listAvailableModels({ modelType: availableType.value }),
      ])
      userProviders.value = p.items
      userModels.value = m.items
      available.value = a
    } finally {
      loadingUser.value = false
    }
  }

  /** 可用模型清单：用户端模型选择器用，按当前类型筛选刷新。 */
  async function fetchAvailable() {
    available.value = await listAvailableModels({ modelType: availableType.value })
  }

  /** 公开系统配置：只读，登录后由设置页拉取。 */
  async function fetchPublicSettings() {
    publicSettings.value = await listPublicSettings()
  }

  // ---------- 上游模型目录 ----------

  /** 拉取平台渠道的模型目录；失败时清空候选，避免看到上一轮的残留。 */
  async function fetchAdminCatalog(providerId: number) {
    loadingRemote.value = true
    try {
      remoteModels.value = await fetchAdminRemoteModels({ providerId })
    } catch {
      remoteModels.value = []
      throw new Error('catalog')
    } finally {
      loadingRemote.value = false
    }
  }

  /** 拉取我的自定义渠道的模型目录。 */
  async function fetchUserCatalog(providerId: number) {
    loadingRemote.value = true
    try {
      remoteModels.value = await fetchUserRemoteModels({ providerId })
    } catch {
      remoteModels.value = []
      throw new Error('catalog')
    } finally {
      loadingRemote.value = false
    }
  }

  /**
   * 批量登记上游目录里勾选的模型。
   *
   * 逐条创建而不是调批量接口：单条失败（撞 code 唯一键、渠道被删）不影响其他条目，
   * 失败清单交回弹窗让用户改完重试。列表只在全部结束后刷新一次，
   * 否则勾 12 个模型就要跟着发 12 次列表请求，页面会一直闪。
   */
  async function registerCatalogModels(
    scope: 'platform' | 'user',
    items: (CreateModelPayload | UserModelPayload)[],
  ) {
    const failed: string[] = []
    for (const item of items) {
      try {
        if (scope === 'platform') {
          await createModel(item as CreateModelPayload)
        } else {
          await createUserModel(item as UserModelPayload)
        }
      } catch {
        failed.push(item.apiModel)
      }
    }
    // 刷新放在循环外：登记成功数大于 0 时列表才需要动，全失败时保持原样。
    if (failed.length < items.length) {
      if (scope === 'platform') {
        await fetchModels()
        await fetchAvailable()
      } else {
        await fetchUserData()
      }
    }
    return { ok: items.length - failed.length, failed }
  }

  // ---------- 系统配置写操作 ----------

  async function fetchSettings() {
    loadingSettings.value = true
    try {
      settings.value = await listSettings(settingFilter)
    } finally {
      loadingSettings.value = false
    }
  }

  /** 按 (分组, 键名) 写入：存在则更新、不存在则创建。 */
  async function saveSetting(group: string, key: string, payload: Parameters<typeof upsertSetting>[2]) {
    const saved = await upsertSetting(group, key, payload)
    await fetchSettings()
    return saved
  }

  async function removeSetting(group: string, key: string) {
    await deleteSetting(group, key)
    await fetchSettings()
  }

  // ---------- 平台渠道写操作 ----------

  async function addProvider(payload: Parameters<typeof createProvider>[0]) {
    const created = await createProvider(payload)
    await fetchProviders()
    await fetchAvailable()
    return created
  }

  async function editProvider(id: number, payload: Parameters<typeof updateProvider>[1]) {
    const updated = await updateProvider(id, payload)
    await fetchProviders()
    await fetchAvailable()
    return updated
  }

  async function toggleProvider(id: number, isActive: boolean) {
    const updated = await setProviderActive(id, isActive)
    await fetchProviders()
    await fetchAvailable()
    return updated
  }

  async function removeProvider(id: number) {
    await deleteProvider(id)
    await fetchProviders()
    await fetchAvailable()
  }

  // ---------- 平台模型写操作 ----------

  async function addModel(payload: Parameters<typeof createModel>[0]) {
    const created = await createModel(payload)
    await fetchModels()
    await fetchAvailable()
    return created
  }

  async function editModel(id: number, payload: Parameters<typeof updateModel>[1]) {
    const updated = await updateModel(id, payload)
    await fetchModels()
    await fetchAvailable()
    return updated
  }

  async function toggleModel(id: number, isActive: boolean) {
    const updated = await setModelActive(id, isActive)
    await fetchModels()
    await fetchAvailable()
    return updated
  }

  async function removeModel(id: number) {
    await deleteModel(id)
    await fetchModels()
    await fetchAvailable()
  }

  // ---------- 用户侧写操作 ----------

  async function addUserProvider(payload: Parameters<typeof createUserProvider>[0]) {
    const created = await createUserProvider(payload)
    await fetchUserData()
    return created
  }

  async function editUserProvider(id: number, payload: Parameters<typeof updateUserProvider>[1]) {
    const updated = await updateUserProvider(id, payload)
    await fetchUserData()
    return updated
  }

  async function removeUserProvider(id: number) {
    await deleteUserProvider(id)
    await fetchUserData()
  }

  async function addUserModel(payload: Parameters<typeof createUserModel>[0]) {
    const created = await createUserModel(payload)
    await fetchUserData()
    return created
  }

  async function editUserModel(id: number, payload: Parameters<typeof updateUserModel>[1]) {
    const updated = await updateUserModel(id, payload)
    await fetchUserData()
    return updated
  }

  async function removeUserModel(id: number) {
    await deleteUserModel(id)
    await fetchUserData()
  }

  /** 平台已上架的模型（不含用户自定义），按当前模型筛选派生。 */
  const platformModels = () => available.value.filter((m) => m.source === 'platform')

  /** 用户自定义的模型，按当前模型筛选派生。 */
  const customModels = () => available.value.filter((m) => m.source === 'custom')

  /** 模型 ID → 所属渠道名（平台渠道或自定义渠道），供列表展示。 */
  function providerName(id: number, custom = false): string {
    const list = custom ? userProviders.value : providers.value
    return list.find((p) => p.id === id)?.name || '—'
  }

  return {
    providers,
    providersTotal,
    loadingProviders,
    providerFilter,
    models,
    modelsTotal,
    loadingModels,
    modelFilter,
    userProviders,
    userModels,
    available,
    availableType,
    publicSettings,
    loadingUser,
    settings,
    settingFilter,
    loadingSettings,
    fetchProviders,
    fetchModels,
    fetchUserData,
    fetchAvailable,
    fetchPublicSettings,
    fetchSettings,
    saveSetting,
    removeSetting,
    addProvider,
    editProvider,
    toggleProvider,
    removeProvider,
    addModel,
    editModel,
    toggleModel,
    removeModel,
    addUserProvider,
    editUserProvider,
    removeUserProvider,
    addUserModel,
    editUserModel,
    removeUserModel,
    remoteModels,
    loadingRemote,
    fetchAdminCatalog,
    fetchUserCatalog,
    registerCatalogModels,
    platformModels,
    customModels,
    providerName,
  }
})
