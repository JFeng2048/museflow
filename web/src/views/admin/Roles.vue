<script setup lang="ts">
import { ref, h, reactive, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NCard, NDataTable, NTag, NButton, NIcon, NModal, NForm, NFormItem,
  NInput, NCheckboxGroup, NCheckbox, NSpace, NEmpty, NSpin,
  useMessage, type DataTableColumns,
} from 'naive-ui'
import { AddOutline, KeyOutline } from '@vicons/ionicons5'
import {
  listRoles, createRole, updateRole, deleteRole, setRolePermissions, listPermissions,
} from '@/api/admin'
import type { AdminRole, AdminPermission } from '@/types/admin'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const message = useMessage()

const roles = ref<AdminRole[]>([])
const permissions = ref<AdminPermission[]>([])
const loading = ref(false)

/** 按资源分组的权限，便于在弹窗中分组勾选。 */
const groupedPermissions = computed(() => {
  const groups: Record<string, AdminPermission[]> = {}
  for (const p of permissions.value) {
    ;(groups[p.resource] ||= []).push(p)
  }
  return Object.entries(groups).map(([resource, items]) => ({ resource, items }))
})

async function load() {
  loading.value = true
  try {
    const [r, p] = await Promise.all([listRoles(), listPermissions()])
    roles.value = r
    permissions.value = p
  } catch (e: any) {
    message.error(e?.message || t('admin.roles.loadFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(load)

// ---------------- 新建 / 编辑角色 ----------------

const showEdit = ref(false)
const editingId = ref<number | null>(null)
const form = reactive({ code: '', name: '', description: '' })

function openCreate() {
  editingId.value = null
  form.code = ''
  form.name = ''
  form.description = ''
  showEdit.value = true
}

function openEdit(row: AdminRole) {
  editingId.value = row.id
  form.code = row.code
  form.name = row.name
  form.description = row.description
  showEdit.value = true
}

async function submitEdit() {
  if (!form.name) {
    message.warning(t('admin.roles.namePh'))
    return
  }
  try {
    if (editingId.value === null) {
      if (!form.code) {
        message.warning(t('admin.roles.codePh'))
        return
      }
      await createRole({ code: form.code, name: form.name, description: form.description })
      message.success(t('admin.roles.created'))
    } else {
      await updateRole(editingId.value, { name: form.name, description: form.description })
      message.success(t('admin.roles.updated'))
    }
    showEdit.value = false
    load()
  } catch (e: any) {
    message.error(e?.message || t('admin.roles.saveFailed'))
  }
}

async function onDelete(row: AdminRole) {
  try {
    await deleteRole(row.id)
    message.success(t('admin.roles.deleted'))
    load()
  } catch (e: any) {
    message.error(e?.message || t('admin.roles.deleteFailed'))
  }
}

// ---------------- 权限分配 ----------------

const showPerm = ref(false)
const permRoleId = ref<number | null>(null)
const permRoleName = ref('')
const checkedCodes = ref<string[]>([])
const permSaving = ref(false)
const permOptions = ref<string[]>([])

/**
 * 打开权限分配弹窗。
 * 后端没有提供「查询角色已拥有权限」的接口，这里以当前勾选状态为起点，
 * 首次打开时默认不带任何权限（覆盖式保存，需重新勾选）。
 */
function openPermissions(row: AdminRole) {
  permRoleId.value = row.id
  permRoleName.value = row.name
  checkedCodes.value = []
  permOptions.value = permissions.value.map((p) => p.code)
  showPerm.value = true
}

async function submitPermissions() {
  if (permRoleId.value === null) return
  permSaving.value = true
  try {
    await setRolePermissions(permRoleId.value, checkedCodes.value)
    message.success(t('admin.roles.permsSaved'))
    showPerm.value = false
  } catch (e: any) {
    message.error(e?.message || t('admin.roles.saveFailed'))
  } finally {
    permSaving.value = false
  }
}

const columns: DataTableColumns<AdminRole> = [
  { title: t('admin.roles.name'), key: 'name', width: 160 },
  { title: t('admin.roles.code'), key: 'code', width: 150, render: (row) => h('code', null, row.code) },
  { title: t('admin.roles.desc'), key: 'description', render: (row) => row.description || '-' },
  {
    title: t('admin.roles.type'), key: 'isSystem', width: 110,
    render: (row) =>
      h(NTag, { size: 'small', type: row.isSystem ? 'warning' : 'default', bordered: false }, {
        default: () => (row.isSystem ? t('admin.roles.system') : t('admin.roles.custom')),
      }),
  },
  {
    title: t('admin.roles.createdAt'), key: 'createdAt', width: 170,
    render: (row) => formatDateTime(row.createdAt),
  },
  {
    title: t('admin.roles.actions'), key: 'actions', width: 240,
    render: (row) =>
      h(NSpace, { size: 8 }, {
        default: () => [
          h(NButton, {
            size: 'small', tertiary: true,
            onClick: () => openPermissions(row),
            renderIcon: () => h(NIcon, { component: KeyOutline }),
          }, { default: () => t('admin.roles.perms') }),
          h(NButton, {
            size: 'small', tertiary: true, disabled: row.isSystem,
            onClick: () => openEdit(row),
          }, { default: () => t('admin.roles.edit') }),
          h(NButton, {
            size: 'small', tertiary: true, type: 'error', disabled: row.isSystem,
            onClick: () => onDelete(row),
          }, { default: () => t('admin.roles.delete') }),
        ],
      }),
  },
]
</script>

<template>
  <div class="admin-page">
    <header class="admin-head with-action">
      <div>
        <h1 class="title">{{ t('admin.roles.title') }}</h1>
        <p class="subtitle">{{ t('admin.roles.subtitle') }}</p>
      </div>
      <n-button type="primary" @click="openCreate">
        <template #icon><n-icon :component="AddOutline" /></template>
        {{ t('admin.roles.create') }}
      </n-button>
    </header>

    <n-card :bordered="false" class="table-card">
      <n-spin :show="loading">
        <n-empty v-if="!loading && !roles.length" :description="t('admin.roles.empty')" />
        <n-data-table v-else :columns="columns" :data="roles" :row-key="(r) => r.id" />
      </n-spin>
    </n-card>

    <!-- 新建 / 编辑角色 -->
    <n-modal
      v-model:show="showEdit"
      :title="editingId === null ? t('admin.roles.create') : t('admin.roles.edit')"
      preset="card"
      style="width: 440px; max-width: 92vw"
    >
      <n-form label-placement="left" :label-width="84">
        <n-form-item :label="t('admin.roles.code')">
          <n-input v-model:value="form.code" :disabled="editingId !== null" :placeholder="t('admin.roles.codePh')" />
        </n-form-item>
        <n-form-item :label="t('admin.roles.name')">
          <n-input v-model:value="form.name" :placeholder="t('admin.roles.namePh')" />
        </n-form-item>
        <n-form-item :label="t('admin.roles.desc')">
          <n-input v-model:value="form.description" type="textarea" :placeholder="t('admin.roles.descPh')" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="modal-foot">
          <n-button @click="showEdit = false">{{ t('admin.announcements.statusDraft') }}</n-button>
          <n-button type="primary" @click="submitEdit">{{ t('admin.roles.save') }}</n-button>
        </div>
      </template>
    </n-modal>

    <!-- 权限分配 -->
    <n-modal
      v-model:show="showPerm"
      :title="`${t('admin.roles.perms')} · ${permRoleName}`"
      preset="card"
      style="width: 620px; max-width: 92vw"
    >
      <n-checkbox-group v-model:value="checkedCodes">
        <div v-for="g in groupedPermissions" :key="g.resource" class="perm-group">
          <p class="perm-group-title">{{ g.resource }}</p>
          <n-space>
            <n-checkbox
              v-for="p in g.items"
              :key="p.code"
              :value="p.code"
              :label="p.name"
            />
          </n-space>
        </div>
      </n-checkbox-group>
      <template #footer>
        <div class="modal-foot">
          <n-button @click="showPerm = false">{{ t('admin.roles.cancel') }}</n-button>
          <n-button type="primary" :loading="permSaving" @click="submitPermissions">
            {{ t('admin.roles.save') }}
          </n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.perm-group {
  margin-bottom: 16px;
}
.perm-group-title {
  margin: 0 0 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--c-ink-soft);
}
</style>
