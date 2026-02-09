<template>
  <header class="topbar glass">
    <div class="brand" @click="goHome">
      <div class="logo">FH</div>
      <div class="brand-text">
        <div class="brand-name">FileHub</div>
        <div class="brand-subtitle">Private File Manager</div>
      </div>
    </div>
    <div class="top-actions">
      <el-input
        v-model="keyword"
        class="search"
        placeholder="搜索文件名"
        clearable
        @input="onSearch"
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
      <el-button type="primary" @click="goUpload">
        <el-icon><Upload /></el-icon>
        上传
      </el-button>
      <el-button @click="openTaskCenter">
        <el-icon><List /></el-icon>
        任务中心
      </el-button>
      <el-button text @click="logout">
        <el-icon><User /></el-icon>
        退出
      </el-button>
    </div>
  </header>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Search, Upload, User, List } from '@element-plus/icons-vue'
import { openTaskCenter } from '../store/taskCenter'
import { clearTokens } from '../store/auth'

const route = useRoute()
const router = useRouter()
const keyword = ref(route.query.keyword || '')
let timer

const onSearch = () => {
  window.clearTimeout(timer)
  timer = window.setTimeout(() => {
    if (route.path !== '/') {
      router.push({ path: '/', query: { keyword: keyword.value || undefined } })
      return
    }
    router.replace({ query: { ...route.query, keyword: keyword.value || undefined } })
  }, 300)
}

watch(
  () => route.query.keyword,
  (value) => {
    keyword.value = value || ''
  },
)

const goUpload = () => router.push('/upload')
const goHome = () => router.push('/')

const logout = () => {
  clearTokens()
  router.push('/login')
}
</script>
