<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
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
const visible = ref(false) // 仅在点击发送验证码时才显示 widget
const error = ref(false)

let widgetId: string | undefined
let scriptReady = false

// 等待用户完成验证的 pending resolver（点击发送时若还没过，则挂起）
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
      visible.value = false // 验证通过后收起，不打扰后续操作
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
}

/**
 * 取一个有效令牌：
 * - 已有（本次会话内通过过）直接返回；
 * - 否则显示 widget 并等待用户完成（mock/无脚本时按 allowFallback 降级为 null）。
 * 即「点击发送验证码时」才进行人机验证。
 */
async function ensureToken(): Promise<string | null> {
  if (token.value) return token.value
  if (!scriptReady || !isTurnstileAvailable()) {
    return props.allowFallback ? null : Promise.reject(new Error('turnstile_unavailable'))
  }
  // 显示 widget 并等待 callback
  visible.value = true
  await nextTick()
  renderWidget()
  return new Promise<string | null>((resolve) => {
    pending = resolve
  })
}

function reset() {
  token.value = null
  emit('update:token', null)
  visible.value = false
  if (widgetId && window.turnstile) {
    window.turnstile.remove(widgetId)
    widgetId = undefined
  }
}

onMounted(async () => {
  try {
    await loadTurnstileScript()
    scriptReady = true
  } catch {
    if (!props.allowFallback) error.value = true
  }
})

watch(
  () => ui.themeId,
  () => {
    // 主题切换时若 widget 已存在则重建以更新明暗
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
  <div v-show="visible" class="ts-wrap">
    <div ref="el" class="cf-turnstile" :data-theme="theme()" :data-action="action"></div>
    <p class="ts-hint">请完成人机验证后继续发送验证码</p>
    <p v-if="error" class="ts-error">
      {{ '人机验证组件加载失败，请刷新页面后重试' }}
    </p>
  </div>
</template>

<style scoped>
.ts-wrap {
  margin: 12px 0 4px;
  text-align: center;
}
.ts-hint {
  margin-top: 6px;
  font-size: 12px;
  color: var(--c-ink-muted, #6b7280);
}
.ts-error {
  margin-top: 6px;
  font-size: 12px;
  color: var(--c-danger, #c0392b);
}
</style>
