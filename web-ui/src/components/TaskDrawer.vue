<template>
  <el-drawer v-model="state.open" size="360px" title="任务中心" append-to-body>
    <div class="drawer-subtitle">上传 / 下载进度实时更新</div>
    <div class="drawer-actions">
      <el-button text @click="clearCompleted">清理已完成</el-button>
    </div>
    <div class="drawer-tabs">
      <el-button text :class="filter === 'all' ? 'active' : ''" @click="filter = 'all'">全部</el-button>
      <el-button text :class="filter === 'upload' ? 'active' : ''" @click="filter = 'upload'">上传中</el-button>
      <el-button text :class="filter === 'download' ? 'active' : ''" @click="filter = 'download'">下载中</el-button>
      <el-button text :class="filter === 'done' ? 'active' : ''" @click="filter = 'done'">已完成</el-button>
    </div>
    <div class="drawer-list">
      <div v-for="task in filtered" :key="task.id" class="drawer-item">
        <div class="file-name">{{ task.name }}</div>
        <div class="file-meta">{{ label(task) }}</div>
        <el-progress :percentage="task.progress" :status="status(task)" />
      </div>
      <div v-if="filtered.length === 0" class="drawer-empty">暂无任务</div>
    </div>
  </el-drawer>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useTaskCenter, clearCompleted } from '../store/taskCenter'

const state = useTaskCenter()
const filter = ref('all')

const filtered = computed(() => {
  if (filter.value === 'all') return state.tasks
  if (filter.value === 'done') return state.tasks.filter((task) => task.status === 'done')
  return state.tasks.filter((task) => task.type === filter.value && task.status === 'running')
})

const status = (task) => (task.status === 'done' ? 'success' : task.status === 'error' ? 'exception' : '')
const label = (task) => {
  const stateText = task.status === 'done' ? '已完成' : task.status === 'error' ? '失败' : task.type === 'upload' ? '上传中' : '下载中'
  return `${stateText}${task.sizeLabel ? ' · ' + task.sizeLabel : ''}`
}
</script>
