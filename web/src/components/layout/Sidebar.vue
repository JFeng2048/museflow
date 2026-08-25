<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import { NIcon } from 'naive-ui'
import { h, computed } from 'vue'
import {
  BookOutline,
  LibraryOutline,
  CompassOutline,
  FlaskOutline,
  SendOutline,
  HomeOutline,
} from '@vicons/ionicons5'
import AppLogo from '@/components/common/AppLogo.vue'

defineProps<{ collapsed: boolean }>()
const emit = defineEmits<{ (e: 'toggle'): void }>()

const route = useRoute()
const router = useRouter()

const menus = [
  { name: 'dashboard', label: '我的项目', icon: HomeOutline },
  { name: 'material', label: '素材库', icon: LibraryOutline },
  { name: 'lorebook', label: '设定集', icon: CompassOutline },
  { name: 'task', label: '生成任务', icon: FlaskOutline },
  { name: 'publish', label: '发布管理', icon: SendOutline },
]

const active = computed(() => {
  if (route.name === 'novel-list' || route.name === 'novel-detail') return 'dashboard'
  return route.name
})

function go(name: string) {
  router.push({ name })
}
</script>

<template>
  <aside class="sidebar" :class="{ collapsed }">
    <div class="brand">
      <AppLogo :collapsed="collapsed" />
      <n-button quaternary size="small" class="toggle" @click="emit('toggle')">
        <n-icon :component="BookOutline" />
      </n-button>
    </div>
    <nav class="menu">
      <button
        v-for="m in menus"
        :key="m.name"
        class="item"
        :class="{ active: active === m.name }"
        @click="go(m.name)"
      >
        <n-icon :component="m.icon" />
        <span v-if="!collapsed" class="label">{{ m.label }}</span>
      </button>
    </nav>
  </aside>
</template>

<style scoped>
.sidebar {
  width: 220px;
  transition: width 0.2s ease;
  background: var(--mf-surface-2);
  border-right: 1px solid var(--mf-border);
  display: flex;
  flex-direction: column;
  padding: 14px 12px;
}
.sidebar.collapsed {
  width: 64px;
}
.brand {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 18px;
  padding: 0 4px;
}
.toggle {
  opacity: 0.6;
}
.menu {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 9px 10px;
  border: none;
  background: transparent;
  border-radius: 8px;
  cursor: pointer;
  color: var(--mf-text-2);
  font-size: 14px;
  text-align: left;
}
.item:hover {
  background: var(--mf-hover);
}
.item.active {
  background: var(--mf-active-bg);
  color: var(--mf-active-text);
  font-weight: 600;
}
</style>
