import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as loginApi, register as registerApi, fetchProfile } from '@/api/user'
import { getStorage, setStorage, removeStorage } from '@/utils/storage'
import type { LoginPayload, RegisterPayload, User } from '@/types'

export const useUserStore = defineStore('user', () => {
  const token = ref<string>(getStorage('token', ''))
  const profile = ref<User | null>(getStorage<User | null>('profile', null))
  const loading = ref(false)

  const isLoggedIn = computed(() => !!token.value)

  async function login(payload: LoginPayload) {
    loading.value = true
    try {
      const res = await loginApi(payload)
      token.value = res.token
      profile.value = res.user
      setStorage('token', res.token)
      setStorage('profile', res.user)
    } finally {
      loading.value = false
    }
  }

  async function register(payload: RegisterPayload) {
    loading.value = true
    try {
      const res = await registerApi(payload)
      token.value = res.token
      profile.value = res.user
      setStorage('token', res.token)
      setStorage('profile', res.user)
    } finally {
      loading.value = false
    }
  }

  async function restore() {
    if (!token.value) return
    try {
      profile.value = await fetchProfile()
      setStorage('profile', profile.value)
    } catch {
      /* keep cached profile */
    }
  }

  function logout() {
    token.value = ''
    profile.value = null
    removeStorage('token')
    removeStorage('profile')
  }

  return { token, profile, loading, isLoggedIn, login, register, restore, logout }
})
