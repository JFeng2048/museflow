import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { fetchProfile } from '@/api/system/auth'
import type { User, UserBindings, BindingProvider } from '@/types/system/auth'

const TOKEN_KEY = 'mf.token'
const USER_KEY = 'mf.user'

/** 兜底用户：API 失败/未登录时使用，确保页面始终有可显示的身份信息。 */
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
  const token = ref<string>(localStorage.getItem(TOKEN_KEY) || '')
  const user = ref<User>(readStoredUser())
  const profileLoaded = ref(false)

  const isLoggedIn = computed(() => !!token.value)
  const displayName = computed(() => user.value?.name || DEFAULT_USER.name)
  const initial = computed(() => displayName.value.slice(0, 1).toUpperCase())

  function setAuth(nextToken: string, nextUser: User) {
    token.value = nextToken
    user.value = nextUser
    profileLoaded.value = true
    persist()
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

  function logout() {
    token.value = ''
    user.value = { ...DEFAULT_USER }
    profileLoaded.value = false
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
  }

  function persist() {
    if (token.value) localStorage.setItem(TOKEN_KEY, token.value)
    localStorage.setItem(USER_KEY, JSON.stringify(user.value))
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
    isLoggedIn,
    displayName,
    initial,
    setAuth,
    update,
    setBinding,
    logout,
    loadProfile,
  }
})

export { DEFAULT_USER }
