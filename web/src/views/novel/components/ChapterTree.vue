<script setup lang="ts">
import { formatWords } from '@/utils/format'
import type { Chapter } from '@/types'

defineProps<{ chapters: Chapter[]; activeId?: string }>()
const emit = defineEmits<{ (e: 'select', chapter: Chapter): void }>()
</script>

<template>
  <div class="chapter-tree">
    <div
      v-for="ch in chapters"
      :key="ch.id"
      class="node"
      :class="{ active: ch.id === activeId }"
      @click="emit('select', ch)"
    >
      <div class="row">
        <span class="title">{{ ch.title }}</span>
        <span class="words">{{ formatWords(ch.words) }}</span>
      </div>
      <ChapterTree
        v-if="ch.children && ch.children.length"
        :chapters="ch.children"
        :active-id="activeId"
        @select="(c) => emit('select', c)"
      />
    </div>
  </div>
</template>
