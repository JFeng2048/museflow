<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    name: string
    color?: string
    size?: number
    /** 当 user.avatar 存在时优先显示 */
    avatar?: string
  }>(),
  { size: 34 },
)

// 传入纯色时为圆形缩写头像；传入 avatar 字符串时：
//  - URL / dataURL（包含 . 或 :// 开头）渲染为图片；
//  - 否则视为 emoji / 字符。
const isImage = computed(() => {
  if (!props.avatar) return false
  return /^data:|^https?:\/\//.test(props.avatar)
})
const dim = computed(() => `${props.size}px`)
const fontPx = computed(() => `${Math.round((props.size || 34) * 0.42)}px`)
</script>

<template>
  <div
    class="user-avatar"
    :class="{ 'is-image': isImage, 'is-emoji': !isImage && !!avatar }"
    :style="{
      width: dim,
      height: dim,
      background: isImage ? 'transparent' : (color || '#5B8DEF'),
      fontSize: isImage ? undefined : (avatar ? Math.round((size || 34) * 0.5) + 'px' : fontPx),
    }"
  >
    <img v-if="isImage" :src="avatar" :alt="name" class="user-avatar-img" />
    <span v-else-if="avatar">{{ avatar }}</span>
    <span v-else>{{ name.slice(0, 1) }}</span>
  </div>
</template>
