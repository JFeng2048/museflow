<script setup lang="ts">
import { ref, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage } from 'naive-ui'
import { NIcon } from 'naive-ui'
import { FlameOutline, BulbOutline, RefreshOutline } from '@vicons/ionicons5'
import { materials } from '@/mock/materials'
import { trendingTopics } from '@/mock/trending'
import { INSPIRATION_GENERATORS } from './constants'

const { t } = useI18n()
const message = useMessage()

const tab = ref<'hot' | 'mine' | 'gen'>('hot')

const hot = trendingTopics
const mine = materials.slice(0, 6)

const generators = INSPIRATION_GENERATORS

function useItem(text: string) {
  message.success(t('inspiration.usedToast'))
}
function writeIn(text: string) {
  message.success(t('inspiration.writtenToast'))
}
</script>

<template>
  <div class="px-10 py-8 max-w-[1080px] mx-auto">
    <header class="flex justify-between items-end gap-5 flex-wrap mb-6.5">
      <div>
        <p class="text-ink-muted m-0 flex items-center gap-1.5">
          <n-icon :component="FlameOutline" class="text-[15px] text-rose-500" /> {{ t('inspiration.eyebrow') }}
        </p>
        <h2 class="text-[28px] my-1">{{ t('inspiration.hero') }}</h2>
        <p class="text-ink-muted text-[14px] m-0">{{ t('inspiration.heroSub') }}</p>
      </div>
      <div class="insp-tabs">
        <button :class="{ on: tab === 'hot' }" @click="tab = 'hot'">{{ t('inspiration.tabHot') }}</button>
        <button :class="{ on: tab === 'mine' }" @click="tab = 'mine'">{{ t('inspiration.tabMine') }}</button>
        <button :class="{ on: tab === 'gen' }" @click="tab = 'gen'">{{ t('inspiration.tabGen') }}</button>
      </div>
    </header>

    <div v-if="tab === 'hot'" class="insp-grid">
      <article v-for="(m, i) in hot" :key="m.id" class="insp-card">
        <div class="insp-rank">#{{ i + 1 }}</div>
        <div class="insp-body">
          <p class="insp-title">{{ m.title }} <span class="insp-heat flex items-center gap-1"><n-icon :component="FlameOutline" class="text-[13px]" /> {{ m.heat }}</span></p>
          <p class="insp-desc">{{ m.description }}</p>
          <p class="insp-tip">{{ t('inspiration.suggestion') }}：{{ m.tags.join('、') }}</p>
        </div>
        <button class="insp-use" @click="useItem(m.title)">{{ t('inspiration.use') }}</button>
      </article>
    </div>

    <div v-else-if="tab === 'mine'" class="insp-grid">
      <article v-for="m in mine" :key="m.id" class="insp-card">
        <div class="insp-rank insp-rank-soft">·</div>
        <div class="insp-body">
          <p class="insp-title">{{ m.title }}</p>
          <p class="insp-desc">{{ m.content }}</p>
        </div>
        <button class="insp-use" @click="useItem(m.title)">{{ t('inspiration.use') }}</button>
      </article>
    </div>

    <div v-else class="max-w-[640px]">
      <p class="text-[13px] text-ink-muted mt-1 mb-3.5 flex items-center gap-1.5">
        <n-icon :component="BulbOutline" class="text-[15px] text-amber-500" /> {{ t('inspiration.genHint') }}
      </p>
      <div v-for="g in generators" :key="g" class="insp-gen">
        <p>{{ g }}</p>
        <button class="insp-use" @click="writeIn(g)">{{ t('inspiration.writeIn') }}</button>
      </div>
      <button class="insp-regen" @click="message.info(t('inspiration.regenToast'))">
        <n-icon :component="RefreshOutline" class="text-[14px]" /> {{ t('inspiration.regen') }}
      </button>
    </div>
  </div>
</template>
