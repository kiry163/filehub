<template>
  <aside class="sidebar glass">
    <nav class="side-nav">
      <el-button text class="nav-item" :class="isActive('/')" @click="go('/')">
        <el-icon><Document /></el-icon>
        文件
      </el-button>
      <el-button text class="nav-item" :class="isActive('/upload')" @click="go('/upload')">
        <el-icon><Upload /></el-icon>
        上传
      </el-button>
      <el-button text class="nav-item" :class="isActive('/file')" @click="goLatest">
        <el-icon><Files /></el-icon>
        详情
      </el-button>
    </nav>
    <div class="sidebar-foot">
      <div class="stat">
        <div class="stat-label">存储使用</div>
        <div class="stat-value">-</div>
        <div class="progress"><span style="width: 0%"></span></div>
      </div>
    </div>
  </aside>
</template>

<script setup>
import { useRoute, useRouter } from 'vue-router'
import { Document, Upload, Files } from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()

const isActive = (path) => (route.path.startsWith(path) ? 'active' : '')
const go = (path) => router.push(path)
const goLatest = () => router.push('/')
</script>
