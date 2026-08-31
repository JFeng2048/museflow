import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { fetchProfile, logout as logoutApi } from '@/api/system/auth'
import { TOKEN_KEY } from '@/constants/auth'
import type { User, UserBindings, BindingProvider, ViewMode } from '@/types/system/auth'

const USER_KEY = 'mf.user'
const VIEW_KEY = 'mf.view'

/** 兜底用户：未登录时使用，确保页面始终有可显示的身份信息。 */
const DEFAULT_USER: User = {
  id: 'u_demo',
  name: '写作者',
  email: 'writer@museflow.app',
  bio: '',
  createdAt: new Date(0).toISOString(),
  bindings: {},
}

/** 从 localStorage 读出已存用户，解析失败则用兜底值。 */
function readStoredUser(): User {
  try {
    const raw = localStorage.getItem(USER_KEY)
    if (!raw) return { ...DEFAULT_USER }
    const parsed = JSON.parse(raw) as User
    if (!parsed || typeof parsed.id !== 'string') return { ...DEFAULT_USER }
    return parsed
  } catch {
    return { ...DEFAULT_USER }
  }
}

export const useUserStore = defineStore('user', () => {
  // 登录态从 localStorage 恢复，保证刷新页面不掉线。
  const token = ref<string>(localStorage.getItem(TOKEN_KEY) || '')
  const user = ref<User>(readStoredUser())
  const profileLoaded = ref(false)
  // 当前激活视图：管理员可切换 user / admin；普通用户固定 user。
  const currentView = ref<ViewMode>(
    (localStorage.getItem(VIEW_KEY) as ViewMode) || 'user',
  )

  const isLoggedIn = computed(() => !!token.value)
  const displayName = computed(() => user.value?.name || DEFAULT_USER.name)
  const initial = computed(() => displayName.value.slice(0, 1).toUpperCase())
  const role = computed(() => user.value?.role || 'writer')
  const isAdmin = computed(() => role.value === 'admin')

  function setAuth(nextToken: string, nextUser: User) {
    token.value = nextToken
    user.value = nextUser
    // 普通用户强制工作台视图；管理员默认进入工作台，稍后由登录页决定。
    currentView.value = nextUser.role === 'admin' ? currentView.value || 'user' : 'user'
    profileLoaded.value = true
    persist()
  }

  function setView(view: ViewMode) {
    currentView.value = view
    localStorage.setItem(VIEW_KEY, view)
  }

  /** 进入管理后台（仅管理员可调用）。 */
  function enterAdmin() {
    if (isAdmin.value) setView('admin')
  }

  /** 返回用户工作台。 */
  function enterUser() {
    setView('user')
  }

  function update(patch: Partial<User>) {
    user.value = { ...user.value, ...patch }
    persist()
  }

  /** 设置某个第三方绑定状态（绑定或解绑）。 */
  function setBinding(provider: BindingProvider, binding: UserBindings[BindingProvider]) {
    const next: UserBindings = { ...(user.value.bindings || {}) }
    if (binding) {
      next[provider] = binding
    } else {
      delete next[provider]
    }
    user.value = { ...user.value, bindings: next }
    persist()
  }

  /** 清理本地登录态。 */
  function clearLocal() {
    token.value = ''
    user.value = { ...DEFAULT_USER }
    profileLoaded.value = false
    currentView.value = 'user'
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
    localStorage.removeItem(VIEW_KEY)
  }

  /**
   * 登出：立即清理本地状态，让界面/路由马上响应；
   * 再用旧令牌后台通知服务端失效（含 refresh token 与其绑定会话）。
   * 网络异常或令牌已失效都不阻塞登出，保证用户点了就能退出。
   */
  function logout() {
    const previousToken = token.value
    clearLocal()
    if (previousToken) {
      logoutApi(previousToken).catch(() => undefined)
    }
  }

  function persist() {
    if (token.value) localStorage.setItem(TOKEN_KEY, token.value)
    localStorage.setItem(USER_KEY, JSON.stringify(user.value))
    localStorage.setItem(VIEW_KEY, currentView.value)
  }

  /**
   * 启动时拉取用户基本信息：
   *  - 未登录：用兜底用户（不发起请求）
   *  - 已登录但本地没缓存：拉取并写入
   *  - 已登录且本地有：直接用本地，避免每次冷启动都打接口
   */
  async function loadProfile(force = false) {
    if (!isLoggedIn.value) {
      user.value = { ...DEFAULT_USER }
      profileLoaded.value = false
      return
    }
    if (profileLoaded.value && !force) return
    try {
      const remote = await fetchProfile()
      user.value = remote
      profileLoaded.value = true
      persist()
    } catch {
      // API 失败时保留本地数据或兜底，profileLoaded 仍置 true 以避免反复重试
      user.value = user.value || { ...DEFAULT_USER }
      profileLoaded.value = true
    }
  }

  return {
    token,
    user,
    currentView,
    isLoggedIn,
    displayName,
    initial,
    role,
    isAdmin,
    setAuth,
    setView,
    enterAdmin,
    enterUser,
    update,
    setBinding,
    logout,
    loadProfile,
  }
})

export { DEFAULT_USER }
