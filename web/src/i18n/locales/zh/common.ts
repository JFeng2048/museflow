/**
 * 跨页面通用文案。
 * 这一模块里的 key 在 i18n/locales/{zh,en}/index.ts 中会被 spread 到 locale 根，
 * 因此 t('nav.novels')、t('common.cancel')、t('app.slogan') 都能直接命中，
 * 不需要再走 common.* 这个二级前缀。
 */
export default {
  nav: {
    novels: '作品',
    inspiration: '灵感',
    publish: '发布',
    tasks: '任务',
    statistics: '统计',
    settings: '设置',
  },
  sidebar: {
    dashboard: '我的作品',
    material: '素材库',
    lorebook: '设定集',
    task: '生成任务',
    publish: '发布管理',
  },
  userMenu: {
    profile: '个人资料',
    model: '模型配置',
    credits: '我的积分',
    logout: '退出登录',
    logoutConfirm: '确定要退出当前账号吗？',
  },
  app: {
    name: 'MuseFlow',
    slogan: '你的私人书房，慢慢写，好好写。',
    save: '已保存',
    mockTip: '当前为演示数据模式，接口返回本地兜底数据，未接入真实后端。',
  },
  status: {
    writing: '写作中',
    saved: '已自动保存',
    online: '连接正常',
  },
  // 业务代码用 t('common.cancel') 形式调用，单独再包一层。
  common: {
    save: '保存',
    cancel: '取消',
    confirm: '确认',
    delete: '删除',
    edit: '编辑',
    confirmDelete: '确定删除吗？',
    create: '创建',
    back: '返回',
    loading: '加载中…',
    empty: '暂无内容',
    retry: '重试',
    add: '新增',
    close: '关闭',
    enable: '启用',
    disable: '停用',
    enabled: '已启用',
    disabled: '已停用',
    yes: '是',
    no: '否',
    more: '更多',
    all: '全部',
    others: '其他',
    unset: '未设置',
    optional: '可选',
    required: '必填',
    submit: '提交',
    next: '下一步',
    prev: '上一步',
    finish: '完成',
    search: '搜索作品、素材、设定…',
    writer: '写作者',
    initial: '作',
    ageUnit: '岁',
  },
}
