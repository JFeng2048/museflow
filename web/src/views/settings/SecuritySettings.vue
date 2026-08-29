<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { NIcon, NSwitch, NButton, NModal, NTag, NSpace, NInput } from 'naive-ui'
import {
  ShieldCheckmarkOutline,
  ShieldOutline,
  KeyOutline,
  HardwareChipOutline,
  ReloadOutline,
} from '@vicons/ionicons5'
import {
  setupMfa,
  verifyMfa,
  disableMfa,
  getMfaStatus,
  getRecoveryCodes,
  listSessions,
  revokeSession,
} from '@/api/system/auth'
import type { MFAStatus, SessionInfo } from '@/types/system/auth'

const { t } = useI18n()
const message = useMessage()

const status = ref<MFAStatus>({ enabled: false, remainingRecoveryCodes: 0 })
const sessions = ref<SessionInfo[]>([])

// 开启 2FA 流程
const showSetup = ref(false)
const otpauthUrl = ref('')
const secret = ref('')
const setupCode = ref('')
const recoveryCodes = ref<string[]>([])
const setupLoading = ref(false)

async function load() {
  status.value = await getMfaStatus()
  sessions.value = await listSessions()
}

onMounted(load)

async function startSetup() {
  const setup = await setupMfa()
  otpauthUrl.value = setup.otpauthUrl
  secret.value = setup.secret
  setupCode.value = ''
  recoveryCodes.value = []
  showSetup.value = true
}

async function confirmSetup() {
  if (!setupCode.value) {
    message.warning(t('auth.fillBoth'))
    return
  }
  setupLoading.value = true
  try {
    const res = await verifyMfa(setupCode.value)
    recoveryCodes.value = res.recoveryCodes
    status.value = { enabled: true, remainingRecoveryCodes: res.recoveryCodes.length }
    message.success(t('security.mfaEnabled'))
  } catch {
    message.error(t('auth.mfaInvalid'))
  } finally {
    setupLoading.value = false
  }
}

async function onDisable(code: string) {
  try {
    await disableMfa(code || '123456')
    status.value = { enabled: false, remainingRecoveryCodes: 0 }
    message.success(t('security.mfaDisabled'))
  } catch {
    message.error(t('auth.mfaInvalid'))
  }
}

async function refreshSessions() {
  sessions.value = await listSessions()
  message.success(t('security.sessionsRefreshed'))
}

async function onRevoke(tokenId: string) {
  await revokeSession(tokenId)
  sessions.value = sessions.value.filter((s) => s.tokenId !== tokenId)
  message.success(t('security.sessionRevoked'))
}
</script>

<template>
  <div class="security-panel">
    <!-- 2FA 卡片 -->
    <div class="security-card">
      <div class="security-head">
        <span class="security-ico" :class="{ on: status.enabled }">
          <n-icon :component="status.enabled ? ShieldCheckmarkOutline : ShieldOutline" />
        </span>
        <div>
          <h3>{{ t('security.mfaTitle') }}</h3>
          <p>{{ status.enabled ? t('security.mfaOnDesc') : t('security.mfaOffDesc') }}</p>
        </div>
        <n-switch
          v-if="!status.enabled"
          :value="false"
          @update:value="startSetup"
          class="security-switch"
        />
        <n-tag v-else type="success" :bordered="false" size="small">{{ t('security.enabled') }}</n-tag>
      </div>

      <div v-if="status.enabled" class="security-actions">
        <n-button size="small" tertiary @click="recoveryCodes = []; getRecoveryCodes().then(r => recoveryCodes = r)">
          {{ t('security.viewRecovery') }}
        </n-button>
        <n-popconfirm @positive-click="onDisable('')">
          <template #trigger>
            <n-button size="small" type="error" tertiary>{{ t('security.disable') }}</n-button>
          </template>
          {{ t('security.disableConfirm') }}
        </n-popconfirm>
      </div>

      <div v-if="recoveryCodes.length" class="recovery-box">
        <p class="recovery-tip">{{ t('security.recoveryTip') }}</p>
        <div class="recovery-grid">
          <code v-for="c in recoveryCodes" :key="c">{{ c }}</code>
        </div>
      </div>
    </div>

    <!-- 会话管理 -->
    <div class="security-card">
      <div class="security-head">
        <span class="security-ico"><n-icon :component="HardwareChipOutline" /></span>
        <div>
          <h3>{{ t('security.sessions') }}</h3>
          <p>{{ t('security.sessionsDesc') }}</p>
        </div>
        <n-button size="small" tertiary @click="refreshSessions">
          <template #icon><n-icon :component="ReloadOutline" /></template>
          {{ t('security.refresh') }}
        </n-button>
      </div>
      <ul class="session-list">
        <li v-for="s in sessions" :key="s.tokenId" class="session-row">
          <div>
            <strong>{{ s.deviceName }}</strong>
            <small>{{ t('security.lastActive') }}: {{ new Date(s.lastRefreshAt).toLocaleString() }}</small>
          </div>
          <n-button size="small" tertiary type="error" @click="onRevoke(s.tokenId)">
            {{ t('security.revoke') }}
          </n-button>
        </li>
        <li v-if="!sessions.length" class="session-empty">{{ t('security.none') }}</li>
      </ul>
    </div>

    <!-- 开启 2FA 弹窗 -->
    <n-modal v-model:show="showSetup" :title="t('security.mfaSetupTitle')" preset="card" style="width: 460px; max-width: 92vw">
      <div class="mfa-setup">
        <p>{{ t('security.mfaSetupDesc') }}</p>
        <div class="qr-zone">
          <img v-if="otpauthUrl" :src="`https://api.qrserver.com/v1/create-qr-code/?size=180x180&data=${encodeURIComponent(otpauthUrl)}`" alt="qr" class="qr-img" />
          <div class="secret-box">
            <span class="secret-lbl">{{ t('security.secret') }}</span>
            <code>{{ secret }}</code>
          </div>
        </div>
        <n-input v-model:value="setupCode" :placeholder="t('auth.codePlaceholder')" />
      </div>
      <template #footer>
        <div class="modal-foot">
          <n-button @click="showSetup = false">{{ t('auth.backToLogin') }}</n-button>
          <n-button type="primary" :loading="setupLoading" @click="confirmSetup">{{ t('security.confirmEnable') }}</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>
