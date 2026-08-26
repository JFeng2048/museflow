import { useUserStore } from '@/stores/system/user'

/** 业务层统一从 user store 取鉴权信息，避免到处 useUserStore。 */
export function useAuth() {
  const userStore = useUserStore()
  return {
    user: userStore.user,
    isLoggedIn: userStore.isLoggedIn,
    displayName: userStore.displayName,
    initial: userStore.initial,
    setAuth: userStore.setAuth,
    update: userStore.update,
    logout: userStore.logout,
    loadProfile: userStore.loadProfile,
  }
}
