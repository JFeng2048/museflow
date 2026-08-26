<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { NIcon } from 'naive-ui'
import { MoonOutline, SunnyOutline, LogOutOutline, SearchOutline, SettingsOutline } from '@vicons/ionicons5'
import UserAvatar from '@/components/common/UserAvatar.vue'
import { useUserStore } from '@/stores/system/user'
import { useUiStore } from '@/stores/ui'
import { useI18n } from 'vue-i18n'
const userStore = useUserStore()
const router = useRouter()
const ui = useUiStore()
const { t } = useI18n()
const nickname = computed(() => userStore.user?.name || t('common.writer'))
// 主题切换按钮：在亮色与深色之间切换（深色系涵盖 dark / premium）。
const isDark = computed(() => !!ui.currentTheme.dark)
function toggleTheme() {
  ui.setTheme(isDark.value ? 'light' : 'dark')
}
function logout() {
  userStore.logout()
  router.push({ name: 'login' })
}
</script>
<template>
  <header class="h-14 flex items-center justify-between px-5 border-b border-line bg-paper">
    <div class="left">
      <n-input round :placeholder="t('common.search')" class="max-w-[280px]">
        <template #prefix>
          <n-icon :component="SearchOutline" class="opacity-60" />
        </template>
      </n-input>
    </div>
    <div class="flex items-center gap-2">
      <n-button quaternary circle @click="toggleTheme()">
        <n-icon :component="isDark ? SunnyOutline : MoonOutline" />
      </n-button>
      <n-button quaternary circle @click="router.push({ name: 'settings' })" :title="t('userMenu.settings')">
        <n-icon :component="SettingsOutline" />
      </n-button>
      <div class="flex items-center gap-2 px-1.5">
        <UserAvatar :name="nickname" :avatar="userStore.user?.avatar" :size="32" />
        <span class="text-[14px] text-ink">{{ nickname }}</span>
      </div>
      <n-button quaternary circle @click="logout" :title="t('userMenu.logout')">
        <n-icon :component="LogOutOutline" />
      </n-button>
    </div>
  </header>
</template>

