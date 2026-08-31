<script setup lang="ts">
import { ref, inject } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { NIcon } from 'naive-ui'
import { MailOutline, LogoGithub, LogoWechat } from '@vicons/ionicons5'
import { useUserStore } from '@/stores/system/user'
import { useUiStore } from '@/stores/ui'
import {
  login as loginApi,
  loginWithCode,
  verifyMfaLogin,
  sendCode,
} from '@/api/system/auth'
import type { AuthResult } from '@/types/system/auth'
import IdentityPicker from '@/components/common/IdentityPicker.vue'
import TurnstileWidget from '@/components/auth/TurnstileWidget.vue'
import type { TurnstileWidgetExposed } from '@/components/auth/TurnstileWidget.vue'

const { t } = useI18n()
const router = useRouter()
const message = useMessage()
const userStore = useUserStore()
const ui = useUiStore()

// 由 AuthLayout 提供的模式切换（翻转用，不重挂路由）
const setAuthMode = inject<(m: 'login' | 'register') => void>('setAuthMode')

const tab = ref<'password' | 'code'>('password')
const email = ref('')
const password = ref('')
const code = ref('')
const loading = ref(false)

// 管理员登录后，先选择进入哪个视图。
const showIdentity = ref(false)

// 2FA 二次验证中间态
const showMfa = ref(false)
const mfaTicket = ref('')
const mfaCode = ref('')
const mfaLoading = ref(false)
const useRecovery = ref(false)

async function finishLogin(result: AuthResult) {
  userStore.setAuth(result.token, result.user)
  message.success(t('auth.loginSuccess'))
  if (result.user.role === 'admin') {
    showIdentity.value = true
  } else {
    userStore.enterUser()
    router.replace('/novels')
  }
}

async function onSubmit() {
  if (!email.value || (tab.value === 'password' && !password.value) || (tab.value === 'code' && !code.value)) {
    message.warning(t('auth.fillBoth'))
    return
  }
  loading.value = true
  try {
    const result =
      tab.value === 'password'
        ? await loginApi({ username: email.value, password: password.value })
        : await loginWithCode({ email: email.value, code: code.value })
    if (result.requiresMfa && result.mfaTicket) {
      mfaTicket.value = result.mfaTicket
      showMfa.value = true
      return
    }
    await finishLogin(result)
  } catch {
    message.error(t('auth.loginFailed'))
  } finally {
    loading.value = false
  }
}

async function onSubmitMfa() {
  if (!mfaCode.value) {
    message.warning(t('auth.fillBoth'))
    return
  }
  mfaLoading.value = true
  try {
    const result = await verifyMfaLogin(mfaTicket.value, mfaCode.value)
    showMfa.value = false
    await finishLogin(result)
  } catch {
    message.error(t('auth.mfaInvalid'))
  } finally {
    mfaLoading.value = false
  }
}

function chooseIdentity(view: 'user' | 'admin') {
  showIdentity.value = false
  if (view === 'admin') {
    userStore.enterAdmin()
    router.replace('/admin')
  } else {
    userStore.enterUser()
    router.replace('/novels')
  }
}

const countDown = ref(0)
const tsRef = ref<TurnstileWidgetExposed | null>(null)
let timer: ReturnType<typeof setInterval> | undefined
async function onSendCode(): Promise<boolean> {
  if (loading.value) return false
  if (!email.value) {
    message.warning(t('auth.fillBoth'))
    return false
  }
  loading.value = true
  // 发送验证码前完成人机验证（mock 环境降级跳过）
  let tsToken: string | null = null
  try {
    tsToken = await tsRef.value?.ensureToken() ?? null
  } catch {
    message.warning(t('auth.turnstileRequired'))
    loading.value = false
    return false
  }
  try {
    await sendCode({ email: email.value, scene: 'login', turnstileToken: tsToken ?? undefined })
  } catch {
    tsRef.value?.reset()
    loading.value = false
    return false
  }
  tsRef.value?.reset() // Turnstile 令牌一次性，成功也要重置
  loading.value = false
  // 验证码由服务端下发到邮箱，前端不再预填
  message.success(t('auth.codeSent'))
  countDown.value = 60
  timer = setInterval(() => {
    countDown.value--
    if (countDown.value <= 0 && timer) clearInterval(timer)
  }, 1000)
  return true
}

  function onWechat() {
  message.info(t('auth.socialDevTip'))
  }
  function onGithub() {
  message.info(t('auth.socialDevTip'))
  }
  </script>

