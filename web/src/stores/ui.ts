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

  /* --------------------- 人机验证降级开关 ---------------------
   * 由环境变量 VITE_ALLOW_CAPTCHA_FALLBACK 控制（见 .env.development / .env.production）。
   * 仅用于本地调试：Turnstile 脚本加载失败或未配置站点密钥时，是否允许跳过人机验证。
   * 默认关闭，生产环境必须保持关闭，否则等于放弃登录/注册的人机校验。
   * 它与页面上的「演示数据」标注无关——后者只说明该页数据来自本地 @/mock。 */
  const captchaFallback = String(import.meta.env.VITE_ALLOW_CAPTCHA_FALLBACK).toLowerCase() === 'true'

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
    captchaFallback,
  }
})
