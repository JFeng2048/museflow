<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { NIcon } from 'naive-ui'
import { PersonOutline, ShieldCheckmarkOutline } from '@vicons/ionicons5'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{
  (e: 'update:show', v: boolean): void
  (e: 'choose', view: 'user' | 'admin'): void
}>()

const { t } = useI18n()

function close() {
  emit('update:show', false)
}
function choose(view: 'user' | 'admin') {
  emit('choose', view)
  close()
}
</script>

<template>
  <transition name="fade">
    <div v-if="props.show" class="identity-mask" @click.self="close">
      <div class="identity-card">
        <h3>{{ t('identity.title') }}</h3>
        <p class="identity-sub">{{ t('identity.subtitle') }}</p>
        <button class="identity-opt" type="button" @click="choose('user')">
          <span class="identity-ico user"><n-icon :component="PersonOutline" /></span>
          <span class="identity-text">
            <strong>{{ t('identity.userTitle') }}</strong>
            <small>{{ t('identity.userDesc') }}</small>
          </span>
        </button>
        <button class="identity-opt" type="button" @click="choose('admin')">
          <span class="identity-ico admin"><n-icon :component="ShieldCheckmarkOutline" /></span>
          <span class="identity-text">
            <strong>{{ t('identity.adminTitle') }}</strong>
            <small>{{ t('identity.adminDesc') }}</small>
          </span>
        </button>
      </div>
    </div>
  </transition>
</template>
