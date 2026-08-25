import { useStorage } from '@vueuse/core'

// 单一真源：全局共享的暗色状态，避免多个 useDark 实例不同步。
const isDark = useStorage('museflow:theme-dark', false)

function sync() {
  if (typeof document !== 'undefined') {
    document.documentElement.classList.toggle('dark', isDark.value)
  }
}

sync()

export function useTheme() {
  function toggle() {
    isDark.value = !isDark.value
    sync()
  }
  return { isDark, toggle }
}
