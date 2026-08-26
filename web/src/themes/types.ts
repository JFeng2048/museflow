export interface Theme {
  id: string
  name: string
  nameEn: string
  icon: string
  description: string
  /** 是否为深色系，用于给 <html> 添加 .dark 类（驱动 Naive UI 暗色）。 */
  dark?: boolean
}
