import { createApp } from 'vue'
import { createPinia } from 'pinia'
import naive from 'naive-ui'
import App from './App.vue'
import router from './router'
import i18n from './i18n'
import { useUiStore } from '@/stores/ui'
import { useUserStore } from '@/stores/system/user'
import '@/assets/styles/index.css'
import '@/assets/styles/components.css'

const app = createApp(App)
const pinia = createPinia()
app.use(pinia)
app.use(router)
app.use(naive)
app.use(i18n)

// 初始化主题与语言（读取 localStorage 并应用到 :root）
const ui = useUiStore(pinia)
ui.initTheme()
ui.initLang()

// 启动时拉取用户基本信息（未登录时直接用兜底用户，不会发起请求）
const userStore = useUserStore(pinia)
userStore.loadProfile()

// 全局快捷键：Ctrl/Cmd + Shift + T 循环主题；Ctrl/Cmd + Shift + L 切换语言
window.addEventListener('keydown', (e) => {
  if (!(e.ctrlKey || e.metaKey) || !e.shiftKey) return
  const k = e.key.toLowerCase()
  if (k === 't') {
    e.preventDefault()
    ui.cycleTheme()
  } else if (k === 'l') {
    e.preventDefault()
    ui.toggleLang()
  }
})

app.directive('focus', {
  mounted(el) {
    el.focus()
  },
})

app.mount('#app')
