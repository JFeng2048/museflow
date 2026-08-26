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
import { useI18n } from 'vue-i18n'

defineProps<{ collapsed: boolean }>()
const emit = defineEmits<{ (e: 'toggle'): void }>()

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const menus = [
  { name: 'dashboard', label: t('sidebar.dashboard'), icon: HomeOutline },
  { name: 'material', label: t('sidebar.material'), icon: LibraryOutline },
  { name: 'lorebook', label: t('sidebar.lorebook'), icon: CompassOutline },
  { name: 'tasks', label: t('sidebar.task'), icon: FlaskOutline },
  { name: 'publish', label: t('sidebar.publish'), icon: SendOutline },
]

const active = computed(() => {
  if (route.name === 'novels' || route.name === 'novel-detail') return 'dashboard'
  return route.name
})

function go(name: string) {
  router.push({ name })
}
</script>

<template>
  <aside
    class="flex flex-col py-3.5 px-3 border-r border-line bg-warm-2 transition-[width] duration-200"
    :style="{ width: collapsed ? '64px' : '220px' }"
  >
    <div class="flex items-center justify-between mb-4.5 px-1">
      <AppLogo :collapsed="collapsed" />
      <n-button quaternary size="small" class="opacity-60" @click="emit('toggle')">
        <n-icon :component="BookOutline" />
      </n-button>
    </div>
    <nav class="flex flex-col gap-1">
      <button
        v-for="m in menus"
        :key="m.name"
        class="sidebar-item"
        :class="{ active: active === m.name }"
        @click="go(m.name)"
      >
        <n-icon :component="m.icon" />
        <span v-if="!collapsed">{{ m.label }}</span>
      </button>
    </nav>
  </aside>
</template>

