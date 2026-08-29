<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { NIcon } from 'naive-ui'
import { useUserStore } from '@/stores/system/user'
import {
  GridOutline,
  PeopleOutline,
  ConstructOutline,
  MegaphoneOutline,
  ListOutline,
  PulseOutline,
  ExitOutline,
} from '@vicons/ionicons5'
import ThemeSwitcher from '@/components/common/ThemeSwitcher.vue'
import LanguageSwitcher from '@/components/common/LanguageSwitcher.vue'
import UserMenu from '@/components/layout/UserMenu.vue'
import AppLogo from '@/components/common/AppLogo.vue'
import { useUiStore } from '@/stores/ui'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const { currentView } = storeToRefs(userStore)
const ui = useUiStore()

const mockLabel = computed(() => (ui.currentLang === 'zh' ? '演示数据' : 'Demo Data'))

const nav = computed(() => [
  { label: t('admin.nav.dashboard'), name: 'admin-dashboard', icon: GridOutline },
  { label: t('admin.nav.users'), name: 'admin-users', icon: PeopleOutline },
  { label: t('admin.nav.models'), name: 'admin-models', icon: ConstructOutline },
  { label: t('admin.nav.announcements'), name: 'admin-announcements', icon: MegaphoneOutline },
  { label: t('admin.nav.logs'), name: 'admin-logs', icon: ListOutline },
  { label: t('admin.nav.services'), name: 'admin-services', icon: PulseOutline },
])

const active = computed(() => String(route.name))
function go(name: string) {
  router.push({ name })
}

function backToUser() {
  userStore.enterUser()
  router.push('/novels')
}
</script>

<template>
  <div class="layout-shell">
    <header class="layout-topbar">
      <div class="layout-brand" @click="go('admin-dashboard')">
        <AppLogo :size="30" />
        <span class="layout-brand-name">MuseFlow <em class="admin-badge">{{ t('admin.badge') }}</em></span>
      </div>

      <nav class="layout-nav">
        <button
          v-for="item in nav"
          :key="item.name"
          class="layout-nav-item"
          :class="{ active: active === item.name }"
          @click="go(item.name)"
        >
          <n-icon :component="item.icon" class="layout-nav-ico" />
          <span>{{ item.label }}</span>
        </button>
      </nav>

      <div class="layout-right">
        <!-- 演示（Mock）模式标识：仅未接入真实后端时显示 -->
        <span v-if="ui.mockMode" class="mock-flag" :title="t('app.mockTip')">{{ mockLabel }}</span>

        <!-- 管理员可随时返回用户工作台 -->
        <button class="switch-btn" @click="backToUser" :title="t('admin.backToWorkbench')">
          <n-icon :component="ExitOutline" :size="16" />
          <span>{{ t('admin.backToWorkbench') }}</span>
        </button>
        <ThemeSwitcher />
        <LanguageSwitcher />
        <UserMenu />
      </div>
    </header>

    <main class="layout-content">
      <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </main>

    <footer class="layout-statusbar">
      <div class="layout-status-left">
        <span class="layout-status-dot" />
        <span>{{ t('app.slogan') }}</span>
      </div>
      <div class="layout-status-right">
        <span>{{ t('admin.badge') }}</span>
        <span class="layout-status-sep">·</span>
        <span>{{ userStore.user?.name || '管理员' }}</span>
      </div>
    </footer>
  </div>
</template>
