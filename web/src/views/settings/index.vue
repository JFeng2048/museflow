<script setup lang="ts">
import { computed, ref, reactive, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import {
  NCard,
  NGrid,
  NGi,
  NTabs,
  NTabPane,
  NButton,
  NSwitch,
  NSelect,
  NInput,
  NModal,
  NForm,
  NFormItem,
  NTag,
  NEmpty,
  NIcon,
  NSpace,
  NPopconfirm,
  useMessage,
} from 'naive-ui'
import { LogoGithub, LogoWechat, LogoAlipay, CameraOutline } from '@vicons/ionicons5'
import UserAvatar from '@/components/common/UserAvatar.vue'
import { useUserStore } from '@/stores/system/user'
import { useNovelStore } from '@/stores/novel'
import { useModelStore } from '@/stores/model'
import { useCreditStore } from '@/stores/credit'
import { fetchChannels } from '@/api/publish'
import { bindProvider, unbindProvider } from '@/api/system/auth'
import type { PublishChannel } from '@/api/publish'
import { PROTOCOLS } from '@/types/model'
import type { ModelProvider, AIModel } from '@/types/model'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const message = useMessage()

const userStore = useUserStore()
const novelStore = useNovelStore()
const modelStore = useModelStore()
const creditStore = useCreditStore()
const { user } = storeToRefs(userStore)
const { records, tasks, activityBalance, permanentBalance, nextExpiry } = storeToRefs(creditStore)

const tab = ref<string>((route.query.tab as string) || 'profile')
function switchTab(name: string) {
  tab.value = name
  router.replace({ name: 'settings', query: { tab: name } })
}

const currentStats = computed(() => novelStore.userStats(user.value?.id || 'u_demo'))

// ——— 个人资料 ———
const profile = reactive({
  name: user.value?.name || '',
  email: user.value?.email || '',
  bio: user.value?.bio || '',
  avatar: user.value?.avatar || '',
  avatarColor: user.value?.avatarColor || '#5B8DEF',
})
function saveProfile() {
  userStore.update({ ...profile })
  message.success(t('settings.saved'))
}

// 头像更换弹窗
const showAvatarPicker = ref(false)
const avatarColor = ref<string>('#5B8DEF')
const avatarEmoji = ref<string>('')
const PRESET_COLORS = ['#5B8DEF', '#b9853f', '#2f6f5e', '#a14b7a', '#7c4dff', '#1a2332', '#c25c4a', '#3aa6a0']
const PRESET_AVATARS = [
  { emoji: '✦', color: '#b9853f' },
  { emoji: '☾', color: '#1a2332' },
  { emoji: '✺', color: '#2f6f5e' },
  { emoji: '❀', color: '#a14b7a' },
  { emoji: '◐', color: '#3aa6a0' },
  { emoji: '✪', color: '#7c4dff' },
]
function openAvatarPicker() {
  avatarColor.value = profile.avatarColor || '#5B8DEF'
  avatarEmoji.value = profile.avatar || ''
  showAvatarPicker.value = true
}
function pickAvatar(item: { emoji: string; color: string }) {
  avatarEmoji.value = item.emoji
  avatarColor.value = item.color
}
function pickColor(c: string) {
  avatarColor.value = c
}
function clearAvatar() {
  avatarEmoji.value = ''
}
function confirmAvatar() {
  profile.avatar = avatarEmoji.value
  profile.avatarColor = avatarColor.value
  // 立即保存
  userStore.update({ avatar: profile.avatar, avatarColor: profile.avatarColor })
  message.success(t('settings.saved'))
  showAvatarPicker.value = false
}

// ——— 模型配置 ———
const providerOptions = computed(() =>
  modelStore.providers.map((p) => ({ label: p.name, value: p.id })),
)
const protocolOptions = PROTOCOLS.map((p) => ({ label: p.label, value: p.value }))

// 供应商弹窗
const showProvider = ref(false)
const editingProvider = ref<string | null>(null)
const providerForm = reactive<Partial<ModelProvider>>({
  name: '',
  protocol: 'openai',
  baseUrl: '',
  apiKey: '',
  organization: '',
})
function openAddProvider() {
  editingProvider.value = null
  Object.assign(providerForm, { name: '', protocol: 'openai', baseUrl: '', apiKey: '', organization: '' })
  showProvider.value = true
}
function openEditProvider(p: ModelProvider) {
  editingProvider.value = p.id
  Object.assign(providerForm, { name: p.name, protocol: p.protocol, baseUrl: p.baseUrl, apiKey: p.apiKey, organization: p.organization || '' })
  showProvider.value = true
}
function saveProvider() {
  if (!providerForm.name?.trim()) {
    message.warning(t('settings.model.nameRequired'))
    return
  }
  if (editingProvider.value) {
    modelStore.updateProvider(editingProvider.value, { ...providerForm })
  } else {
    modelStore.addProvider({ name: providerForm.name!, protocol: providerForm.protocol!, baseUrl: providerForm.baseUrl!, apiKey: providerForm.apiKey!, organization: providerForm.organization })
  }
  message.success(t('settings.saved'))
  showProvider.value = false
}

// 模型弹窗
const showModel = ref(false)
const editingModel = ref<string | null>(null)
const modelForm = reactive<Partial<AIModel>>({
  providerId: '',
  name: '',
  apiModel: '',
  contextK: 32,
  enabled: true,
  description: '',
})
function openAddModel() {
  editingModel.value = null
  Object.assign(modelForm, { providerId: modelStore.providers[0]?.id || '', name: '', apiModel: '', contextK: 32, enabled: true, description: '' })
  showModel.value = true
}
function openEditModel(m: AIModel) {
  editingModel.value = m.id
  Object.assign(modelForm, { providerId: m.providerId, name: m.name, apiModel: m.apiModel, contextK: m.contextK, enabled: m.enabled, description: m.description || '' })
  showModel.value = true
}
function saveModel() {
  if (!modelForm.name?.trim() || !modelForm.apiModel?.trim()) {
    message.warning(t('settings.model.modelFieldsRequired'))
    return
  }
  if (editingModel.value) {
    modelStore.updateModel(editingModel.value, { ...modelForm })
  } else {
    modelStore.addModel({ providerId: modelForm.providerId!, name: modelForm.name!, apiModel: modelForm.apiModel!, contextK: modelForm.contextK || 32, description: modelForm.description })
  }
  message.success(t('settings.saved'))
  showModel.value = false
}

// ——— 积分 ———
const showPay = ref(false)
const paying = ref(false)
const selectedPkg = ref('p_98')
const payPkg = ref<{ credits: number; price: number } | null>(null)
function openRecharge() {
  const pkg = creditStore.packages.find((p) => p.id === selectedPkg.value)
  if (!pkg) return
  payPkg.value = { credits: pkg.credits, price: pkg.price }
  paying.value = false
  showPay.value = true
}
function confirmPay() {
  if (!payPkg.value) return
  paying.value = true
  setTimeout(() => {
    creditStore.recharge(payPkg.value!.credits, `充值 · ${payPkg.value!.credits} 永久积分`)
    message.success(t('credits.recharged', { n: payPkg.value!.credits }))
    paying.value = false
    showPay.value = false
  }, 800)
}
function doTask(taskId: string) {
  creditStore.completeTask(taskId)
  message.success(t('credits.taskDone'))
}

// ——— 小说平台配置 ———
const channels = ref<PublishChannel[]>([])
const showChannel = ref(false)
const editingChannel = ref<PublishChannel | null>(null)
const channelForm = reactive<Partial<PublishChannel>>({
  account: '',
  penName: '',
})
onMounted(async () => {
  channels.value = await fetchChannels()
})
function openChannel(c: PublishChannel) {
  editingChannel.value = c
  Object.assign(channelForm, { account: c.account || '', penName: c.penName || '' })
  showChannel.value = true
}
function saveChannel() {
  const c = editingChannel.value
  if (!c) return
  const idx = channels.value.findIndex((x) => x.id === c.id)
  if (idx >= 0) {
    channels.value[idx] = { ...c, account: channelForm.account, penName: channelForm.penName }
  }
  message.success(t('settings.saved'))
  showChannel.value = false
}
function toggleChannel(c: PublishChannel, enabled: boolean) {
  c.enabled = enabled
  c.status = enabled ? 'connected' : 'disconnected'
}

// ——— 第三方账号绑定（GitHub / 微信）———
const bindings = computed(() => user.value?.bindings || {})
async function bindGithub() {
  const info = await bindProvider('github')
  userStore.setBinding('github', info)
  message.success(t('settings.account.bindSuccess', { p: 'GitHub' }))
}
async function bindWechat() {
  const info = await bindProvider('wechat')
  userStore.setBinding('wechat', info)
  message.success(t('settings.account.bindSuccess', { p: '微信' }))
}
function unbind(provider: 'github' | 'wechat') {
  unbindProvider(provider)
  userStore.setBinding(provider, undefined)
  message.success(t('settings.account.unbindSuccess'))
}
</script>

<template>
  <div class="settings-page">
    <header class="settings-page-head">
      <p class="eyebrow">{{ t('settings.title') }}</p>
      <h1>{{ t('settings.subtitle') }}</h1>
    </header>

    <n-tabs :value="tab" @update:value="switchTab" type="line" animated>
      <!-- 个人资料 -->
      <n-tab-pane name="profile" :tab="t('settings.profile')">
        <n-card :bordered="false" class="settings-card">
          <div class="profile-head">
            <div class="profile-avatar-wrap">
              <UserAvatar
                :name="profile.name"
                :avatar="profile.avatar"
                :color="profile.avatarColor"
                :size="84"
              />
              <button class="profile-avatar-mask" type="button" @click="openAvatarPicker">
                <n-icon :component="CameraOutline" :size="18" />
                <span>{{ t('settings.profileChangeAvatar') }}</span>
              </button>
            </div>
            <div class="profile-meta">
              <p class="eyebrow">{{ t('settings.profile') }}</p>
              <h2 class="profile-name">{{ profile.name || t('common.writer') }}</h2>
              <p class="profile-bio">{{ profile.bio || t('settings.profileBioHint') }}</p>
            </div>
          </div>
          <n-form :model="profile" label-placement="top" style="max-width: 480px; margin-top: 8px">
            <n-form-item :label="t('settings.name')">
              <n-input v-model:value="profile.name" />
            </n-form-item>
            <n-form-item :label="t('settings.email')">
              <n-input v-model:value="profile.email" />
            </n-form-item>
            <n-form-item :label="t('settings.bio')">
              <n-input v-model:value="profile.bio" type="textarea" :autosize="{ minRows: 3, maxRows: 6 }" />
            </n-form-item>
          </n-form>
          <template #footer>
            <n-space justify="end">
              <n-button type="primary" @click="saveProfile">{{ t('common.save') }}</n-button>
            </n-space>
          </template>
        </n-card>
      </n-tab-pane>

      <!-- 模型配置 -->
      <n-tab-pane name="model" :tab="t('settings.model.title')">
        <div class="settings-panel">
          <section class="settings-block">
            <div class="settings-block-head">
              <h3>{{ t('settings.model.providers') }}</h3>
              <n-button size="small" @click="openAddProvider">+ {{ t('settings.model.addProvider') }}</n-button>
            </div>
            <n-empty v-if="!modelStore.providers.length" :description="t('settings.model.empty')" />
            <div class="settings-list">
              <div v-for="p in modelStore.providers" :key="p.id" class="settings-row">
                <div class="settings-row-main">
                  <span class="settings-row-title">{{ p.name }}</span>
                  <n-tag size="small" :bordered="false">{{ PROTOCOLS.find((x) => x.value === p.protocol)?.label }}</n-tag>
                  <span class="settings-row-sub">{{ p.baseUrl }}</span>
                  <n-tag v-if="p.system" size="small" type="warning" :bordered="false">{{ t('settings.model.system') }}</n-tag>
                </div>
                <n-space>
                  <n-button v-if="!p.system" size="small" quaternary @click="openEditProvider(p)">{{ t('common.edit') }}</n-button>
                  <n-popconfirm v-if="!p.system" @positive-click="modelStore.removeProvider(p.id)">
                    <template #trigger>
                      <n-button size="small" quaternary type="error">{{ t('common.delete') }}</n-button>
                    </template>
                    {{ t('common.confirmDelete') }}
                  </n-popconfirm>
                </n-space>
              </div>
            </div>
          </section>

          <section class="settings-block">
            <div class="settings-block-head">
              <h3>{{ t('settings.model.models') }}</h3>
              <n-button size="small" @click="openAddModel">+ {{ t('settings.model.addModel') }}</n-button>
            </div>
            <div class="settings-list">
              <div v-for="m in modelStore.models" :key="m.id" class="settings-row">
                <div class="settings-row-main">
                  <span class="settings-row-title">{{ m.name }}</span>
                  <n-tag size="small" :bordered="false">{{ m.apiModel }}</n-tag>
                  <span v-if="m.system" class="settings-row-sub">{{ t('settings.model.cost', { n: m.creditCost }) }}</span>
                  <span v-else class="settings-row-sub free">{{ t('settings.model.free') }}</span>
                </div>
                <n-space>
                  <n-switch :value="m.enabled" @update:value="(v: boolean) => modelStore.toggleModel(m.id, v)" />
                  <n-button v-if="!m.system" size="small" quaternary @click="openEditModel(m)">{{ t('common.edit') }}</n-button>
                  <n-popconfirm v-if="!m.system" @positive-click="modelStore.removeModel(m.id)">
                    <template #trigger>
                      <n-button size="small" quaternary type="error">{{ t('common.delete') }}</n-button>
                    </template>
                    {{ t('common.confirmDelete') }}
                  </n-popconfirm>
                </n-space>
              </div>
            </div>
          </section>
        </div>
      </n-tab-pane>

      <!-- 我的积分 -->
      <n-tab-pane name="credits" :tab="t('settings.credits')">
        <n-grid :cols="2" :x-gap="16" responsive="screen" item-responsive>
          <n-gi span="2 m:1">
            <n-card :bordered="false" class="settings-card settings-stat credit-stat activity">
              <p class="eyebrow">{{ t('credits.activity') }}</p>
              <p class="big">{{ activityBalance }}</p>
              <p class="sub">{{ t('credits.activityDesc') }}</p>
              <p v-if="nextExpiry" class="exp">{{ t('credits.expireAt', { d: nextExpiry.slice(0, 10) }) }}</p>
            </n-card>
          </n-gi>
          <n-gi span="2 m:1">
            <n-card :bordered="false" class="settings-card settings-stat credit-stat permanent">
              <p class="eyebrow">{{ t('credits.permanent') }}</p>
              <p class="big">{{ permanentBalance }}</p>
              <p class="sub">{{ t('credits.permanentDesc') }}</p>
              <p class="exp">{{ t('credits.noExpiry') }}</p>
            </n-card>
          </n-gi>
        </n-grid>

        <n-card :bordered="false" class="settings-card" style="margin-top: 16px">
          <div class="settings-block-head">
            <h3>{{ t('credits.tasks') }}</h3>
          </div>
          <div class="settings-list">
            <div v-for="tk in tasks" :key="tk.id" class="settings-row">
              <div class="settings-row-main">
                <span class="settings-row-title">{{ tk.title }}</span>
                <span class="settings-row-sub">{{ tk.desc }}</span>
              </div>
              <n-button size="small" :disabled="tk.done" type="primary" ghost @click="doTask(tk.id)">
                {{ tk.done ? t('credits.done') : `+${tk.reward}` }}
              </n-button>
            </div>
          </div>
        </n-card>

        <n-card :bordered="false" class="settings-card" style="margin-top: 16px">
          <div class="settings-block-head"><h3>{{ t('credits.records') }}</h3></div>
          <n-empty v-if="!records.length" :description="t('credits.noRecords')" />
          <div v-else class="settings-list">
            <div v-for="r in records" :key="r.id" class="settings-row">
              <div class="settings-row-main">
                <span class="settings-row-title">{{ r.note }}</span>
                <span class="settings-row-sub">{{ r.at.slice(0, 16).replace('T', ' ') }}</span>
              </div>
              <span :class="['amt', r.amount > 0 ? 'gain' : 'loss']">
                {{ r.amount > 0 ? '+' : '' }}{{ r.amount }}
              </span>
            </div>
          </div>
        </n-card>

        <n-card :bordered="false" class="settings-card recharge-card" style="margin-top: 16px">
          <div class="settings-block-head">
            <div>
              <h3>{{ t('credits.recharge') }}</h3>
              <p class="recharge-sub">{{ t('credits.rechargeDesc') }}</p>
            </div>
            <n-tag type="success" :bordered="false" size="small">{{ t('credits.permanent') }}</n-tag>
          </div>
          <div class="settings-pkg-row">
            <button
              v-for="p in creditStore.packages"
              :key="p.id"
              class="settings-pkg"
              :class="{ active: selectedPkg === p.id }"
              @click="selectedPkg = p.id"
            >
              <span v-if="p.popular" class="badge">{{ p.tag }}</span>
              <span class="amt">{{ p.credits }}</span>
              <span class="price">¥{{ p.price }}</span>
            </button>
          </div>
          <n-space justify="end" style="margin-top: 12px">
            <n-button type="primary" @click="openRecharge">
              <template #icon>
                <n-icon :component="LogoAlipay" />
              </template>
              {{ t('credits.confirmRecharge') }}
            </n-button>
          </n-space>
        </n-card>
      </n-tab-pane>

      <!-- 使用统计 -->
      <n-tab-pane name="usage" :tab="t('settings.usage')">
        <n-grid :cols="4" :x-gap="16" responsive="screen" item-responsive>
          <n-gi v-for="s in [
            { k: t('stats.totalNovels'), v: currentStats.totalNovels },
            { k: t('stats.totalWords'), v: currentStats.totalWords },
            { k: t('stats.ongoing'), v: currentStats.statusOngoing },
            { k: t('stats.completed'), v: currentStats.statusCompleted },
          ]" :key="s.k" span="2 m:1">
            <n-card :bordered="false" class="settings-card settings-stat">
              <p class="eyebrow">{{ s.k }}</p>
              <p class="big">{{ s.v }}</p>
            </n-card>
          </n-gi>
        </n-grid>
      </n-tab-pane>

      <!-- 小说平台配置 -->
      <n-tab-pane name="publish" :tab="t('settings.publish.title')">
        <n-card :bordered="false" class="settings-card">
          <div class="settings-block-head">
            <h3>{{ t('settings.publish.channels') }}</h3>
          </div>
          <n-empty v-if="!channels.length" :description="t('common.empty')" />
          <div class="settings-list">
            <div v-for="c in channels" :key="c.id" class="settings-row">
              <div class="settings-row-main">
                <span class="settings-row-title">{{ c.name }}</span>
                <n-tag size="small" :bordered="false" :type="c.status === 'connected' ? 'success' : 'default'">
                  {{ c.status === 'connected' ? t('settings.publish.connected') : t('settings.publish.disconnected') }}
                </n-tag>
                <span class="settings-row-sub">{{ c.desc }}</span>
                <span v-if="c.penName" class="settings-row-sub">{{ t('settings.publish.penName') }}：{{ c.penName }}</span>
              </div>
              <n-space>
                <n-switch :value="c.enabled" @update:value="(v: boolean) => toggleChannel(c, v)" />
                <n-button size="small" quaternary @click="openChannel(c)">{{ t('common.edit') }}</n-button>
              </n-space>
            </div>
          </div>
        </n-card>
      </n-tab-pane>

      <!-- 账号绑定 -->
      <n-tab-pane name="account" :tab="t('settings.account.title')">
        <n-grid :cols="2" :x-gap="16" responsive="screen" item-responsive>
          <n-gi v-for="item in [
            { key: 'github', name: 'GitHub', icon: 'logo-github', bound: !!bindings.github, nick: bindings.github?.nickname },
            { key: 'wechat', name: t('settings.account.wechat'), icon: 'logo-wechat', bound: !!bindings.wechat, nick: bindings.wechat?.nickname },
          ]" :key="item.key" span="2 m:1">
            <n-card :bordered="false" class="settings-card bind-card">
              <div class="bind-head">
                <div class="bind-logo" :class="item.key">
                  <n-icon :component="item.key === 'github' ? LogoGithub : LogoWechat" />
                </div>
                <div class="bind-meta">
                  <p class="bind-name">{{ item.name }}</p>
                  <p class="bind-sub">
                    <n-tag v-if="item.bound" size="small" type="success" :bordered="false">{{ t('settings.account.bound') }}</n-tag>
                    <span v-else class="bind-unbound">{{ t('settings.account.unbound') }}</span>
                    <span v-if="item.bound && item.nick" class="bind-nick">· {{ item.nick }}</span>
                  </p>
                </div>
              </div>
              <n-space justify="end" class="bind-actions">
                <n-popconfirm v-if="item.bound" @positive-click="unbind(item.key as 'github' | 'wechat')">
                  <template #trigger>
                    <n-button size="small" quaternary type="error">{{ t('settings.account.unbind') }}</n-button>
                  </template>
                  {{ t('common.confirmDelete') }}
                </n-popconfirm>
                <n-button v-else size="small" type="primary" @click="item.key === 'github' ? bindGithub() : bindWechat()">
                  {{ t('settings.account.bind') }}
                </n-button>
              </n-space>
            </n-card>
          </n-gi>
        </n-grid>
        <p class="bind-tip">{{ t('settings.account.tip') }}</p>
      </n-tab-pane>
    </n-tabs>

    <!-- 供应商弹窗 -->
    <n-modal v-model:show="showProvider" preset="card" :title="editingProvider ? t('settings.model.editProvider') : t('settings.model.addProvider')" style="width: 520px; max-width: 92vw" :bordered="false">
      <n-form :model="providerForm" label-placement="top">
        <n-form-item :label="t('settings.model.name')">
          <n-input v-model:value="providerForm.name" :placeholder="t('settings.model.namePlaceholder')" />
        </n-form-item>
        <n-form-item :label="t('settings.model.protocol')">
          <n-select v-model:value="providerForm.protocol" :options="protocolOptions" />
        </n-form-item>
        <n-form-item :label="t('settings.model.baseUrl')">
          <n-input v-model:value="providerForm.baseUrl" :placeholder="PROTOCOLS.find((x) => x.value === providerForm.protocol)?.baseUrlHint" />
        </n-form-item>
        <n-form-item :label="t('settings.model.apiKey')">
          <n-input v-model:value="providerForm.apiKey" type="password" show-password-on="click" />
        </n-form-item>
        <n-form-item :label="t('settings.model.organization')">
          <n-input v-model:value="providerForm.organization" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button quaternary @click="showProvider = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" @click="saveProvider">{{ t('common.save') }}</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 模型弹窗 -->
    <n-modal v-model:show="showModel" preset="card" :title="editingModel ? t('settings.model.editModel') : t('settings.model.addModel')" style="width: 520px; max-width: 92vw" :bordered="false">
      <n-form :model="modelForm" label-placement="top">
        <n-form-item :label="t('settings.model.provider')">
          <n-select v-model:value="modelForm.providerId" :options="providerOptions" />
        </n-form-item>
        <n-form-item :label="t('settings.model.modelName')">
          <n-input v-model:value="modelForm.name" :placeholder="t('settings.model.modelNamePlaceholder')" />
        </n-form-item>
        <n-form-item :label="t('settings.model.apiModel')">
          <n-input v-model:value="modelForm.apiModel" :placeholder="t('settings.model.apiModelPlaceholder')" />
        </n-form-item>
        <n-form-item :label="t('settings.model.contextK')">
          <n-input
            :value="String(modelForm.contextK ?? '')"
            @update:value="(v: string) => (modelForm.contextK = Number(v) || 32)"
            placeholder="32"
          />
        </n-form-item>
        <n-form-item :label="t('settings.model.description')">
          <n-input v-model:value="modelForm.description" type="textarea" :autosize="{ minRows: 2 }" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button quaternary @click="showModel = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" @click="saveModel">{{ t('common.save') }}</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 头像选择弹窗 -->
    <n-modal v-model:show="showAvatarPicker" preset="card" :title="t('settings.profileChangeAvatar')" style="width: 520px; max-width: 92vw" :bordered="false">
      <div class="avatar-picker">
        <div class="avatar-picker-preview">
          <UserAvatar
            :name="profile.name"
            :avatar="avatarEmoji"
            :color="avatarColor"
            :size="76"
          />
          <p class="avatar-picker-hint">{{ t('settings.avatarPickerHint') }}</p>
        </div>
        <div class="avatar-picker-section">
          <p class="avatar-picker-label">{{ t('settings.avatarPickerPresets') }}</p>
          <div class="avatar-picker-grid">
            <button
              v-for="(p, i) in PRESET_AVATARS"
              :key="i"
              type="button"
              class="avatar-picker-item"
              :class="{ active: avatarEmoji === p.emoji && avatarColor === p.color }"
              :style="{ background: p.color }"
              @click="pickAvatar(p)"
            >
              {{ p.emoji }}
            </button>
          </div>
        </div>
        <div class="avatar-picker-section">
          <p class="avatar-picker-label">{{ t('settings.avatarPickerColors') }}</p>
          <div class="avatar-picker-colors">
            <button
              v-for="c in PRESET_COLORS"
              :key="c"
              type="button"
              class="avatar-picker-color"
              :class="{ active: avatarColor === c }"
              :style="{ background: c }"
              @click="pickColor(c)"
            />
            <button
              type="button"
              class="avatar-picker-color reset"
              :class="{ active: !avatarEmoji }"
              @click="clearAvatar"
            >
              <span>{{ t('settings.avatarPickerReset') }}</span>
            </button>
          </div>
        </div>
      </div>
      <template #footer>
        <n-space justify="end">
          <n-button quaternary @click="showAvatarPicker = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" @click="confirmAvatar">{{ t('common.save') }}</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 支付宝支付浮层 -->
    <n-modal v-model:show="showPay" preset="card" :title="t('credits.payTitle')" style="width: 420px; max-width: 92vw" :bordered="false">
      <div v-if="payPkg" class="pay-modal">
        <div class="pay-method">
          <span class="pay-method-label">{{ t('credits.payMethod') }}</span>
          <div class="pay-method-list">
            <div class="pay-method-item active">
              <n-icon :component="LogoAlipay" class="pay-method-ico alipay" />
              <span>{{ t('credits.payAlipay') }}</span>
            </div>
          </div>
        </div>
        <div class="pay-amount">
          <span class="pay-amount-label">{{ t('credits.payAmount') }}</span>
          <span class="pay-amount-value">¥{{ payPkg.price }}</span>
          <span class="pay-amount-sub">= {{ payPkg.credits }} {{ t('credits.permanent') }}</span>
        </div>
        <div class="pay-qr">
          <div class="pay-qr-box">
            <n-icon :component="LogoAlipay" class="pay-qr-ico" />
            <p class="pay-qr-tip">{{ t('credits.payScanTip') }}</p>
          </div>
        </div>
      </div>
      <template #footer>
        <n-space justify="end">
          <n-button quaternary :disabled="paying" @click="showPay = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="paying" @click="confirmPay">
            {{ paying ? t('credits.payPending') : t('credits.payConfirm') }}
          </n-button>
        </n-space>
      </template>
    </n-modal>
    <n-modal v-model:show="showChannel" preset="card" :title="editingChannel ? t('settings.publish.edit', { n: editingChannel.name }) : ''" style="width: 520px; max-width: 92vw" :bordered="false">
      <n-form :model="channelForm" label-placement="top">
        <n-form-item :label="t('settings.publish.account')">
          <n-input v-model:value="channelForm.account" :placeholder="t('settings.publish.accountPlaceholder')" />
        </n-form-item>
        <n-form-item :label="t('settings.publish.penName')">
          <n-input v-model:value="channelForm.penName" :placeholder="t('settings.publish.penNamePlaceholder')" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button quaternary @click="showChannel = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" @click="saveChannel">{{ t('common.save') }}</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

