<template>
  <div class="login-wrap">
    <div class="login-card glass">
      <div class="login-title">登录 FileHub</div>
      <div class="login-subtitle">安全管理你的私有文件</div>
      <el-form @submit.prevent="submit">
        <el-form-item label="用户名">
          <el-input v-model="username" placeholder="admin" autocomplete="username" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="password" placeholder="输入密码" type="password" autocomplete="current-password" />
        </el-form-item>
        <el-button type="primary" class="full" :loading="loading" @click="submit">登录</el-button>
      </el-form>
      <div class="login-hint">使用 JWT 登录，24h 自动续期</div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { login } from '../api/auth'
import { setTokens } from '../store/auth'
import { ElMessage } from 'element-plus'

const username = ref('admin')
const password = ref('')
const loading = ref(false)
const route = useRoute()
const router = useRouter()

const submit = async () => {
  loading.value = true
  try {
    const response = await login(username.value, password.value)
    const data = response.data?.data
    if (!data?.access_token) throw new Error('login failed')
    setTokens(data.access_token, data.refresh_token)
    router.push(route.query.redirect || '/')
  } catch {
    ElMessage.error('登录失败')
  } finally {
    loading.value = false
  }
}
</script>
