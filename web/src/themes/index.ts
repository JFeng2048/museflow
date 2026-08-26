import type { Theme } from './types'
import light from './light'
import dark from './dark'
import eyeCare from './eye-care'
import focus from './focus'
import retro from './retro'
import premium from './premium'

export type { Theme }

/** 主题按优先级排列，循环切换时依此顺序。 */
export const themes: Theme[] = [light, dark, eyeCare, focus, retro, premium]

export const DEFAULT_THEME_ID = 'light'

export function getThemeById(id: string): Theme {
  return themes.find((t) => t.id === id) || light
}

/**
 * 应用主题：仅设置 <html> 的 data-theme 与 .dark 类，
 * 具体颜色由 src/assets/styles/index.css 中 [data-theme] 选择器定义。
 */
export function applyTheme(theme: Theme) {
  const root = document.documentElement
  root.setAttribute('data-theme', theme.id)
  root.classList.toggle('dark', !!theme.dark)
}
