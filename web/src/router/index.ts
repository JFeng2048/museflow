import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useUserStore } from '@/stores/user'

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
    redirect: '/dashboard',
    meta: { requiresAuth: true },
    children: [
      { path: 'dashboard', name: 'dashboard', component: () => import('@/views/dashboard/Dashboard.vue') },
      { path: 'novels', name: 'novel-list', component: () => import('@/views/novel/NovelList.vue') },
      { path: 'novels/:id', name: 'novel-detail', component: () => import('@/views/novel/NovelDetail.vue') },
      { path: 'materials', name: 'material', component: () => import('@/views/material/MaterialList.vue') },
      { path: 'lorebook', name: 'lorebook', component: () => import('@/views/lorebook/LorebookPanel.vue') },
      { path: 'tasks', name: 'task', component: () => import('@/views/task/TaskQueue.vue') },
      { path: 'publish', name: 'publish', component: () => import('@/views/publish/PublishSetting.vue') },
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
