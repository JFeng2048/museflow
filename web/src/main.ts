import { createApp } from 'vue'
import { createPinia } from 'pinia'
import naive from 'naive-ui'
import App from './App.vue'
import router from './router'
import { useUserStore } from '@/stores/user'
import './assets/styles/index.css'

const app = createApp(App)
const pinia = createPinia()
app.use(pinia)
app.use(router)
app.use(naive)

// 恢复登录态后再挂载，避免首屏闪烁到登录页再跳回。
const userStore = useUserStore()
await userStore.restore()

app.mount('#app')
