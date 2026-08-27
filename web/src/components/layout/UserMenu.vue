<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { NIcon, useDialog } from 'naive-ui'
import { LogOutOutline } from '@vicons/ionicons5'
import { useUserStore } from '@/stores/system/user'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const userStore = useUserStore()
const dialog = useDialog()
const { t } = useI18n()
const open = ref(false)
let closeTimer: ReturnType<typeof setTimeout> | null = null

function openMenu() {
  if (closeTimer) {
    clearTimeout(closeTimer)
    closeTimer = null
  }
  open.value = true
}

function scheduleClose() {
  if (closeTimer) clearTimeout(closeTimer)
  closeTimer = setTimeout(() => {
    open.value = false
    closeTimer = null
  }, 350)
}

function confirmLogout() {
  open.value = false
  userStore.logout()
  router.replace({ name: 'login' })
}

function askLogout() {
  open.value = false
  dialog.warning({
    title: t('userMenu.logout'),
    content: t('userMenu.logoutConfirm'),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: confirmLogout,
  })
}
</script>

<template>
  <div class="usermenu-root">
    <button class="usermenu-trigger" @click="open = !open" @mouseenter="openMenu" @mouseleave="scheduleClose">
      <span class="usermenu-avatar">{{ userStore.user?.name?.slice(0, 1) || t('common.initial') }}</span>
      <span class="usermenu-uname">{{ userStore.user?.name }}</span>
    </button>
    <transition name="drop">
      <div v-if="open" class="usermenu-menu" @click.stop @mouseenter="openMenu" @mouseleave="scheduleClose">
        <div class="px-2.5 py-2">
          <p class="font-semibold m-0">{{ userStore.user?.name }}</p>
          <p class="text-[12px] text-ink-muted mt-0.5">{{ userStore.user?.email }}</p>
        </div>
        <div class="h-px bg-line my-1" />
        <button class="usermenu-item usermenu-danger" @click="askLogout">
          <n-icon :component="LogOutOutline" /> {{ t('userMenu.logout') }}
        </button>
      </div>
    </transition>
  </div>
</template>
