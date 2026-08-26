<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { NIcon } from 'naive-ui'
import { useUserStore } from '@/stores/system/user'
import { useCreditStore } from '@/stores/credit'
import {
  BookOutline,
  BulbOutline,
  RocketOutline,
  FolderOutline,
  BarChartOutline,
  SettingsOutline,
} from '@vicons/ionicons5'
import ThemeSwitcher from '@/components/common/ThemeSwitcher.vue'
import LanguageSwitcher from '@/components/common/LanguageSwitcher.vue'
import UserMenu from '@/components/layout/UserMenu.vue'
import AppLogo from '@/components/common/AppLogo.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const creditStore = useCreditStore()
const { activityBalance, permanentBalance, validBalance } = storeToRefs(creditStore)

const nav = computed(() => [
  { label: t('nav.novels'), name: 'novels', icon: BookOutline },
  { label: t('nav.inspiration'), name: 'inspiration', icon: BulbOutline },
  { label: t('nav.publish'), name: 'publish', icon: RocketOutline },
  { label: t('nav.tasks'), name: 'tasks', icon: FolderOutline },
  { label: t('nav.statistics'), name: 'statistics', icon: BarChartOutline },
  { label: t('nav.settings'), name: 'settings', icon: SettingsOutline },
])

const active = computed(() => String(route.name))
function go(name: string) {
  router.push({ name })
}

function logout() {
  userStore.logout()
  router.push({ name: 'login' })
}
</script>

<template>
  <div class="layout-shell">
    <!-- 顶部栏 -->
    <header class="layout-topbar">
      <div class="layout-brand" @click="go('novels')">
        <AppLogo :size="30" />
        <span class="layout-brand-name">MuseFlow</span>
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
        <!-- 积分（活动 + 永久） -->
        <div class="credit-chip" :title="`${t('credits.activity')} ${activityBalance} / ${t('credits.permanent')} ${permanentBalance}`">
          <span class="credit-coin">✦</span>
          <span class="credit-bal">{{ validBalance }}</span>
          <span class="credit-exp" v-if="activityBalance">{{ t('credits.activityShort', { n: activityBalance }) }}</span>
        </div>

        <ThemeSwitcher />
        <LanguageSwitcher />
        <UserMenu />
      </div>
    </header>

    <!-- 主体 -->
    <main class="layout-content">
      <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </main>

    <!-- 底部状态栏 -->
    <footer class="layout-statusbar">
      <div class="layout-status-left">
        <span class="layout-status-dot" />
        <span>{{ t('app.slogan') }}</span>
      </div>
      <div class="layout-status-right">
        <span>{{ t('status.writing') }}</span>
        <span class="layout-status-sep">·</span>
        <span>{{ userStore.user?.name || '写作者' }}</span>
      </div>
    </footer>
  </div>
</template>

