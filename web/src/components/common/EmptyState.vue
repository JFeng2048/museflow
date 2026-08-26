<script setup lang="ts">
defineProps<{ title?: string; description?: string }>()
</script>

<template>
  <div class="flex flex-col items-center justify-center text-center text-ink-muted px-4 py-12">
    <div class="text-[32px] mb-3">✨</div>
    <p v-if="title" class="text-[15px] font-semibold text-ink-soft m-0 mb-1">{{ title }}</p>
    <p v-if="description" class="text-[13px] m-0">{{ description }}</p>
    <slot />
  </div>
</template>

