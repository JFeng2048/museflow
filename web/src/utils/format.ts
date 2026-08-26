export function formatDate(input: string | number | Date): string {
  const date = new Date(input)
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}

export function formatDateTime(input: string | number | Date): string {
  const date = new Date(input)
  const h = String(date.getHours()).padStart(2, '0')
  const min = String(date.getMinutes()).padStart(2, '0')
  return `${formatDate(date)} ${h}:${min}`
}

export function formatWords(count: number): string {
  if (count >= 10000) return `${(count / 10000).toFixed(1)} 万字`
  return `${count.toLocaleString('zh-CN')} 字`
}

export function formatRelative(input: string | number | Date): string {
  const diff = Date.now() - new Date(input).getTime()
  const minute = 60 * 1000
  const hour = 60 * minute
  const day = 24 * hour
  if (diff < hour) return `${Math.max(1, Math.floor(diff / minute))} 分钟前`
  if (diff < day) return `${Math.floor(diff / hour)} 小时前`
  if (diff < 30 * day) return `${Math.floor(diff / day)} 天前`
  return formatDate(input)
}

/** 将十六进制颜色按百分比调亮（正数）或调暗（负数）。amount ∈ [-100, 100]。 */
export function shade(hex: string, amount: number): string {
  const h = hex.replace('#', '')
  const full = h.length === 3 ? h.split('').map((c) => c + c).join('') : h
  const num = parseInt(full, 16)
  let r = (num >> 16) & 0xff
  let g = (num >> 8) & 0xff
  let b = num & 0xff
  const t = amount < 0 ? 0 : 255
  const p = amount < 0 ? amount / -100 : amount / 100
  r = Math.round((t - r) * p + r)
  g = Math.round((t - g) * p + g)
  b = Math.round((t - b) * p + b)
  return `#${((1 << 24) + (r << 16) + (g << 8) + b).toString(16).slice(1)}`
}

/** 取颜色相对可读的前景色（黑 / 白）。 */
export function readableOn(hex: string): string {
  const h = hex.replace('#', '')
  const full = h.length === 3 ? h.split('').map((c) => c + c).join('') : h
  const num = parseInt(full, 16)
  const r = (num >> 16) & 0xff
  const g = (num >> 8) & 0xff
  const b = num & 0xff
  const lum = (0.299 * r + 0.587 * g + 0.114 * b) / 255
  return lum > 0.62 ? '#1a2332' : '#fffdf9'
}
