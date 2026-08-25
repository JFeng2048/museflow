<script setup lang="ts">
defineProps<{ collapsed?: boolean }>()
</script>

<template>
  <div class="app-logo">
    <span class="mark">M</span>
    <span v-if="!collapsed" class="name">MuseFlow</span>
  </div>
</template>

<style scoped>
.app-logo {
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 600;
  color: var(--mf-text);
}
.mark {
  width: 30px;
  height: 30px;
  border-radius: 8px;
  background: var(--mf-accent);
  color: #fff;
  display: grid;
  place-items: center;
  font-size: 16px;
}
.name {
  font-size: 16px;
  letter-spacing: 0.2px;
}
</style>
