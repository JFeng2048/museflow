<script setup lang="ts">
import { ref } from 'vue'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { NIcon } from 'naive-ui'
import { LockClosedOutline } from '@vicons/ionicons5'
import { changePassword } from '@/api/system/auth'

const { t } = useI18n()
const message = useMessage()

const oldPwd = ref('')
const newPwd = ref('')
const confirmPwd = ref('')
const loading = ref(false)

async function onSubmit() {
  if (!oldPwd.value || !newPwd.value || !confirmPwd.value) {
    message.warning(t('auth.fillBoth'))
    return
  }
  if (newPwd.value.length < 6) {
    message.warning(t('auth.pwTooShort'))
    return
  }
  if (newPwd.value !== confirmPwd.value) {
    message.warning(t('auth.pwMismatch'))
    return
  }
  loading.value = true
  try {
    await changePassword({ oldPassword: oldPwd.value, newPassword: newPwd.value })
    message.success(t('settings.password.changed'))
    oldPwd.value = newPwd.value = confirmPwd.value = ''
  } catch {
    message.error(t('settings.password.failed'))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="pw-panel">
    <div class="pw-head">
      <span class="pw-ico"><n-icon :component="LockClosedOutline" /></span>
      <div>
        <h3>{{ t('settings.password.title') }}</h3>
        <p>{{ t('settings.password.desc') }}</p>
      </div>
    </div>

    <form class="pw-form" @submit.prevent="onSubmit">
      <label class="auth-field">
        <span class="auth-lbl">{{ t('settings.password.old') }}</span>
        <input v-model="oldPwd" type="password" :placeholder="t('settings.password.oldPlaceholder')" autocomplete="current-password" />
      </label>
      <label class="auth-field">
        <span class="auth-lbl">{{ t('settings.password.new') }}</span>
        <input v-model="newPwd" type="password" :placeholder="t('settings.password.newPlaceholder')" autocomplete="new-password" />
      </label>
      <label class="auth-field">
        <span class="auth-lbl">{{ t('settings.password.confirm') }}</span>
        <input v-model="confirmPwd" type="password" :placeholder="t('settings.password.confirmPlaceholder')" autocomplete="new-password" />
      </label>
      <button class="auth-primary pw-submit" :disabled="loading" type="submit">
        {{ loading ? t('auth.loggingIn') : t('settings.password.submit') }}
      </button>
    </form>
  </div>
</template>
