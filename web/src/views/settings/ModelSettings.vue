<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NEmpty,
  NIcon,
  NPopconfirm,
  NSelect,
  NSpace,
  NSwitch,
  NTag,
  useMessage,
} from 'naive-ui'
import { CreateOutline, PencilOutline, TrashOutline } from '@vicons/ionicons5'
import { useModelStore } from '@/stores/model'
import ProviderFormModal from '@/components/model/ProviderFormModal.vue'
import ModelFormModal from '@/components/model/ModelFormModal.vue'
import CapabilityIcons from '@/components/model/CapabilityIcons.vue'
import KeySlot from '@/components/model/KeySlot.vue'
import type { UserModel, UserProvider } from '@/types/model'
import { MODEL_TYPES, modelTypeMeta } from '@/types/model'

/**
 * 用户端模型配置。
 *
 * 与后台的模型配置是同一套数据的两个视角，差别只在读写边界：
 *  - 可用模型清单是只读的，平台模型由后台维护并计平台积分，自定义模型不计费；
 *  - 自定义渠道与自定义模型可增删改，密钥同样是「只写不读」的一次性凭据。
 * 因为费用由用户直接付给上游厂商，自定义模型没有积分价一列。
 */
const { t } = useI18n()
const message = useMessage()
const store = useModelStore()

onMounted(() => {
  store.fetchUserData()
  store.fetchPublicSettings()
})

const typeOptions = computed(() => [
  { label: t('common.all'), value: '' },
  ...MODEL_TYPES.map((m) => ({ label: t(`model.type.${m.value}`), value: m.value })),
])

function changeType(value: string) {
  store.availableType = value as typeof store.availableType
  store.fetchAvailable()
}

function typeLabel(type: string): string {
  return t(`model.type.${modelTypeMeta(type).value}`)
}

function creditLabel(cost: number): string {
  return cost ? t('settings.model.cost', { n: cost }) : t('settings.model.free')
}

/**
 * 平台公开配置：后台把「对用户可见」打开、且不是机密的配置项。
 *
 * 机密项即便被误标成公开也不展示：后端不会下发机密明文，
 * 这里再兜底过滤一次，免得出现一行只有标签没有值的列表。
 */
const publicConfigs = computed(() => store.publicSettings.filter((s) => !s.isSecret))

// ---------- 自定义渠道 ----------

const showProvider = ref(false)
const editingProvider = ref<UserProvider | null>(null)

function openAddProvider() {
  editingProvider.value = null
  showProvider.value = true
}

function openEditProvider(row: UserProvider) {
  editingProvider.value = row
  showProvider.value = true
}

async function submitProvider(payload: Record<string, unknown>) {
  try {
    if (editingProvider.value) {
      await store.editUserProvider(editingProvider.value.id, payload as never)
    } else {
      await store.addUserProvider(payload as never)
    }
    message.success(t('settings.saved'))
    showProvider.value = false
  } catch {
    // 失败提示由请求层统一给出。
  }
}

/**
 * 渠道启用开关。
 *
 * 更新接口要求整条渠道信息，所以这里把行上原有的字段一起带上，
 * 只改 is_active 一个值——不能让用户为了拨一个开关把密钥重填一遍。
 */
async function toggleUserProvider(row: UserProvider, value: boolean) {
  await store.editUserProvider(row.id, {
    name: row.name,
    protocol: row.protocol,
    baseUrl: row.baseUrl,
    organization: row.organization,
    isActive: value,
  })
}

// ---------- 自定义模型 ----------

const showModel = ref(false)
const editingModel = ref<UserModel | null>(null)
const userProviderOptions = computed(() =>
  store.userProviders.map((p) => ({ label: p.name, value: p.id })),
)

function openAddModel() {
  if (!store.userProviders.length) {
    message.warning(t('settings.model.needProviderFirst'))
    return
  }
  editingModel.value = null
  showModel.value = true
}

function openEditModel(row: UserModel) {
  editingModel.value = row
  showModel.value = true
}

async function submitModel(payload: Record<string, unknown>) {
  try {
    if (editingModel.value) {
      await store.editUserModel(editingModel.value.id, payload as never)
    } else {
      await store.addUserModel(payload as never)
    }
    message.success(t('settings.saved'))
    showModel.value = false
  } catch {
    // 失败提示由请求层统一给出。
  }
}

function providerName(id: number): string {
  return store.providerName(id, true)
}
</script>