<template>
  <!-- 卡片本体（被 AuthLayout 的 .auth-flip-wrap 包住，rotateY 应用在 wrap 上） -->
  <div class="auth-form-card">
    <h2>{{ t('auth.loginTitle') }}</h2>
    <p class="auth-sub">{{ t('auth.loginSub') }}</p>

    <form @submit.prevent="onSubmit">
      <label class="auth-field">
        <span class="auth-lbl"><n-icon :component="MailOutline" class="text-[14px]" /> {{ t('auth.email') }}</span>
        <input v-model="email" type="email" :placeholder="t('auth.emailPlaceholder')" autocomplete="email" />
      </label>

      <!-- 登录方式切换：内联到字段上方，作为「小分段控件」。
           选中态用底部细线表达，避免任何会跟主按钮抢视觉权的样式。 -->
      <div class="auth-method" role="tablist">
        <button
          type="button"
          role="tab"
          class="auth-method-tab"
          :class="{ active: tab === 'password' }"
          :aria-selected="tab === 'password'"
          @click="tab = 'password'"
        >
          {{ t('auth.password') }}
        </button>
        <button
          type="button"
          role="tab"
          class="auth-method-tab"
          :class="{ active: tab === 'code' }"
          :aria-selected="tab === 'code'"
          @click="tab = 'code'"
        >
          {{ t('auth.codeLogin') }}
        </button>
      </div>

      <!-- 字段本身不需要再重复「密码 / 验证码」标签：
           上方的分段控件已经告诉用户当前在哪种登录方式，这里只留 placeholder。 -->
      <label v-if="tab === 'password'" class="auth-field auth-field-tight">
        <input v-model="password" type="password" :placeholder="t('auth.passwordPlaceholder')" autocomplete="current-password" />
      </label>

      <label v-else class="auth-field auth-field-tight">
        <div class="auth-code-row">
          <input v-model="code" :placeholder="t('auth.codePlaceholder')" autocomplete="one-time-code" />
          <button class="auth-code-btn" type="button" :disabled="loading || countDown > 0" @click="onSendCode">
            {{ countDown > 0 ? t('auth.resendIn', { s: countDown }) : t('auth.sendCode') }}
          </button>
        </div>
        <!-- 人机验证：点击「发送验证码」时才弹出校验 -->
        <TurnstileWidget ref="tsRef" action="login" :allow-fallback="ui.mockMode" />
      </label>

      <button class="auth-primary" :disabled="loading" type="submit">
        {{ loading ? t('auth.loggingIn') : t('auth.loginBtn') }}
      </button>
    </form>

    <p class="auth-secondary">
      <span class="auth-secondary-q">{{ t('auth.noAccount') }}</span>
      <button class="auth-secondary-link" type="button" @click="setAuthMode?.('register')">
        {{ t('auth.toRegister') }}
      </button>
    </p>

    <div class="auth-social">
      <span class="auth-social-lbl">{{ t('auth.social') }}</span>
      <button class="auth-social-ico" type="button" :title="t('auth.socialWechat')" @click="onWechat">
        <n-icon :component="LogoWechat" />
      </button>
      <button class="auth-social-ico" type="button" :title="t('auth.socialGithub')" @click="onGithub">
        <n-icon :component="LogoGithub" />
      </button>
    </div>

  </div>

  <!-- 弹窗通过 Teleport 送到 body，避开 .auth-flip-wrap 的 transform 新包含块，
       保证 position: fixed 始终以视口为参照居中。 -->
  <Teleport to="body">
    <IdentityPicker v-model:show="showIdentity" @choose="chooseIdentity" />

    <div v-if="showMfa" class="mfa-mask" @click.self="showMfa = false">
      <div class="mfa-card">
        <h3>{{ t('auth.mfaTitle') }}</h3>
        <p class="mfa-sub">{{ t('auth.mfaSubtitle') }}</p>
        <label v-if="!useRecovery" class="auth-field">
          <span class="auth-lbl">{{ t('auth.mfaCode') }}</span>
          <input v-model="mfaCode" :placeholder="t('auth.codePlaceholder')" autocomplete="one-time-code" />
        </label>
        <label v-else class="auth-field">
          <span class="auth-lbl">{{ t('auth.recoveryCode') }}</span>
          <input v-model="mfaCode" placeholder="XXXXXXXX" />
        </label>
        <button class="auth-link-btn" type="button" @click="useRecovery = !useRecovery">
          {{ useRecovery ? t('auth.mfaCode') : t('auth.useRecovery') }}
        </button>
        <button class="auth-primary" :disabled="mfaLoading" type="button" @click="onSubmitMfa">
          {{ mfaLoading ? t('auth.loggingIn') : t('auth.mfaVerify') }}
        </button>
      </div>
    </div>
  </Teleport>
</template>
