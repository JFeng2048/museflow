<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { NIcon } from 'naive-ui'
import {
  CreateOutline,
  IdCardOutline,
  MailOutline,
  LockClosedOutline,
} from '@vicons/ionicons5'
import { useUserStore } from '@/stores/system/user'
import { register as registerApi } from '@/api/system/auth'

const { t } = useI18n()
const router = useRouter()
const message = useMessage()
const userStore = useUserStore()

const name = ref('')
const email = ref('')
const password = ref('')
const confirm = ref('')
const loading = ref(false)

const valid = computed(
  () => !!email.value && password.value.length >= 6 && password.value === confirm.value,
)

async function onSubmit() {
  if (!valid.value) {
    message.warning(password.value !== confirm.value ? t('auth.pwMismatch') : t('auth.pwTooShort'))
    return
  }
  loading.value = true
  try {
    const result = await registerApi({
      name: name.value || email.value.split('@')[0],
      email: email.value,
      password: password.value,
      confirmPassword: confirm.value,
    })
    userStore.setAuth(result.token, result.user)
    message.success(t('auth.registerSuccess'))
    router.replace('/novels')
  } catch {
    message.error(t('auth.registerFailed'))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <aside class="auth-brand">
      <div class="auth-brand-inner">
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
    </aside>

    <main class="auth-form-side">
      <div class="auth-form-card">
        <h2>{{ t('auth.registerTitle') }}</h2>
        <p class="auth-sub">{{ t('auth.registerSub') }}</p>

        <form @submit.prevent="onSubmit">
          <label class="auth-field">
            <span class="auth-lbl"><n-icon :component="IdCardOutline" class="text-[14px]" /> {{ t('auth.nickname') }}</span>
            <input v-model="name" type="text" :placeholder="t('auth.nicknamePlaceholder')" />
          </label>
          <label class="auth-field">
            <span class="auth-lbl"><n-icon :component="MailOutline" class="text-[14px]" /> {{ t('auth.email') }}</span>
            <input v-model="email" type="email" :placeholder="t('auth.emailPlaceholder')" autocomplete="email" />
          </label>
          <div class="auth-row">
            <label class="auth-field">
              <span class="auth-lbl"><n-icon :component="LockClosedOutline" class="text-[14px]" /> {{ t('auth.password') }}</span>
              <input v-model="password" type="password" :placeholder="t('auth.passwordPlaceholder')" autocomplete="new-password" />
            </label>
            <label class="auth-field">
              <span class="auth-lbl"><n-icon :component="LockClosedOutline" class="text-[14px]" /> {{ t('auth.confirmPassword') }}</span>
              <input v-model="confirm" type="password" :placeholder="t('auth.confirmPlaceholder')" autocomplete="new-password" />
            </label>
          </div>

          <button class="auth-primary" :disabled="loading || !valid" type="submit">
            {{ loading ? t('auth.registering') : t('auth.registerBtn') }}
          </button>
        </form>

        <p class="auth-switch">
          {{ t('auth.hasAccount') }}
          <RouterLink to="/login">{{ t('auth.toLogin') }}</RouterLink>
        </p>
      </div>
      <p class="auth-foot">{{ t('auth.agree') }}</p>
    </main>
  </div>
</template>
