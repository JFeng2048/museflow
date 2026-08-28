<script setup lang="ts">
import { ref, h, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NCard, NList, NListItem, NTag, NButton, NSpace, NSwitch, NIcon, NEmpty,
  NModal, NForm, NFormItem, NInput, NSelect, useMessage,
} from 'naive-ui'
import { AddOutline, MegaphoneOutline } from '@vicons/ionicons5'
import { adminAnnouncements } from '@/mock/admin'
import type { AdminAnnouncement } from '@/types/admin'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const message = useMessage()
const list = ref<AdminAnnouncement[]>([...adminAnnouncements])

const levelTag: Record<string, { type: 'default' | 'warning' | 'error'; key: string }> = {
  normal: { type: 'default', key: 'admin.announcements.levelNormal' },
  important: { type: 'warning', key: 'admin.announcements.levelImportant' },
  maintenance: { type: 'error', key: 'admin.announcements.levelMaintenance' },
}

const sorted = computed(() =>
  [...list.value].sort((a, b) => Number(b.pinned) - Number(a.pinned) || b.publishedAt.localeCompare(a.publishedAt)),
)

function togglePublish(row: AdminAnnouncement) {
  row.status = row.status === 'published' ? 'draft' : 'published'
  message.success(row.status === 'published' ? t('admin.announcements.published') : t('admin.announcements.unpublishedMsg'))
}

// 新建公告
const showCreate = ref(false)
const form = ref({ title: '', content: '', level: 'normal' as AdminAnnouncement['level'], pinned: false })

const levelOptions = [
  { label: t('admin.announcements.levelNormal'), value: 'normal' },
  { label: t('admin.announcements.levelImportant'), value: 'important' },
  { label: t('admin.announcements.levelMaintenance'), value: 'maintenance' },
]

function openCreate() {
  form.value = { title: '', content: '', level: 'normal', pinned: false }
  showCreate.value = true
}

function submitCreate() {
  if (!form.value.title || !form.value.content) {
    message.warning(t('admin.announcements.titlePh'))
    return
  }
  list.value.unshift({
    id: 'a-' + Math.random().toString(36).slice(2, 8),
    title: form.value.title,
    content: form.value.content,
    level: form.value.level,
    pinned: form.value.pinned,
    publishedAt: new Date().toISOString(),
    status: 'draft',
  })
  showCreate.value = false
  message.success(t('admin.announcements.saved'))
}
</script>

<template>
  <div class="admin-page">
    <header class="admin-head with-action">
      <div>
        <h1 class="title">{{ t('admin.announcements.title') }}</h1>
        <p class="subtitle">{{ t('admin.announcements.subtitle') }}</p>
      </div>
      <n-button type="primary" @click="openCreate">
        <template #icon><n-icon :component="AddOutline" /></template>
        {{ t('admin.announcements.create') }}
      </n-button>
    </header>

    <n-card :bordered="false" class="list-card">
      <n-empty v-if="!sorted.length" :description="t('admin.announcements.empty')" />
      <n-list v-else>
        <n-list-item v-for="a in sorted" :key="a.id">
          <div class="anno-head">
            <div class="anno-title">
              <n-icon :component="MegaphoneOutline" v-if="a.pinned" class="anno-pin" />
              <strong>{{ a.title }}</strong>
              <n-tag size="small" :type="levelTag[a.level].type" :bordered="false" class="anno-level">
                {{ t(levelTag[a.level].key) }}
              </n-tag>
              <n-tag v-if="a.status === 'draft'" size="small" type="default" :bordered="false">
                {{ t('admin.announcements.statusDraft') }}
              </n-tag>
            </div>
            <span class="anno-time">{{ formatDateTime(a.publishedAt) }}</span>
          </div>
          <p class="anno-body">{{ a.content }}</p>
          <div class="anno-foot">
            <n-space :size="10">
              <n-switch v-model:value="a.pinned" size="small" />
              <span class="anno-foot-label">{{ t('admin.announcements.pinned') }}</span>
            </n-space>
            <n-button size="small" tertiary @click="togglePublish(a)">
              {{ a.status === 'published' ? t('admin.announcements.unpublished') : t('admin.announcements.publish') }}
            </n-button>
          </div>
        </n-list-item>
      </n-list>
    </n-card>

    <n-modal
      v-model:show="showCreate"
      :title="t('admin.announcements.create')"
      preset="card"
      style="width: 520px; max-width: 92vw"
    >
      <n-form label-placement="top">
        <n-form-item :label="t('admin.announcements.title')">
          <n-input v-model:value="form.title" :placeholder="t('admin.announcements.titlePh')" />
        </n-form-item>
        <n-form-item :label="t('admin.announcements.level')">
          <n-select v-model:value="form.level" :options="levelOptions" />
        </n-form-item>
        <n-form-item :label="t('admin.announcements.content')">
          <n-input
            v-model:value="form.content"
            type="textarea"
            :autosize="{ minRows: 4, maxRows: 8 }"
            :placeholder="t('admin.announcements.contentPh')"
          />
        </n-form-item>
        <n-form-item>
          <n-space :size="10">
            <n-switch v-model:value="form.pinned" />
            <span>{{ t('admin.announcements.pinned') }}</span>
          </n-space>
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="modal-foot">
          <n-button @click="showCreate = false">{{ t('admin.announcements.statusDraft') }}</n-button>
          <n-button type="primary" @click="submitCreate">{{ t('admin.announcements.save') }}</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>
