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
  { path: '/:pathMatch(.*)*', redirect: '/dashboard' },
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
  if (to.name === 'login' && userStore.isLoggedIn) {
    return { name: 'dashboard' }
  }
})

export default router