<template>
  <div class="settings-panel">
    <!-- 可用模型：平台 + 自定义的合并视图，只读 -->
    <section class="settings-block">
      <div class="settings-block-head">
        <div>
          <h3>{{ t('settings.model.available') }}</h3>
          <p class="block-hint">{{ t('settings.model.availableHint') }}</p>
        </div>
        <n-select
          :value="store.availableType"
          :options="typeOptions"
          size="small"
          style="width: 150px"
          @update:value="changeType"
        />
      </div>
      <n-empty v-if="!store.available.length" :description="t('settings.model.availableEmpty')" />
      <div v-else class="settings-list">
        <div v-for="m in store.available" :key="`${m.source}-${m.modelId}`" class="settings-row">
          <div class="settings-row-main">
            <div class="row-titles">
              <span class="settings-row-title">{{ m.name }}</span>
              <n-tag v-if="m.source === 'platform'" size="small" type="warning" :bordered="false">
                {{ t('settings.model.platform') }}
              </n-tag>
              <n-tag v-else size="small" type="success" :bordered="false">
                {{ t('settings.model.custom') }}
              </n-tag>
              <n-tag size="small" :bordered="false">{{ typeLabel(m.modelType) }}</n-tag>
            </div>
            <span class="settings-row-sub mono">{{ m.apiModel }}</span>
            <span class="settings-row-sub">
              {{ m.description || t('settings.model.noDescription') }}
            </span>
          </div>
          <div class="row-trail">
            <span class="row-cost">{{ creditLabel(m.creditCost) }}</span>
            <capability-icons :capabilities="m.capabilities" />
          </div>
        </div>
      </div>
    </section>

    <!-- 平台公开配置：管理端标记为「对用户可见」的配置项，只读 -->
    <section v-if="publicConfigs.length" class="settings-block">
      <div class="settings-block-head">
        <div>
          <h3>{{ t('settings.model.publicConfig') }}</h3>
          <p class="block-hint">{{ t('settings.model.publicConfigHint') }}</p>
        </div>
      </div>
      <div class="settings-list">
        <div v-for="s in publicConfigs" :key="s.id" class="settings-row">
          <div class="settings-row-main">
            <div class="row-titles">
              <span class="settings-row-title">{{ s.key }}</span>
              <n-tag size="small" :bordered="false">{{ s.configGroup }}</n-tag>
            </div>
            <span v-if="s.description" class="settings-row-sub">{{ s.description }}</span>
          </div>
          <span class="public-value">{{ s.value || '—' }}</span>
        </div>
      </div>
    </section>

    <!-- 我的自定义渠道：自带 Base URL 与 API Key -->
    <section class="settings-block">
      <div class="settings-block-head">
        <div>
          <h3>{{ t('settings.model.myProviders') }}</h3>
          <p class="block-hint">{{ t('settings.model.myProvidersHint') }}</p>
        </div>
        <n-button size="small" @click="openAddProvider">
          <template #icon><n-icon :component="CreateOutline" /></template>
          {{ t('settings.model.addProvider') }}
        </n-button>
      </div>
      <n-empty v-if="!store.userProviders.length" :description="t('settings.model.myProvidersEmpty')" />
      <div v-else class="settings-list">
        <div v-for="p in store.userProviders" :key="p.id" class="settings-row">
          <div class="settings-row-main">
            <div class="row-titles">
              <span class="settings-row-title">{{ p.name }}</span>
              <n-tag size="small" :bordered="false">{{ p.protocol }}</n-tag>
            </div>
            <span class="settings-row-sub mono">{{ p.baseUrl }}</span>
            <key-slot :hint="p.apiKeyHint" :updated-at="p.apiKeyUpdatedAt" compact />
          </div>
          <n-space align="center">
            <n-switch
              size="small"
              :value="p.isActive"
              @update:value="(v: boolean) => toggleUserProvider(p, v)"
            />
            <n-button size="small" quaternary @click="openEditProvider(p)">
              <template #icon><n-icon :component="PencilOutline" /></template>
              {{ t('common.edit') }}
            </n-button>
            <n-popconfirm @positive-click="store.removeUserProvider(p.id)">
              <template #trigger>
                <n-button size="small" quaternary type="error">
                  <template #icon><n-icon :component="TrashOutline" /></template>
                  {{ t('common.delete') }}
                </n-button>
              </template>
              {{ t('settings.model.confirmDeleteProvider') }}
            </n-popconfirm>
          </n-space>
        </div>
      </div>
    </section>

    <!-- 我的自定义模型：挂在自定义渠道下 -->
    <section class="settings-block">
      <div class="settings-block-head">
        <div>
          <h3>{{ t('settings.model.myModels') }}</h3>
          <p class="block-hint">{{ t('settings.model.myModelsHint') }}</p>
        </div>
        <n-button size="small" @click="openAddModel">
          <template #icon><n-icon :component="CreateOutline" /></template>
          {{ t('settings.model.addModel') }}
        </n-button>
      </div>
      <n-empty v-if="!store.userModels.length" :description="t('settings.model.myModelsEmpty')" />
      <div v-else class="settings-list">
        <div v-for="m in store.userModels" :key="m.id" class="settings-row">
          <div class="settings-row-main">
            <div class="row-titles">
              <span class="settings-row-title">{{ m.name }}</span>
              <n-tag size="small" :bordered="false">{{ typeLabel(m.modelType) }}</n-tag>
            </div>
            <span class="settings-row-sub mono">{{ m.apiModel }}</span>
            <span class="settings-row-sub">{{ providerName(m.providerId) }}</span>
          </div>
          <n-space align="center">
            <n-switch
              size="small"
              :value="m.isActive"
              @update:value="
                (v: boolean) =>
                  store.editUserModel(m.id, {
                    providerId: m.providerId,
                    name: m.name,
                    modelType: m.modelType,
                    apiModel: m.apiModel,
                    contextWindow: m.contextWindow,
                    maxOutputTokens: m.maxOutputTokens,
                    capabilities: m.capabilities,
                    description: m.description,
                    isActive: v,
                  })
              "
            />
            <n-button size="small" quaternary @click="openEditModel(m)">
              <template #icon><n-icon :component="PencilOutline" /></template>
              {{ t('common.edit') }}
            </n-button>
            <n-popconfirm @positive-click="store.removeUserModel(m.id)">
              <template #trigger>
                <n-button size="small" quaternary type="error">
                  <template #icon><n-icon :component="TrashOutline" /></template>
                  {{ t('common.delete') }}
                </n-button>
              </template>
              {{ t('settings.model.confirmDeleteModel') }}
            </n-popconfirm>
          </n-space>
        </div>
      </div>
    </section>
  </div>

  <provider-form-modal
    :show="showProvider"
    :provider="editingProvider"
    scope="user"
    @update:show="showProvider = $event"
    @submit="submitProvider"
  />
  <model-form-modal
    :show="showModel"
    :model="editingModel"
    scope="user"
    :providers="userProviderOptions"
    @update:show="showModel = $event"
    @submit="submitModel"
  />
</template>
