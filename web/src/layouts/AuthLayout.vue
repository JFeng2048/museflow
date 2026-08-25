<script setup lang="ts">
import AppLogo from '@/components/common/AppLogo.vue'
</script>

<template>
  <div class="auth-layout">
    <div class="panel">
      <div class="brand">
        <AppLogo />
      </div>
      <router-view />
      <p class="foot">MuseFlow · 从灵感到发布的全链路创作工具</p>
    </div>
    <div class="art">
      <div class="quote">
        <p class="big">“写下第一行字，灵感便开始流动。”</p>
        <p class="sub">让 AI 陪你完成从大纲到发布的每一步。</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.auth-layout {
  display: grid;
  grid-template-columns: 1fr 1fr;
  height: 100vh;
}
.panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px;
  gap: 24px;
  background: var(--mf-surface);
}
.brand {
  margin-bottom: 8px;
}
.foot {
  color: var(--mf-text-3);
  font-size: 12px;
}
.art {
  background: linear-gradient(150deg, #4c6ef5, #8a6cff);
  display: grid;
  place-items: center;
  color: #fff;
}
.quote {
  max-width: 360px;
  padding: 32px;
}
.big {
  font-size: 26px;
  font-weight: 600;
  line-height: 1.5;
  margin: 0 0 12px;
}
.sub {
  font-size: 15px;
  opacity: 0.85;
  margin: 0;
}
@media (max-width: 860px) {
  .art {
    display: none;
  }
  .auth-layout {
    grid-template-columns: 1fr;
  }
}
</style>
