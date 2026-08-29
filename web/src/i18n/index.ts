import { createI18n } from 'vue-i18n'
import messages from './locales'

/** 语言偏好在 localStorage 中的存储键。 */
export const LANG_KEY = 'mf.lang'

/** 支持的语言列表。 */
export type AppLocale = 'zh' | 'en'
export const SUPPORTED_LOCALES: AppLocale[] = ['zh', 'en']

/** 读取用户语言偏好：优先 localStorage，其次浏览器语言，兜底英文。 */
export function detectLocale(): AppLocale {
  const saved = localStorage.getItem(LANG_KEY)
  if (saved === 'zh' || saved === 'en') return saved

  const nav = navigator.language?.toLowerCase() ?? ''
  if (nav.startsWith('zh')) return 'zh'
  // 其它任意浏览器语言（en / ja / ko / fr …）都兜底为英文，避免无意义语言显示。
  return 'en'
}

/** 将语言偏好持久化并应用到 <html lang> 与 i18n 实例。 */
export function applyLocale(i18n: any, lang: AppLocale) {
  i18n.global.locale.value = lang
  localStorage.setItem(LANG_KEY, lang)
  document.documentElement.setAttribute('lang', lang)
}

const initialLocale = detectLocale()

// eslint 下 createI18n 返回类型在 legacy:false 时 global 为 Composer，用 any 简化跨模块类型
const i18n: any = createI18n({
  legacy: false,
  locale: initialLocale,
  fallbackLocale: 'en',
  messages,
})

// 首次启动时把探测到的偏好写回 <html lang> 与存储
document.documentElement.setAttribute('lang', initialLocale)
localStorage.setItem(LANG_KEY, initialLocale)

/** 切换语言并持久化（供组件/store 调用）。 */
export function setLanguage(lang: AppLocale) {
  applyLocale(i18n, lang)
}

/** 中文 → 英文 → 中文 循环切换。 */
export function toggleLanguage(): AppLocale {
  const next: AppLocale = i18n.global.locale.value === 'zh' ? 'en' : 'zh'
  setLanguage(next)
  return next
}

/** 组件外便捷 t()。 */
export function t(key: string, named?: Record<string, unknown>): string {
  return i18n.global.t(key, named as any)
}

export default i18n
