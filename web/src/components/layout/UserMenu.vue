<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { NIcon, NPopconfirm } from 'naive-ui'
import { LogOutOutline } from '@vicons/ionicons5'
import { useUserStore } from '@/stores/system/user'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const userStore = useUserStore()
const { t } = useI18n()
const open = ref(false)

function confirmLogout() {
  userStore.logout()
  router.replace({ name: 'login' })
}
</script>

<template>
  <div class="usermenu-root" @mouseleave="open = false">
    <button class="usermenu-trigger" @click="open = !open" @mouseenter="open = true">
      <span class="usermenu-avatar">{{ userStore.user?.name?.slice(0, 1) || t('common.initial') }}</span>
      <span class="usermenu-uname">{{ userStore.user?.name }}</span>
    </button>
    <transition name="drop">
      <div v-if="open" class="usermenu-menu">
        <div class="px-2.5 py-2">
          <p class="font-semibold m-0">{{ userStore.user?.name }}</p>
          <p class="text-[12px] text-ink-muted mt-0.5">{{ userStore.user?.email }}</p>
        </div>
        <div class="h-px bg-line my-1" />
        <n-popconfirm
          :show-icon="false"
          :positive-text="t('common.confirm')"
          :negative-text="t('common.cancel')"
          @positive-click="confirmLogout"
        >
          <template #trigger>
            <button class="usermenu-item usermenu-danger" @click.stop>
              <n-icon :component="LogOutOutline" /> {{ t('userMenu.logout') }}
            </button>
          </template>
          {{ t('userMenu.logoutConfirm') }}
        </n-popconfirm>
      </div>
    </transition>
  </div>
</template>
