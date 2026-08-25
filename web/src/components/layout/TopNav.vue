<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { NIcon } from 'naive-ui'
import { MoonOutline, SunnyOutline, LogOutOutline } from '@vicons/ionicons5'
import UserAvatar from '@/components/common/UserAvatar.vue'
import { useUserStore } from '@/stores/user'
import { useTheme } from '@/composables/useTheme'

const userStore = useUserStore()
const router = useRouter()
const { isDark, toggle } = useTheme()

const nickname = computed(() => userStore.profile?.nickname || '写作者')

function logout() {
  userStore.logout()
  router.push({ name: 'login' })
}
</script>

<template>
  <header class="topnav">
    <div class="left">
      <n-input round placeholder="搜索项目、素材、设定…" style="max-width: 280px">
        <template #prefix>
          <span class="search-icon">🔍</span>
        </template>
      </n-input>
    </div>
    <div class="right">
      <n-button quaternary circle @click="toggle()">
        <n-icon :component="isDark ? SunnyOutline : MoonOutline" />
      </n-button>
      <div class="user">
        <UserAvatar :name="nickname" :size="32" />
        <span class="nick">{{ nickname }}</span>
      </div>
      <n-button quaternary circle @click="logout">
        <n-icon :component="LogOutOutline" />
      </n-button>
    </div>
  </header>
</template>

<style scoped>
.topnav {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  border-bottom: 1px solid var(--mf-border);
  background: var(--mf-surface);
}
.right {
  display: flex;
  align-items: center;
  gap: 8px;
}
.user {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 6px;
}
.nick {
  font-size: 14px;
  color: var(--mf-text);
}
.search-icon {
  font-size: 13px;
  opacity: 0.6;
}
</style>
