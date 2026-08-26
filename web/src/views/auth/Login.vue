<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { NIcon } from 'naive-ui'
import {
  CreateOutline,
  HomeOutline,
  PeopleOutline,
  FlashOutline,
  MailOutline,
  LockClosedOutline,
} from '@vicons/ionicons5'
import { useUserStore } from '@/stores/system/user'
import { login as loginApi } from '@/api/system/auth'

const { t } = useI18n()
const router = useRouter()
const message = useMessage()
const userStore = useUserStore()

const email = ref('zhiqiu@museflow.app')
const password = ref('demo1234')
const loading = ref(false)

async function onSubmit() {
  if (!email.value || !password.value) {
    message.warning(t('auth.fillBoth'))
    return
  }
  loading.value = true
  try {
    const result = await loginApi({ username: email.value, password: password.value })
    userStore.setAuth(result.token, result.user)
    message.success(t('auth.loginSuccess'))
    router.replace('/novels')
  } catch {
    message.error(t('auth.loginFailed'))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <!-- 左侧品牌区 -->
    <aside class="auth-brand">
      <div class="auth-brand-inner">
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
    </aside>

    <!-- 右侧表单 -->
    <main class="auth-form-side">
      <div class="auth-form-card">
        <h2>{{ t('auth.loginTitle') }}</h2>
        <p class="auth-sub">{{ t('auth.loginSub') }}</p>

        <form @submit.prevent="onSubmit">
          <label class="auth-field">
            <span class="auth-lbl"><n-icon :component="MailOutline" class="text-[14px]" /> {{ t('auth.email') }}</span>
            <input v-model="email" type="email" :placeholder="t('auth.emailPlaceholder')" autocomplete="email" />
          </label>
          <label class="auth-field">
            <span class="auth-lbl"><n-icon :component="LockClosedOutline" class="text-[14px]" /> {{ t('auth.password') }}</span>
            <input v-model="password" type="password" :placeholder="t('auth.passwordPlaceholder')" autocomplete="current-password" />
          </label>

          <button class="auth-primary" :disabled="loading" type="submit">
            {{ loading ? t('auth.loggingIn') : t('auth.loginBtn') }}
          </button>
        </form>

        <p class="auth-switch">
          {{ t('auth.noAccount') }}
          <RouterLink to="/register">{{ t('auth.toRegister') }}</RouterLink>
        </p>

        <div class="auth-social">
          <span>{{ t('auth.social') }}</span>
          <button class="auth-ghost" type="button">{{ t('auth.socialWechat') }}</button>
          <button class="auth-ghost" type="button">{{ t('auth.socialGithub') }}</button>
        </div>
      </div>
      <p class="auth-foot">{{ t('auth.agree') }}</p>
    </main>
  </div>
</template>
