<script setup lang="ts">
import { computed } from 'vue'

/**
 * 品牌印记：顶部栏左侧的方形标识。
 *
 * 早期这里是空占位，窄屏下品牌名被隐藏后顶部栏左侧就什么都不剩；
 * 尺寸由 size 驱动，配色跟随主题变量，深浅色主题都不刺眼。
 */
const props = withDefaults(defineProps<{ size?: number; collapsed?: boolean }>(), {
  size: 30,
})

const box = computed(() => ({
  width: `${props.size}px`,
  height: `${props.size}px`,
  fontSize: `${Math.round(props.size * 0.56)}px`,
}))
</script>

<template>
  <div class="app-logo" :style="box" aria-hidden="true">M</div>
</template>
