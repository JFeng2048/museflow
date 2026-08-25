<script setup lang="ts">
import { reactive, ref } from 'vue'
import { NModal, NForm, NFormItem, NInput, NSelect, NButton, NDynamicTags, useMessage } from 'naive-ui'
import { useNovelStore } from '@/stores/novel'
import type { CreateNovelPayload } from '@/types'

const show = defineModel<boolean>('show', { required: true })
const emit = defineEmits<{ (e: 'created'): void }>()

const message = useMessage()
const novelStore = useNovelStore()

const form = reactive<CreateNovelPayload>({ title: '', description: '', genre: '', tags: [] })
const submitting = ref(false)

const genreOptions = ['科幻', '奇幻', '历史', '悬疑', '轻小说', '都市', '言情', '武侠'].map((g) => ({
  label: g,
  value: g,
}))

const rules = {
  title: { required: true, message: '请输入项目名称', trigger: 'blur' },
  genre: { required: true, message: '请选择类型', trigger: 'change' },
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
    message.warning('请输入项目名称')
    return
  }
  submitting.value = true
  try {
    await novelStore.createNovel({ ...form, title: form.title.trim() })
    message.success('项目已创建')
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
    title="新建项目"
    style="width: 520px; max-width: 92vw"
    :bordered="false"
  >
    <n-form :model="form" :rules="rules" label-placement="top">
      <n-form-item label="项目名称" path="title">
        <n-input v-model:value="form.title" placeholder="例如：星海拾遗者" />
      </n-form-item>
      <n-form-item label="类型" path="genre">
        <n-select v-model:value="form.genre" :options="genreOptions" placeholder="选择作品类型" />
      </n-form-item>
      <n-form-item label="简介" path="description">
        <n-input
          v-model:value="form.description"
          type="textarea"
          placeholder="一句话描述这本小说的核心设定或卖点"
          :autosize="{ minRows: 3, maxRows: 6 }"
        />
      </n-form-item>
      <n-form-item label="标签">
        <n-dynamic-tags v-model:value="form.tags" />
      </n-form-item>
    </n-form>
    <template #footer>
      <div class="modal-footer">
        <n-button quaternary @click="close">取消</n-button>
        <n-button type="primary" :loading="submitting" @click="submit">创建项目</n-button>
      </div>
    </template>
  </n-modal>
</template>

<style scoped>
.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
