<script setup lang="ts">
import { reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const message = useMessage()
const userStore = useUserStore()

const form = reactive({ username: '', email: '', password: '', confirmPassword: '' })

async function onSubmit() {
  if (form.password !== form.confirmPassword) {
    message.warning('两次输入的密码不一致')
    return
  }
  try {
    await userStore.register(form)
    message.success('注册成功')
    router.push('/dashboard')
  } catch {
    message.error('注册失败，请重试')
  }
}
</script>

<template>
  <n-form class="auth-form" :model="form" @submit.prevent="onSubmit">
    <h2>创建账号</h2>
    <p class="sub">加入 MuseFlow，开启 AI 创作</p>
    <n-alert :show-icon="false" type="default" class="mock-hint">
      演示环境 · 注册信息仅保存在本地浏览器，可直接用演示账号登录
    </n-alert>
    <n-form-item label="用户名">
      <n-input v-model:value="form.username" placeholder="给自己起个名字" />
    </n-form-item>
    <n-form-item label="邮箱">
      <n-input v-model:value="form.email" placeholder="you@example.com" />
    </n-form-item>
    <n-form-item label="密码">
      <n-input
        v-model:value="form.password"
        type="password"
        show-password-on="click"
        placeholder="至少 6 位"
      />
    </n-form-item>
    <n-form-item label="确认密码">
      <n-input
        v-model:value="form.confirmPassword"
        type="password"
        show-password-on="click"
        placeholder="再次输入密码"
      />
    </n-form-item>
    <n-button type="primary" block attr-type="submit" :loading="userStore.loading">
      注册
    </n-button>
    <p class="switch">
      已有账号？
      <router-link to="/auth/login">返回登录</router-link>
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
