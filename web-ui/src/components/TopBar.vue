<template>
  <header class="topbar">
    <div class="brand" @click="goHome">
      <div class="logo">FH</div>
      <div class="brand-text">
        <div class="brand-name">FileHub</div>
        <div class="brand-subtitle">Secure File Control</div>
      </div>
    </div>
    <div class="top-actions">
      <el-button class="btn ghost" @click="openTaskCenter">
        <el-icon><List /></el-icon>
        任务中心
      </el-button>
      <el-button class="btn primary" @click="goUpload">
        <el-icon><Upload /></el-icon>
        上传
      </el-button>
      <el-button class="btn subtle" @click="handleAuth">
        <el-icon><User /></el-icon>
        {{ authLabel }}
      </el-button>
    </div>
  </header>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { Upload, User, List } from '@element-plus/icons-vue'
import { openTaskCenter } from '../store/taskCenter'
import { clearTokens, getAccessToken } from '../store/auth'

const router = useRouter()

const authLabel = computed(() => (getAccessToken() ? '退出' : '登录'))

const goUpload = () => router.push('/upload')
const goHome = () => router.push('/')

const handleAuth = () => {
  if (getAccessToken()) {
    clearTokens()
    router.push('/login')
  } else {
    router.push('/login')
  }
}
</script>
