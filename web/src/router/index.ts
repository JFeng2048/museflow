import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useUserStore } from '@/stores/system/user'

const routes: RouteRecordRaw[] = [
  {
    path: '/auth',
    component: () => import('@/layouts/AuthLayout.vue'),
    children: [
      { path: 'login', name: 'login', component: () => import('@/views/auth/Login.vue') },
      { path: 'register', name: 'register', component: () => import('@/views/auth/Register.vue') },
    ],
  },
  {
    path: '/',
    component: () => import('@/layouts/DefaultLayout.vue'),
    redirect: '/novels',
    meta: { requiresAuth: true },
    children: [
      { path: 'dashboard', name: 'dashboard', component: () => import('@/views/dashboard/index.vue') },
      { path: 'novels', name: 'novels', component: () => import('@/views/novel/index.vue') },
      { path: 'novels/:id', name: 'novel-detail', component: () => import('@/views/novel/NovelDetail.vue') },
      { path: 'inspiration', name: 'inspiration', component: () => import('@/views/inspiration/index.vue') },
      { path: 'lorebook', name: 'lorebook', component: () => import('@/views/lorebook/index.vue') },
      { path: 'materials', name: 'material', component: () => import('@/views/material/index.vue') },
      { path: 'publish', name: 'publish', component: () => import('@/views/publish/index.vue') },
      { path: 'task', name: 'tasks', component: () => import('@/views/task/index.vue') },
      { path: 'statistics', name: 'statistics', component: () => import('@/views/statistics/index.vue') },
      { path: 'settings', name: 'settings', component: () => import('@/views/settings/index.vue') },
    ],
  },
  {
    path: '/admin',
    component: () => import('@/layouts/AdminLayout.vue'),
    redirect: '/admin/dashboard',
    meta: { requiresAuth: true },
    children: [
      { path: 'dashboard', name: 'admin-dashboard', component: () => import('@/views/admin/Dashboard.vue'), meta: { permission: 'user:admin' } },
      { path: 'users', name: 'admin-users', component: () => import('@/views/admin/Users.vue'), meta: { permission: 'user:admin' } },
      { path: 'roles', name: 'admin-roles', component: () => import('@/views/admin/Roles.vue'), meta: { permission: 'system:admin' } },
      { path: 'models', name: 'admin-models', component: () => import('@/views/admin/Models.vue'), meta: { permission: 'system:admin' } },
      { path: 'settings', name: 'admin-settings', component: () => import('@/views/admin/Settings.vue'), meta: { permission: 'system:admin' } },
      { path: 'announcements', name: 'admin-announcements', component: () => import('@/views/admin/Announcements.vue'), meta: { permission: 'user:admin' } },
      { path: 'logs', name: 'admin-logs', component: () => import('@/views/admin/Logs.vue'), meta: { permission: 'system:admin' } },
      { path: 'services', name: 'admin-services', component: () => import('@/views/admin/Services.vue'), meta: { permission: 'system:admin' } },
    ],
  },
  { path: '/:pathMatch(.*)*', redirect: '/novels' },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to) => {
  const userStore = useUserStore()
  if (to.meta.requiresAuth && !userStore.isLoggedIn) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  // 按路由声明的权限码校验：无权限则退回工作台（菜单本就按权限码隐藏，这里兜住直链访问）。
  if (to.meta.permission && !userStore.hasPermission(to.meta.permission as string)) {
    return { name: 'novels' }
  }
  if (to.name === 'login' && userStore.isLoggedIn) {
    // 已登录用户访问登录页：按当前视图跳回对应工作台。
    return userStore.canEnterAdmin && userStore.currentView === 'admin'
      ? { name: 'admin-dashboard' }
      : { name: 'novels' }
  }
})

export default router
