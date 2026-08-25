<script setup lang="ts">
import { reactive } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const route = useRoute()
const message = useMessage()
const userStore = useUserStore()

const form = reactive({ username: 'demo', password: '123456' })

async function onSubmit() {
  try {
    await userStore.login(form)
    message.success('登录成功')
    const redirect = (route.query.redirect as string) || '/dashboard'
    router.push(redirect)
  } catch {
    message.error('登录失败，请重试')
  }
}
</script>

<template>
  <n-form class="auth-form" :model="form" @submit.prevent="onSubmit">
    <h2>欢迎回来</h2>
    <p class="sub">登录以继续你的创作旅程</p>
    <n-alert :show-icon="false" type="default" class="mock-hint">
      演示账号 · 用户名 <b>demo</b> · 密码 <b>123456</b>（已自动填入，可直接登录）
    </n-alert>
    <n-form-item label="用户名">
      <n-input v-model:value="form.username" placeholder="请输入用户名" />
    </n-form-item>
    <n-form-item label="密码">
      <n-input
        v-model:value="form.password"
        type="password"
        show-password-on="click"
        placeholder="请输入密码"
      />
    </n-form-item>
    <n-button type="primary" block attr-type="submit" :loading="userStore.loading">
      登录
    </n-button>
    <p class="switch">
      还没有账号？
      <router-link to="/auth/register">立即注册</router-link>
    </p>
  </n-form>
</template>

<style scoped>
.auth-form {
  width: 320px;
}
.auth-form h2 {
  margin: 0 0 4px;
  font-size: 22px;
}
.sub {
  color: #8a9099;
  font-size: 13px;
  margin: 0 0 18px;
}
.mock-hint {
  margin: 0 0 16px;
  font-size: 13px;
  color: var(--mf-text-2);
  line-height: 1.6;
}
.switch {
  text-align: center;
  font-size: 13px;
  color: #6b7280;
  margin-top: 14px;
}
</style>
