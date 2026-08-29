import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { themes, getThemeById, applyTheme, DEFAULT_THEME_ID } from '@/themes'
import type { Theme } from '@/themes'
import { setLanguage, toggleLanguage, LANG_KEY, type AppLocale } from '@/i18n'

const THEME_KEY = 'mf.theme'

export const useUiStore = defineStore('ui', () => {
  /* ----------------------------- 主题 ----------------------------- */
  const themeId = ref<string>(localStorage.getItem(THEME_KEY) || DEFAULT_THEME_ID)
  const currentTheme = computed<Theme>(() => getThemeById(themeId.value))

  function setTheme(id: string) {
    const t = getThemeById(id)
    themeId.value = t.id
    applyTheme(t)
    localStorage.setItem(THEME_KEY, t.id)
  }

  /** 按 themes 顺序循环切换（供快捷键使用）。 */
  function cycleTheme() {
    const idx = themes.findIndex((t) => t.id === themeId.value)
    const next = themes[(idx + 1) % themes.length]
    setTheme(next.id)
  }

  function initTheme() {
    applyTheme(currentTheme.value)
  }

  /* ----------------------------- 语言 ----------------------------- */
  const currentLang = ref<string>(localStorage.getItem(LANG_KEY) || 'zh')

  function setLang(lang: AppLocale) {
    currentLang.value = lang
    setLanguage(lang)
  }

  function toggleLang() {
    currentLang.value = toggleLanguage()
  }

  function initLang() {
    document.documentElement.setAttribute('lang', currentLang.value)
  }

  /* --------------------- 演示（Mock）模式 ---------------------
   * 由环境变量 VITE_ENABLE_MOCK 控制（见 .env.development / .env.production）：
   * - true：所有接口走本地兜底数据，并展示「演示数据」标识与登录页演示账号；
   * - false：接入真实后端，隐藏一切演示痕迹。
   * 兜底：未显式配置时按「无真实后端」处理，避免空值导致误判为生产。 */
  const mockMode = String(import.meta.env.VITE_ENABLE_MOCK).toLowerCase() !== 'false'

  return {
    themeId,
    currentTheme,
    setTheme,
    cycleTheme,
    initTheme,
    currentLang,
    setLang,
    toggleLang,
    initLang,
    mockMode,
  }
})
