<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { useUiStore } from '@/stores/ui'
import { loadTurnstileScript, isTurnstileAvailable } from '@/composables/useTurnstileScript'

const props = withDefaults(
  defineProps<{
    /** 稳定动作名，用于后端校验 action（1-32 字符）。 */
    action?: string
    /** 不可用时的降级开关：true 表示无脚本也允许通过（开发/mock）。 */
    allowFallback?: boolean
  }>(),
  { action: 'challenge', allowFallback: false },
)

const emit = defineEmits<{ (e: 'update:token', v: string | null): void }>()

const ui = useUiStore()
const el = ref<HTMLDivElement | null>(null)
const token = ref<string | null>(null)
const ready = ref(false)
const error = ref(false)

let widgetId: string | undefined

// 等待用户完成验证的 pending resolver（点发送时若还没过，则挂起）
let pending: ((t: string | null) => void) | null = null

function theme(): 'light' | 'dark' {
  // 暗夜/高级走暗色 widget，其余亮色
  return ui.themeId === 'dark' || ui.themeId === 'premium' ? 'dark' : 'light'
}

function renderWidget() {
  if (!el.value || !window.turnstile) return
  widgetId = window.turnstile.render(el.value, {
    sitekey: '0x4AAAAAAEiEuf6sgPSil8qR',
    action: props.action,
    theme: theme(),
    callback: (t: string) => {
      token.value = t
      emit('update:token', t)
      if (pending) {
        pending(t)
        pending = null
      }
    },
    'expired-callback': () => {
      token.value = null
      emit('update:token', null)
    },
    'error-callback': () => {
      error.value = true
    },
  })
  ready.value = true
}

/** 取一个有效令牌：已通过直接返回；否则等待用户完成（mock/无脚本时降级为 null）。 */
async function ensureToken(): Promise<string | null> {
  if (token.value) return token.value
  if (!isTurnstileAvailable()) {
    // 脚本未加载：mock / 开发环境，按 allowFallback 降级
    return props.allowFallback ? null : Promise.reject(new Error('turnstile_unavailable'))
  }
  // 挂起，等 callback 触发
  return new Promise<string | null>((resolve) => {
    pending = resolve
  })
}

function reset() {
  token.value = null
  emit('update:token', null)
  if (widgetId && window.turnstile) window.turnstile.reset(widgetId)
}

onMounted(async () => {
  try {
    await loadTurnstileScript()
    renderWidget()
  } catch {
    // 加载失败：若允许降级则保持不可用态，由 ensureToken 返回 null
    if (!props.allowFallback) error.value = true
  }
})

watch(
  () => ui.themeId,
  () => {
    // 主题切换时重建 widget 以更新明暗
    if (widgetId && window.turnstile) {
      window.turnstile.remove(widgetId)
      widgetId = undefined
      renderWidget()
    }
  },
)

onBeforeUnmount(() => {
  if (widgetId && window.turnstile) window.turnstile.remove(widgetId)
})

export type TurnstileWidgetExposed = {
  ensureToken: () => Promise<string | null>
  reset: () => void
}

defineExpose<TurnstileWidgetExposed>({ ensureToken, reset })
</script>

<template>
  <div class="ts-wrap">
    <div ref="el" class="cf-turnstile" :data-theme="theme()" :data-action="action"></div>
    <p v-if="error" class="ts-error">
      {{ '人机验证组件加载失败，请刷新页面后重试' }}
    </p>
  </div>
</template>

<style scoped>
.ts-wrap {
  margin: 10px 0 2px;
}
.ts-error {
  margin-top: 6px;
  font-size: 12px;
  color: var(--c-danger, #c0392b);
}
</style>
