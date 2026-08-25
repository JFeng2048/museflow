<script setup lang="ts">
import { ref } from 'vue'
import Sidebar from '@/components/layout/Sidebar.vue'
import TopNav from '@/components/layout/TopNav.vue'

const collapsed = ref(false)
</script>

<template>
  <div class="default-layout">
    <Sidebar :collapsed="collapsed" @toggle="collapsed = !collapsed" />
    <div class="main">
      <TopNav />
      <main class="content">
        <router-view />
      </main>
    </div>
  </div>
</template>

<style scoped>
.default-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
}
.main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.content {
  flex: 1;
  overflow-y: auto;
  padding: 24px 28px;
  background: var(--mf-bg);
}
</style>
