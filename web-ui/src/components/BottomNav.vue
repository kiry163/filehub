<template>
  <nav class="bottom-nav glass" aria-label="移动端导航">
    <el-button text class="nav-item" :class="isActive('/')" @click="go('/')">
      <el-icon><Document /></el-icon>
      文件
    </el-button>
    <el-button text class="nav-item" :class="isActive('/upload')" @click="go('/upload')">
      <el-icon><Upload /></el-icon>
      上传
    </el-button>
    <el-button text class="nav-item" :class="isActive('/file')" @click="go('/')">
      <el-icon><Files /></el-icon>
      详情
    </el-button>
    <el-button text class="nav-item" @click="openTaskCenter">
      <el-icon><List /></el-icon>
      任务
    </el-button>
  </nav>
</template>

<script setup>
import { useRoute, useRouter } from 'vue-router'
import { Document, Upload, Files, List } from '@element-plus/icons-vue'
import { openTaskCenter } from '../store/taskCenter'

const route = useRoute()
const router = useRouter()

const isActive = (path) => (route.path.startsWith(path) ? 'active' : '')
const go = (path) => router.push(path)
</script>
