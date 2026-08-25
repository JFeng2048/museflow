<script setup lang="ts">
defineProps<{ title?: string; description?: string }>()
</script>

<template>
  <div class="empty-state">
    <div class="icon">✨</div>
    <p v-if="title" class="title">{{ title }}</p>
    <p v-if="description" class="desc">{{ description }}</p>
    <slot />
  </div>
</template>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 16px;
  color: var(--mf-text-3);
  text-align: center;
}
.icon {
  font-size: 32px;
  margin-bottom: 12px;
}
.title {
  font-size: 15px;
  font-weight: 600;
  color: var(--mf-text-2);
  margin: 0 0 4px;
}
.desc {
  font-size: 13px;
  margin: 0;
}
</style>
