<script setup lang="ts">
import { reactive, ref, computed } from 'vue'
import { NModal, NForm, NFormItem, NInput, NSelect, NButton, NDynamicTags, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useNovelStore } from '@/stores/novel'
import { useUserStore } from '@/stores/system/user'
import { NOVEL_GENRES } from './constants'
import type { CreateNovelPayload } from '@/types'

const { t } = useI18n()
const show = defineModel<boolean>('show', { required: true })
const emit = defineEmits<{ (e: 'created'): void }>()

const message = useMessage()
const novelStore = useNovelStore()
const userStore = useUserStore()

const form = reactive<CreateNovelPayload>({ title: '', description: '', genre: '', tags: [] })
const submitting = ref(false)

const genreOptions = computed(() => NOVEL_GENRES.map((g) => ({ label: t(g.labelKey), value: g.value })))

const rules = {
  title: { required: true, message: () => t('novel.createWarnName'), trigger: 'blur' },
  genre: { required: true, message: () => t('novel.createWarnType'), trigger: 'change' },
}

function close() {
  show.value = false
  form.title = ''
  form.description = ''
  form.genre = ''
  form.tags = []
}

async function submit() {
  if (!form.title.trim()) {
    message.warning(t('novel.createWarnName'))
    return
  }
  if (!form.genre) {
    message.warning(t('novel.createWarnType'))
    return
  }
  submitting.value = true
  try {
    await novelStore.createNovel({ ...form, title: form.title.trim() }, userStore.user?.id || 'u_demo')
    message.success(t('novel.createSuccess'))
    emit('created')
    close()
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="t('novel.createTitle')"
    style="width: 520px; max-width: 92vw"
    :bordered="false"
  >
    <n-form :model="form" :rules="rules" label-placement="top">
      <n-form-item :label="t('novel.createProjectName')" path="title">
        <n-input v-model:value="form.title" :placeholder="t('novel.createProjectNamePlaceholder')" />
      </n-form-item>
      <n-form-item :label="t('novel.createType')" path="genre">
        <n-select v-model:value="form.genre" :options="genreOptions" :placeholder="t('novel.createTypePlaceholder')" />
      </n-form-item>
      <n-form-item :label="t('novel.createIntro')" path="description">
        <n-input
          v-model:value="form.description"
          type="textarea"
          :placeholder="t('novel.createIntroPlaceholder')"
          :autosize="{ minRows: 3, maxRows: 6 }"
        />
      </n-form-item>
      <n-form-item :label="t('novel.createTagsLabel')">
        <n-dynamic-tags v-model:value="form.tags" />
      </n-form-item>
    </n-form>
    <template #footer>
      <div class="flex justify-end gap-3">
        <n-button quaternary @click="close">{{ t('common.cancel') }}</n-button>
        <n-button type="primary" :loading="submitting" @click="submit">{{ t('novel.createSubmit') }}</n-button>
      </div>
    </template>
  </n-modal>
</template>

