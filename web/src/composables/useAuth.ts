import { useUserStore } from '@/stores/user'

export function useAuth() {
  const userStore = useUserStore()
  return {
    user: userStore.profile,
    isLoggedIn: userStore.isLoggedIn,
    loading: userStore.loading,
    login: userStore.login,
    register: userStore.register,
    logout: userStore.logout,
  }
}
