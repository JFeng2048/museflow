import type { Component } from 'vue'
import {
  BookOutline,
  BulbOutline,
  RocketOutline,
  FolderOutline,
  BarChartOutline,
  SettingsOutline,
  GridOutline,
  PeopleOutline,
  ShieldOutline,
  ConstructOutline,
  MegaphoneOutline,
  ListOutline,
  PulseOutline,
  OptionsOutline,
} from '@vicons/ionicons5'

/** 菜单所属区块：用户工作台(user) 或 管理后台(admin)。 */
export type NavSection = 'user' | 'admin'

export interface NavItem {
  /** 路由 name，用于 router.push 与高亮匹配 */
  name: string
  /** i18n 文案 key（common.nav.* 或 admin.nav.*） */
  labelKey: string
  icon: Component
  section: NavSection
  /**
   * 进入该菜单所需的权限码；缺省表示「已登录即可见」。
   * 前端据此动态显隐，与后端网关/角色权限保持一致。
   */
  permission?: string
}

/**
 * 全站导航清单（单一事实来源）。
 *
 * 用户工作台与后台菜单都从这里派生，是否可见完全由当前用户的权限码决定：
 *  - user 区：作品/素材/发布等创作类，按对应 :read 权限显隐；
 *  - admin 区：各后台页绑定其权限码，system:admin 这类系统级页面仅 super_admin 可见。
 * 想增删菜单或调整可见性，改这里与对应权限码即可，无需动布局组件。
 */
export const NAV_ITEMS: NavItem[] = [
  // ---- 用户工作台 ----
  { name: 'novels', labelKey: 'nav.novels', icon: BookOutline, section: 'user', permission: 'novel:read' },
  { name: 'inspiration', labelKey: 'nav.inspiration', icon: BulbOutline, section: 'user', permission: 'material:read' },
  { name: 'publish', labelKey: 'nav.publish', icon: RocketOutline, section: 'user', permission: 'publish:read' },
  { name: 'tasks', labelKey: 'nav.tasks', icon: FolderOutline, section: 'user' },
  { name: 'statistics', labelKey: 'nav.statistics', icon: BarChartOutline, section: 'user' },
  { name: 'settings', labelKey: 'nav.settings', icon: SettingsOutline, section: 'user' },

  // ---- 管理后台（每项绑定权限码，菜单按权限码动态显隐）----
  { name: 'admin-dashboard', labelKey: 'admin.nav.dashboard', icon: GridOutline, section: 'admin', permission: 'user:admin' },
  { name: 'admin-users', labelKey: 'admin.nav.users', icon: PeopleOutline, section: 'admin', permission: 'user:admin' },
  { name: 'admin-roles', labelKey: 'admin.nav.roles', icon: ShieldOutline, section: 'admin', permission: 'system:admin' },
  { name: 'admin-models', labelKey: 'admin.nav.models', icon: ConstructOutline, section: 'admin', permission: 'system:admin' },
  { name: 'admin-settings', labelKey: 'admin.nav.settings', icon: OptionsOutline, section: 'admin', permission: 'system:admin' },
  { name: 'admin-announcements', labelKey: 'admin.nav.announcements', icon: MegaphoneOutline, section: 'admin', permission: 'user:admin' },
  { name: 'admin-logs', labelKey: 'admin.nav.logs', icon: ListOutline, section: 'admin', permission: 'system:admin' },
  { name: 'admin-services', labelKey: 'admin.nav.services', icon: PulseOutline, section: 'admin', permission: 'system:admin' },
]

/** 按区块与权限码过滤出当前用户可见的菜单。 */
export function filterNav(section: NavSection, permissions: string[]): NavItem[] {
  return NAV_ITEMS.filter(
    (item) =>
      item.section === section &&
      (item.permission === undefined || permissions.includes(item.permission)),
  )
}
