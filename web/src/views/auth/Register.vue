<script setup lang="ts">
import { ref, computed, inject } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { NIcon } from 'naive-ui'
import {
  IdCardOutline,
  MailOutline,
  LockClosedOutline,
} from '@vicons/ionicons5'
import { useUserStore } from '@/stores/system/user'
import { register as registerApi, sendCode, MOCK_CODE } from '@/api/system/auth'

const { t } = useI18n()
const router = useRouter()
const message = useMessage()
const userStore = useUserStore()

// 由 AuthLayout 提供的模式切换（翻转用，不重挂路由）
const setAuthMode = inject<(m: 'login' | 'register') => void>('setAuthMode')

const step = ref<'form' | 'code'>('form')
const name = ref('')
const email = ref('')
const password = ref('')
const confirm = ref('')
const code = ref('')
const loading = ref(false)

const valid = computed(
  () => !!email.value && password.value.length >= 6 && password.value === confirm.value,
)

const countDown = ref(0)
let timer: ReturnType<typeof setInterval> | undefined
async function onSendCode() {
  if (!email.value) {
    message.warning(t('auth.fillBoth'))
    return
  }
  await sendCode({ email: email.value, scene: 'register' })
  message.success(t('auth.codeSent'))
  code.value = MOCK_CODE
  countDown.value = 60
  timer = setInterval(() => {
    countDown.value--
    if (countDown.value <= 0 && timer) clearInterval(timer)
  }, 1000)
}

async function onSubmit() {
  if (step.value === 'form') {
    if (!valid.value) {
      message.warning(password.value !== confirm.value ? t('auth.pwMismatch') : t('auth.pwTooShort'))
      return
    }
    await onSendCode()
    step.value = 'code'
    return
  }
  if (!code.value) {
    message.warning(t('auth.fillBoth'))
    return
  }
  loading.value = true
  try {
    const result = await registerApi({
      name: name.value || email.value.split('@')[0],
      email: email.value,
      password: password.value,
      confirmPassword: confirm.value,
      code: code.value,
    })
    userStore.setAuth(result.token, result.user)
    message.success(t('auth.registerSuccess'))
    router.replace('/novels')
  } catch (e: any) {
    message.error(e?.message || t('auth.registerFailed'))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <!-- 根元素即卡片，翻转过渡作用于此 -->
  <div class="auth-form-card">
    <h2>{{ t('auth.registerTitle') }}</h2>
    <p class="auth-sub">{{ t('auth.registerSub') }}</p>

    <form @submit.prevent="onSubmit">
      <label v-if="step === 'form'" class="auth-field">
        <span class="auth-lbl"><n-icon :component="IdCardOutline" class="text-[14px]" /> {{ t('auth.nickname') }}</span>
        <input v-model="name" type="text" :placeholder="t('auth.nicknamePlaceholder')" />
      </label>
      <label class="auth-field">
        <span class="auth-lbl"><n-icon :component="MailOutline" class="text-[14px]" /> {{ t('auth.email') }}</span>
        <input v-model="email" :disabled="step === 'code'" type="email" :placeholder="t('auth.emailPlaceholder')" autocomplete="email" />
      </label>
      <div v-if="step === 'form'" class="auth-row">
        <label class="auth-field">
          <span class="auth-lbl"><n-icon :component="LockClosedOutline" class="text-[14px]" /> {{ t('auth.password') }}</span>
          <input v-model="password" type="password" :placeholder="t('auth.passwordPlaceholder')" autocomplete="new-password" />
        </label>
        <label class="auth-field">
          <span class="auth-lbl"><n-icon :component="LockClosedOutline" class="text-[14px]" /> {{ t('auth.confirmPassword') }}</span>
          <input v-model="confirm" type="password" :placeholder="t('auth.confirmPlaceholder')" autocomplete="new-password" />
        </label>
      </div>

      <!-- 邮箱验证码：第二步输入并校验 -->
      <label v-if="step === 'code'" class="auth-field">
        <span class="auth-lbl"><n-icon :component="LockClosedOutline" class="text-[14px]" /> {{ t('auth.emailCode') }}</span>
        <div class="auth-code-row">
          <input v-model="code" :placeholder="t('auth.codePlaceholder')" autocomplete="one-time-code" />
          <button class="auth-code-btn" type="button" :disabled="countDown > 0" @click="onSendCode">
            {{ countDown > 0 ? t('auth.resendIn', { s: countDown }) : t('auth.sendCode') }}
          </button>
        </div>
      </label>

      <button class="auth-primary" :disabled="loading || (step === 'form' && !valid)" type="submit">
        {{ loading ? t('auth.registering') : step === 'form' ? t('auth.sendCode') : t('auth.registerBtn') }}
      </button>
      <button v-if="step === 'code'" class="auth-back" type="button" @click="step = 'form'">
        ‹ {{ t('auth.back') }}
      </button>
    </form>

    <p class="auth-secondary">
      <span class="auth-secondary-q">{{ t('auth.hasAccount') }}</span>
      <button class="auth-secondary-link" type="button" @click="setAuthMode?.('login')">
        {{ t('auth.toLogin') }}
      </button>
    </p>
  </div>
</template>
