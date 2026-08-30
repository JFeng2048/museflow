/**
 * 单例加载 Cloudflare Turnstile 脚本。
 * 仅在真实环境（非 mock）需要；mock 模式下 window.turnstile 不存在，
 * 调用方应降级为「跳过人机验证」（开发期友好）。
 */
const SRC = 'https://challenges.cloudflare.com/turnstile/v0/api.js'

let scriptPromise: Promise<void> | null = null

declare global {
  interface Window {
    turnstile?: {
      render: (el: string | HTMLElement, opts: Record<string, unknown>) => string
      reset: (widgetId?: string) => void
      remove: (widgetId: string) => void
      getResponse: (widgetId?: string) => string
    }
  }
}

export function loadTurnstileScript(): Promise<void> {
  if (typeof window === 'undefined') return Promise.resolve()
  if (window.turnstile) return Promise.resolve()
  if (scriptPromise) return scriptPromise

  scriptPromise = new Promise<void>((resolve, reject) => {
    const s = document.createElement('script')
    s.src = SRC
    s.async = true
    s.defer = true
    s.onload = () => resolve()
    s.onerror = () => reject(new Error('turnstile script load failed'))
    document.head.appendChild(s)
  })
  return scriptPromise
}

export function isTurnstileAvailable(): boolean {
  return typeof window !== 'undefined' && !!window.turnstile
}
