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

<style scoped>
.chapter-tree {
  padding-left: 4px;
}
.node {
  padding: 4px 0;
}
.node .node {
  padding-left: 16px;
  border-left: 1px solid var(--mf-border);
  margin-left: 4px;
}
.row {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 10px;
  border-radius: 8px;
  cursor: pointer;
  font-size: 13px;
  color: var(--mf-text-2);
}
.row:hover {
  background: var(--mf-hover);
}
.node.active > .row {
  background: var(--mf-active-bg);
  color: var(--mf-active-text);
  font-weight: 600;
}
.words {
  color: var(--mf-text-3);
  font-size: 12px;
}
</style>
