<script setup lang="ts">
/**
 * App.vue — 整棵应用外壳。
 *
 * 单一色源策略：
 *   - 业务 CSS（Tailwind / 自定义组件）从 `data-theme` 切换的 CSS 变量取色；
 *   - Naive UI 组件也必须跟当前主题走，因此本文件用一个响应式的
 *     `themeOverrides` 把整组 `--c-*` 喂给 `<n-config-provider>`。
 *
 * 关键细节：`themeOverrides` 是 `computed`，依赖 `ui.themeId`；
 * 主题切换时 `setAttribute('data-theme')` 后，CSS 变量在下一帧生效，
 * 因此监听里用 `requestAnimationFrame` 等浏览器 paint 完再读一次。
 */
import { ref, watch } from 'vue'
import { NConfigProvider, NMessageProvider, NDialogProvider } from 'naive-ui'
import { useUiStore } from '@/stores/ui'

const ui = useUiStore()

/* ----------------------------------------------------------- *
 * 读 `:root` 上由 CSS 当前主题声明的 CSS 变量。
 * SSR 环境下没有 `document`，回落到 fallback。
 * ----------------------------------------------------------- */
function cssVar(name: string, fallback = ''): string {
  if (typeof window === 'undefined') return fallback
  const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  return v || fallback
}

/**
 * 收集主题的所有相关 token 并生成 Naive UI themeOverrides。
 * 不在调用栈内做缓存，每次创建新对象以让 Naive UI 内部 watcher 触发刷新。
 */
function buildOverrides() {
  // 基础语义色（来自 :root 与各 [data-theme] 选择器）
  const ink = cssVar('--c-ink')
  const inkSoft = cssVar('--c-ink-soft')
  const inkMuted = cssVar('--c-ink-muted')
  const amberDeep = cssVar('--c-amber-deep')
  const amber = cssVar('--c-amber')
  const amberSoft = cssVar('--c-amber-soft')
  const warm = cssVar('--c-warm')
  const warm2 = cssVar('--c-warm-2')
  const paper = cssVar('--c-paper')
  const line = cssVar('--c-line')
  const lineSoft = cssVar('--c-line-soft')
  const success = cssVar('--c-success')
  const warn = cssVar('--c-warn')
  const danger = cssVar('--c-danger')
  const info = cssVar('--c-info')

  return {
    common: {
      /* 主题色：琥珀/护眼绿/灰 等各主题的「amber*」 */
      primaryColor: amberDeep,
      primaryColorHover: amber,
      primaryColorPressed: amberDeep,
      primaryColorSuppl: amber,

      /* 状态色 */
      successColor: success,
      warningColor: warn,
      errorColor: danger,
      infoColor: info,

      /* 字体 / 文本 */
      fontSize: '14px',
      fontSizeMedium: '14px',
      fontSizeLarge: '15px',
      textColorBase: ink,
      textColor1: ink,
      textColor2: inkSoft,
      textColor3: inkMuted,
      textColorDisabled: inkMuted,
      placeholderColor: inkMuted,
      placeholderColorDisabled: inkMuted,
      iconColor: inkSoft,
      iconColorHover: ink,

      /* 容器表面 —— 这一组是「主题切换在组件层不响应」的根因，必须每主题重喂 */
      bodyColor: warm,
      cardColor: paper,
      modalColor: paper,
      popoverColor: paper,
      tableColor: paper,
      tableColorHover: warm2,
      tableColorStriped: warm2,
      inputColor: paper,
      inputColorDisabled: warm2,
      buttonColor2: paper,
      buttonColor2Hover: warm2,
      buttonColor2Pressed: warm2,

      /* 边框 / 分割线 */
      borderColor: line,
      dividerColor: lineSoft,

      /* 中性色 —— NTag default / NSwitch 关闭轨道 / 装饰 hover 都用它 */
      neutralColor: warm2,
      neutralColorHover: line,
      neutralColorPressed: lineSoft,
      railColor: warm2,

      /* 交互色 */
      hoverColor: warm2,
      pressedColor: lineSoft,

      /* 用一套统一圆角（来自 --radius-m） */
      borderRadius: '10px',
      borderRadiusSmall: '8px',
    },
    Card: { borderRadius: '14px', color: paper, colorEmbedded: warm2, borderColor: line },
    Tag: { borderRadius: '999px' },
    DataTable: {
      thColor: warm2,
      thColorHover: warm2,
      tdColorHover: warm2,
      tdColorStriped: warm2,
      borderColor: line,
      borderRadius: '10px',
    },
    Input: {
      borderRadius: '8px',
      color: paper,
      colorFocus: paper,
      placeholderColor: inkMuted,
    },
    Select: { peers: { InternalSelection: { borderRadius: '8px', color: paper } } },
    Button: { borderRadiusMedium: '10px', borderRadiusSmall: '8px', borderRadiusLarge: '12px' },
    Layout: { color: warm },
    Menu: { itemColorHover: warm2, itemColorActive: warm2, color: warm },
    Dropdown: { color: paper, borderColor: line, borderRadius: '10px' },
    Modal: { color: paper },
    Tooltip: { color: paper, textColorBase: ink, borderRadius: '8px' },
    Notification: { color: paper, borderRadius: '10px' },
    Message: { color: paper, borderRadius: '10px' },
    Tabs: { tabFontWeightActive: '600', barColor: amberDeep },
  } as const
}

/* ----------------------------------------------------------- *
 * 响应式：跟 `ui.themeId` 联动。
 *
 * 注意：Vue 的 `computed` 是只读的；如果想用 computed 必须读
 * 闭包里的响应式依赖。这里为了在 watch 里能强制刷新（避免
 * computed 缓存错过 frame 读到的旧值），改用 `ref + watch`。
 * ----------------------------------------------------------- */
const themeOverrides = ref(buildOverrides())

function refreshOverrides() {
  themeOverrides.value = buildOverrides()
}

/**
 * 主题切换流程：
 *   1) 主题 store 调 `applyTheme` → setAttribute('data-theme', id) + class
 *   2) CSS 变量在下一帧重解析，浏览器 paint
 *   3) 我们等 paint 完成后，再读一次变量，把新值灌给 Naive UI
 */
let rafId = 0
watch(
  () => ui.themeId,
  () => {
    cancelAnimationFrame(rafId)
    rafId = requestAnimationFrame(refreshOverrides)
  },
  { immediate: false },
)
</script>

<template>
  <n-config-provider :theme-overrides="themeOverrides">
    <n-message-provider :max="3">
      <n-dialog-provider>
        <router-view />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>
