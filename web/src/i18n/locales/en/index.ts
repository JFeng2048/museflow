import common from './common'
import auth from './auth'
import editor from './editor'
import novel from './novel'
import hotspot from './hotspot'
import publish from './publish'
import settings from './settings'
import model from './model'
import credits from './credits'
import task from './task'
import inspiration from './inspiration'
import material from './material'
import lore from './lore'
import dashboard from './dashboard'
import stats from './stats'
import admin from './admin'
import identity from './identity'
import security from './security'

/**
 * 命名空间聚合：t('dashboard.eyebrow') → en.dashboard.eyebrow。
 * common 是跨页面通用文案（nav/sidebar/userMenu/app/status/save/cancel/...），
 * 业务代码直接用 t('nav.novels')、t('common.cancel') 这类扁平 key，
 * 因此这里把它 spread 到 locale 根而不是嵌套到 common.*。
 * 其他模块各自一个命名空间。
 */
export default {
  ...common,
  auth,
  editor,
  novel,
  hotspot,
  publish,
  settings,
  model,
  credits,
  task,
  inspiration,
  material,
  lore,
  dashboard,
  stats,
  admin,
  identity,
  security,
}
