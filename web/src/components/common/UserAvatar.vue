<script setup lang="ts">
defineProps<{ name: string; color?: string; size?: number }>()
</script>

<template>
  <div
    class="user-avatar"
    :style="{
      background: color || '#5B8DEF',
      width: (size || 34) + 'px',
      height: (size || 34) + 'px',
      fontSize: ((size || 34) * 0.42) + 'px',
    }"
  >
    {{ name.slice(0, 1) }}
  </div>
</template>

<style scoped>
.user-avatar {
  border-radius: 50%;
  color: #fff;
  display: grid;
  place-items: center;
  font-weight: 600;
  flex: none;
}
</style>
