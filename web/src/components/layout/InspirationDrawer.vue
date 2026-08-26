<script setup lang="ts">
import { ref, computed } from 'vue'
import { useMessage } from 'naive-ui'
import { NIcon } from 'naive-ui'
import { FlameOutline, BulbOutline, RefreshOutline, CloseOutline, PulseOutline } from '@vicons/ionicons5'

import { materials } from '@/mock/materials'
import { trendingTopics } from '@/mock/trending'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ 'update:show': [boolean] }>()

const message = useMessage()

const tab = ref<'hot' | 'mine' | 'gen'>('hot')

const hot = trendingTopics.slice(0, 5)
const mine = materials.slice(0, 5)

const generators = [
  '一个社恐在AI时代成为职场整顿先锋',
  '人类最后的记忆被AI格式化，一个人决定反抗',
  '雨夜便利店，所有客人都带着说不出口的秘密',
]

function close() {
  emit('update:show', false)
}

function useItem(text: string) {
  message.success('已放进灵感暂存区，回到编辑器就能用～')
}

function writeIn(text: string) {
  message.success('已写入当前工作区草稿开头。')
}
</script>

<template>
  <transition name="slide">
    <aside v-if="show" class="insp-drawer">
      <header class="flex items-center justify-between px-5 py-[18px] border-b border-line">
        <h3 class="text-[19px] m-0 flex items-center gap-2">
          <n-icon :component="FlameOutline" class="text-[18px] text-rose-500" /> 灵感发现
        </h3>
        <button class="insp-x" @click="close"><n-icon :component="CloseOutline" class="text-[15px]" /> 关闭</button>
      </header>

      <div class="insp-tabs">
        <button :class="{ on: tab === 'hot' }" @click="tab = 'hot'">今日热梗</button>
        <button :class="{ on: tab === 'mine' }" @click="tab = 'mine'">素材库</button>
        <button :class="{ on: tab === 'gen' }" @click="tab = 'gen'">灵感生成器</button>
      </div>

      <div class="insp-body mf-scroll">
        <template v-if="tab === 'hot'">
          <p class="insp-sec flex items-center gap-1.5">
            <n-icon :component="FlameOutline" class="text-[14px] text-rose-500" /> 热搜榜 · 只取能写进故事的那一点
          </p>
          <div v-for="(m, i) in hot" :key="m.id" class="insp-trend">
            <div class="insp-rank">#{{ i + 1 }}</div>
            <div class="insp-main">
              <p class="insp-t-title">{{ m.title }} <span class="insp-heat flex items-center gap-1"><n-icon :component="FlameOutline" class="text-[12px]" /> {{ m.heat }}</span></p>
              <p class="insp-desc">{{ m.description }}</p>
              <p class="insp-tip">建议场景：{{ m.tags.join('、') }}</p>
            </div>
            <button class="insp-use" @click="useItem(m.title)">使用</button>
          </div>
        </template>

        <template v-else-if="tab === 'mine'">
          <p class="insp-sec flex items-center gap-1.5">
            <n-icon :component="PulseOutline" class="text-[14px] text-amber-500" /> 我的素材 · 工作区「我的书房」
          </p>
          <div v-for="m in mine" :key="m.id" class="insp-trend">
            <div class="insp-rank insp-rank-soft">·</div>
            <div class="insp-main">
              <p class="insp-t-title">{{ m.title }}</p>
              <p class="insp-desc">{{ m.content }}</p>
            </div>
            <button class="insp-use" @click="useItem(m.title)">使用</button>
          </div>
        </template>

        <template v-else>
          <p class="insp-sec flex items-center gap-1.5">
            <n-icon :component="BulbOutline" class="text-[14px] text-amber-500" /> 创作灵感 · 写不出时，先挑一句
          </p>
          <div v-for="g in generators" :key="g" class="insp-gen">
            <p>{{ g }}</p>
            <button class="insp-use" @click="writeIn(g)">写入</button>
          </div>
          <button class="insp-regen" @click="message.info('已为你换一批灵感～')">
            <n-icon :component="RefreshOutline" class="text-[14px]" /> 换一批
          </button>
        </template>
      </div>
    </aside>
  </transition>
</template>
