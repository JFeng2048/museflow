<script setup lang="ts">
// 登录 / 注册共用同一套左右分栏外壳。左右两栏各自保持挂载，
// 仅内部内容随 mode（login | register）做 3D 翻转过渡：
//   - 左侧品牌区：登录文案 ⇄ 注册文案 各自翻转
//   - 右侧表单区：登录卡片 ⇄ 注册卡片 各自翻转
// 切换走本地 mode（同步更新 URL，但不重新挂载路由组件），因此没有整页重挂的漂移。
import { ref, computed, provide, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  CreateOutline,
  HomeOutline,
  PeopleOutline,
  FlashOutline,
} from '@vicons/ionicons5'
import LoginCard from '@/views/auth/Login.vue'
import RegisterCard from '@/views/auth/Register.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

type AuthMode = 'login' | 'register'
const mode = ref<AuthMode>(route.name === 'register' ? 'register' : 'login')

// 翻转方向：进入注册向右翻（next），回到登录向左翻（prev），左右同步方向。
const flipName = computed(() => (mode.value === 'register' ? 'auth-flip-next' : 'auth-flip-prev'))

// 暴露给子卡片：当前模式 + 切换（同时同步 URL）
function setMode(next: AuthMode) {
  if (next === mode.value) return
  mode.value = next
  router.replace({ name: next })
}
provide('authMode', mode)
provide('setAuthMode', setMode)

// 直接通过 URL 进入时，同步 mode（不影响正在进行的翻转方向感）
watch(
  () => route.name,
  (name) => {
    mode.value = name === 'register' ? 'register' : 'login'
  },
)
</script>

<template>
  <div class="auth-page">
    <!-- 左侧品牌区：登录 / 注册两套文案各自翻转 -->
    <aside class="auth-brand">
      <Transition :name="flipName" mode="out-in">
        <div v-if="mode === 'login'" key="brand-login" class="auth-brand-inner">
          <div class="auth-logo">
            <n-icon :component="CreateOutline" class="text-[26px]" />
            <span>MuseFlow</span>
          </div>
          <h1 class="auth-slogan">{{ t('auth.brandSlogan') }}</h1>
          <p class="auth-pitch">{{ t('auth.brandPitch') }}</p>
          <ul class="auth-promises">
            <li><n-icon :component="HomeOutline" class="text-[16px]" /> {{ t('auth.promise1') }}</li>
            <li><n-icon :component="PeopleOutline" class="text-[16px]" /> {{ t('auth.promise2') }}</li>
            <li><n-icon :component="FlashOutline" class="text-[16px]" /> {{ t('auth.promise3') }}</li>
          </ul>
          <p class="auth-whisper">{{ t('auth.brandWhisper') }}</p>
        </div>
        <div v-else key="brand-register" class="auth-brand-inner">
          <div class="auth-logo">
            <n-icon :component="CreateOutline" class="text-[26px]" />
            <span>MuseFlow</span>
          </div>
          <h1 class="auth-slogan">{{ t('auth.registerSlogan') }}</h1>
          <p class="auth-pitch">{{ t('auth.registerPitch') }}</p>
          <ol class="auth-steps">
            <li><b>①</b> {{ t('auth.step1') }}</li>
            <li><b>②</b> {{ t('auth.step2') }}</li>
            <li><b>③</b> {{ t('auth.step3') }}</li>
          </ol>
          <p class="auth-whisper">{{ t('auth.registerWhisper') }}</p>
        </div>
      </Transition>
    </aside>

    <!-- 右侧表单区：登录 / 注册两张卡片各自翻转。
         关键：用 .auth-flip-wrap 作为 Transition 的唯一单根子节点，
         让 rotateY 作用在外层容器上，内部的 <Teleport> 弹窗（position: fixed）
         通过被传送到 body 而避开 transform 的新包含块，保持视口居中。 -->
    <main class="auth-form-side">
      <Transition :name="flipName" mode="out-in">
        <div v-if="mode === 'login'" key="login" class="auth-flip-wrap">
          <LoginCard />
        </div>
        <div v-else key="register" class="auth-flip-wrap">
          <RegisterCard />
        </div>
      </Transition>
      <p class="auth-foot">{{ t('auth.agree') }}</p>
    </main>
  </div>
</template>
