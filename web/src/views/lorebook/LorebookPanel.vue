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
import { storeToRefs } from 'pinia'
import { useNovelStore } from '@/stores/novel'
import UserAvatar from '@/components/common/UserAvatar.vue'
import { characters, worlds, foreshadows } from '@/mock'
import type { Character, WorldSetting, Foreshadow, ForeshadowStatus } from '@/types'

const novelStore = useNovelStore()
const { novels } = storeToRefs(novelStore)

const novelFilter = ref<string>('all')

const novelOptions = computed(() => [
  { label: '全部作品', value: 'all' },
  ...novels.value.map((n) => ({ label: n.title, value: n.id })),
])

const filteredCharacters = computed(() =>
  novelFilter.value === 'all'
    ? characters
    : characters.filter((c) => c.novelId === novelFilter.value),
)
const filteredWorlds = computed(() =>
  novelFilter.value === 'all' ? worlds : worlds.filter((w) => w.novelId === novelFilter.value),
)
const filteredForeshadows = computed(() =>
  novelFilter.value === 'all'
    ? foreshadows
    : foreshadows.filter((f) => f.novelId === novelFilter.value),
)

const roleTypeMap: Record<Character['role'], 'error' | 'info' | 'warning' | 'default'> = {
  主角: 'error',
  配角: 'info',
  反派: 'warning',
  龙套: 'default',
}

const foreshadowMeta: Record<ForeshadowStatus, { label: string; type: 'default' | 'info' | 'success' }> = {
  planted: { label: '已埋设', type: 'default' },
  revealing: { label: '揭示中', type: 'info' },
  resolved: { label: '已回收', type: 'success' },
}

onMounted(() => {
  if (!novels.value.length) novelStore.loadNovels()
})
</script>

<template>
  <div class="page">
    <header class="page__head">
      <div>
        <h1 class="page__title">设定集</h1>
        <p class="page__sub">把世界的每一个零件都收拢起来，让它真实可触。</p>
      </div>
      <n-select v-model:value="novelFilter" :options="novelOptions" style="width: 200px" />
    </header>

    <n-tabs type="line" animated>
      <n-tab-pane name="character" tab="角色">
        <n-grid v-if="filteredCharacters.length" :cols="3" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
          <n-gi v-for="c in filteredCharacters" :key="c.id" span="3 s:2 l:1">
            <n-card :bordered="false" class="lore-card">
              <div class="char__head">
                <UserAvatar :name="c.name" :color="c.avatarColor" :size="40" />
                <div>
                  <div class="char__name">{{ c.name }}</div>
                  <n-space :size="6">
                    <n-tag size="small" :bordered="false" :type="roleTypeMap[c.role]">{{ c.role }}</n-tag>
                    <n-text v-if="c.age" depth="3" style="font-size: 12px">· {{ c.age }} 岁</n-text>
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
        <n-empty v-else description="该作品下暂无角色" />
      </n-tab-pane>

      <n-tab-pane name="world" tab="世界观">
        <n-grid v-if="filteredWorlds.length" :cols="3" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
          <n-gi v-for="w in filteredWorlds" :key="w.id" span="3 s:2 l:1">
            <n-card :bordered="false" class="lore-card">
              <div class="world__head">
                <span class="world__name">{{ w.name }}</span>
                <n-tag size="small" :bordered="false" type="success">{{ w.category }}</n-tag>
              </div>
              <p class="lore__summary">{{ w.summary }}</p>
              <p class="world__details">{{ w.details }}</p>
            </n-card>
          </n-gi>
        </n-grid>
        <n-empty v-else description="该作品下暂无世界观设定" />
      </n-tab-pane>

      <n-tab-pane name="foreshadow" tab="伏笔">
        <n-grid v-if="filteredForeshadows.length" :cols="2" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
          <n-gi v-for="f in filteredForeshadows" :key="f.id" span="2 s:1">
            <n-card :bordered="false" class="lore-card">
              <div class="fs__head">
                <span class="fs__chapter">将于 · {{ f.revealChapter }}</span>
                <n-tag size="small" :bordered="false" :type="foreshadowMeta[f.status].type">{{
                  foreshadowMeta[f.status].label
                }}</n-tag>
              </div>
              <p class="lore__summary">{{ f.clue }}</p>
              <p class="fs__note">💡 {{ f.note }}</p>
            </n-card>
          </n-gi>
        </n-grid>
        <n-empty v-else description="该作品下暂无伏笔" />
      </n-tab-pane>
    </n-tabs>
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
.lore-card {
  height: 100%;
}
.char__head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 10px;
}
.char__name {
  font-weight: 600;
  font-size: 15px;
  color: var(--mf-text);
}
.lore__summary {
  color: var(--mf-text-2);
  font-size: 13px;
  line-height: 1.7;
  margin: 0 0 10px;
}
.world__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 10px;
}
.world__name {
  font-weight: 600;
  font-size: 15px;
  color: var(--mf-text);
}
.world__details {
  color: var(--mf-text-3);
  font-size: 12px;
  line-height: 1.7;
  margin: 0;
}
.fs__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 10px;
}
.fs__chapter {
  font-size: 12px;
  color: var(--mf-text-3);
}
.fs__note {
  color: var(--mf-text-3);
  font-size: 12px;
  line-height: 1.7;
  margin: 0;
}
</style>
