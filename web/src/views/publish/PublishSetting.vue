<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import {
  NCard,
  NTag,
  NButton,
  NEmpty,
  NSwitch,
  NSelect,
  NCheckboxGroup,
  NCheckbox,
  NInput,
  NIcon,
  NText,
  useMessage,
} from 'naive-ui'
import { SendOutline } from '@vicons/ionicons5'
import { fetchChannels } from '@/api/publish'
import { useNovelStore } from '@/stores/novel'
import { storeToRefs } from 'pinia'
import type { PublishChannel } from '@/api/publish'

const message = useMessage()
const novelStore = useNovelStore()
const { novels } = storeToRefs(novelStore)

const channels = ref<PublishChannel[]>([])
const selectedNovel = ref<string>('')
const selectedChannels = ref<string[]>([])
const note = ref('')

const novelOptions = computed(() => novels.value.map((n) => ({ label: n.title, value: n.id })))
const enabledChannels = computed(() => channels.value.filter((c) => c.enabled))

onMounted(async () => {
  channels.value = await fetchChannels()
  if (!novels.value.length) await novelStore.loadNovels()
  if (novels.value.length) selectedNovel.value = novels.value[0].id
  selectedChannels.value = channels.value.filter((c) => c.enabled).map((c) => c.id)
})

function toggleChannel(id: string, enabled: boolean) {
  const ch = channels.value.find((c) => c.id === id)
  if (ch) ch.enabled = enabled
}

function publish() {
  if (!selectedNovel.value) {
    message.warning('请选择要发布的作品')
    return
  }
  if (!selectedChannels.value.length) {
    message.warning('请至少选择一个发布渠道')
    return
  }
  const names = channels.value
    .filter((c) => selectedChannels.value.includes(c.id))
    .map((c) => c.name)
    .join('、')
  message.success(`已向 ${names} 提交发布申请`)
}
</script>

<template>
  <div class="page">
    <header class="page__head">
      <div>
        <h1 class="page__title">发布管理</h1>
        <p class="page__sub">把写好的故事送到读者面前，多渠道一键分发。</p>
      </div>
    </header>

    <n-card :bordered="false" title="发布渠道" class="section">
      <div v-if="channels.length" class="channels">
        <div v-for="ch in channels" :key="ch.id" class="channel">
          <div class="channel__left">
            <span class="channel__name">{{ ch.name }}</span>
            <n-tag
              size="small"
              :bordered="false"
              :type="ch.status === 'connected' ? 'success' : 'default'"
            >
              {{ ch.status === 'connected' ? '已连接' : '未连接' }}
            </n-tag>
          </div>
          <n-switch
            :value="ch.enabled"
            @update:value="toggleChannel(ch.id, $event)"
          />
        </div>
      </div>
      <n-empty v-else description="暂无可用的发布渠道" />
    </n-card>

    <n-card :bordered="false" title="发布作品" class="section">
      <div class="publish-form">
        <div class="field">
          <n-text depth="2" class="field__label">选择作品</n-text>
          <n-select
            v-model:value="selectedNovel"
            :options="novelOptions"
            placeholder="选择要发布的作品"
            style="max-width: 320px"
          />
        </div>
        <div class="field">
          <n-text depth="2" class="field__label">发布渠道</n-text>
          <n-checkbox-group v-model:value="selectedChannels">
            <n-space>
              <n-checkbox
                v-for="ch in enabledChannels"
                :key="ch.id"
                :value="ch.id"
                :label="ch.name"
              />
            </n-space>
          </n-checkbox-group>
        </div>
        <div class="field">
          <n-text depth="2" class="field__label">发布说明</n-text>
          <n-input
            v-model:value="note"
            type="textarea"
            placeholder="给编辑或读者的一段话，例如「第一卷完结，开始连载第二卷」"
            :autosize="{ minRows: 3, maxRows: 6 }"
            style="max-width: 520px"
          />
        </div>
        <n-button type="primary" :disabled="!enabledChannels.length" @click="publish">
          <template #icon><n-icon :component="SendOutline" /></template>
          发布
        </n-button>
      </div>
    </n-card>
  </div>
</template>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.page__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
}
.page__title {
  margin: 0;
  font-size: 22px;
  font-weight: 600;
}
.page__sub {
  margin: 4px 0 0;
  color: var(--mf-text-3);
}
.section {
  width: 100%;
}
.channels {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.channel {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border: 1px solid var(--mf-border);
  border-radius: 10px;
}
.channel__left {
  display: flex;
  align-items: center;
  gap: 10px;
}
.channel__name {
  font-weight: 600;
  color: var(--mf-text);
}
.publish-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.field__label {
  font-size: 13px;
}
</style>
