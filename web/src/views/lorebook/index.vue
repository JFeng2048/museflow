<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import {
  NTabs,
  NTabPane,
  NGrid,
  NGi,
  NCard,
  NTag,
  NSelect,
  NEmpty,
  NSpace,
  NText,
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { NIcon } from 'naive-ui'
import { BulbOutline } from '@vicons/ionicons5'
import { storeToRefs } from 'pinia'
import { useNovelStore } from '@/stores/novel'

import UserAvatar from '@/components/common/UserAvatar.vue'
import { characters, worlds, foreshadows } from '@/mock'
import type { Character, WorldSetting, Foreshadow, ForeshadowStatus } from '@/types'
import { ROLE_META, FORESHADOW_META } from './constants'

const { t } = useI18n()
const novelStore = useNovelStore()

const { allNovels } = storeToRefs(novelStore)

const novelFilter = ref<string>('all')

const workspaceNovelIds = computed(() =>
  new Set(novelStore.byUser('u_demo').map((n) => n.id)),
)

const novelOptions = computed(() => [
  { label: t('lore.allWorkspace'), value: 'all' },
  ...novelStore.byUser('u_demo').map((n) => ({ label: n.title, value: n.id })),
])

const filteredCharacters = computed(() => {
  const list = characters.filter((c) => workspaceNovelIds.value.has(c.novelId))
  return novelFilter.value === 'all' ? list : list.filter((c) => c.novelId === novelFilter.value)
})
const filteredWorlds = computed(() => {
  const list = worlds.filter((w) => workspaceNovelIds.value.has(w.novelId))
  return novelFilter.value === 'all' ? list : list.filter((w) => w.novelId === novelFilter.value)
})
const filteredForeshadows = computed(() => {
  const list = foreshadows.filter((f) => workspaceNovelIds.value.has(f.novelId))
  return novelFilter.value === 'all' ? list : list.filter((f) => f.novelId === novelFilter.value)
})

onMounted(() => {
  if (!allNovels.value.length) novelStore.loadNovels()
})
</script>

<template>
  <div class="flex flex-col gap-4">
    <header class="flex items-start justify-between">
      <div>
        <h1 class="text-[22px] font-semibold m-0">{{ t('lore.title') }}</h1>
        <p class="text-ink-muted mt-1 mb-0">{{ t('lore.subtitle') }}</p>
      </div>
      <n-select v-model:value="novelFilter" :options="novelOptions" class="w-[200px]" />
    </header>

    <n-tabs type="line" animated>
      <n-tab-pane name="character" :tab="t('lore.tabCharacter')">
        <n-grid v-if="filteredCharacters.length" :cols="3" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
          <n-gi v-for="c in filteredCharacters" :key="c.id" span="3 s:2 l:1">
            <n-card :bordered="false" class="h-full">
              <div class="flex items-center gap-3 mb-2.5">
                <UserAvatar :name="c.name" :color="c.avatarColor" :size="40" />
                <div>
                  <div class="font-semibold text-[15px] text-ink">{{ c.name }}</div>
                  <n-space :size="6">
                    <n-tag size="small" :bordered="false" :type="ROLE_META[c.role]?.type || 'default'">{{
                      t(ROLE_META[c.role]?.labelKey || 'lore.roleMinor')
                    }}</n-tag>
                    <n-text v-if="c.age" depth="3" style="font-size: 12px">· {{ c.age }} {{ t('common.ageUnit') }}</n-text>
                  </n-space>
                </div>
              </div>
              <p class="lore__summary">{{ c.summary }}</p>
              <n-space :size="6">
                <n-tag v-for="t in c.traits" :key="t" size="tiny" :bordered="false" type="info">{{
                  t
                }}</n-tag>
              </n-space>
            </n-card>
          </n-gi>
        </n-grid>
        <n-empty v-else :description="t('lore.emptyCharacter')" />
      </n-tab-pane>

      <n-tab-pane name="world" :tab="t('lore.tabWorld')">
        <n-grid v-if="filteredWorlds.length" :cols="3" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
          <n-gi v-for="w in filteredWorlds" :key="w.id" span="3 s:2 l:1">
            <n-card :bordered="false" class="h-full">
              <div class="flex items-center justify-between gap-2 mb-2.5">
                <span class="font-semibold text-[15px] text-ink">{{ w.name }}</span>
                <n-tag size="small" :bordered="false" type="success">{{ w.category }}</n-tag>
              </div>
              <p class="lore__summary">{{ w.summary }}</p>
              <p class="lore-world-details">{{ w.details }}</p>
            </n-card>
          </n-gi>
        </n-grid>
        <n-empty v-else :description="t('lore.emptyWorld')" />
      </n-tab-pane>

      <n-tab-pane name="foreshadow" :tab="t('lore.tabForeshadow')">
        <n-grid v-if="filteredForeshadows.length" :cols="2" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
          <n-gi v-for="f in filteredForeshadows" :key="f.id" span="2 s:1">
            <n-card :bordered="false" class="h-full">
              <div class="flex items-center justify-between gap-2 mb-2.5">
                <span class="text-[12px] text-ink-muted">{{ t('lore.revealLabel') }} · {{ f.revealChapter }}</span>
                <n-tag size="small" :bordered="false" :type="FORESHADOW_META[f.status].type">{{
                  t(FORESHADOW_META[f.status].labelKey)
                }}</n-tag>
              </div>
              <p class="lore__summary">{{ f.clue }}</p>
              <p class="lore-fs-note"><n-icon :component="BulbOutline" class="text-[13px] mr-1" />{{ f.note }}</p>
            </n-card>
          </n-gi>
        </n-grid>
        <n-empty v-else :description="t('lore.emptyForeshadow')" />
      </n-tab-pane>
    </n-tabs>
  </div>
</template>

