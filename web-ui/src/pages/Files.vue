<template>
  <section class="page">
    <div class="section-header">
      <div>
        <h1>文件</h1>
        <p>集中管理上传文件，支持复制 filehub:// 链接与快速下载。</p>
      </div>
      <div class="toolbar">
        <el-button @click="reload">刷新</el-button>
        <el-button type="danger" plain disabled>批量删除</el-button>
      </div>
    </div>

    <div class="task-center glass">
      <div class="task-header">
        <div>
          <div class="task-title">任务中心</div>
          <div class="task-subtitle">上传/下载实时进度</div>
        </div>
        <el-button text @click="openTaskCenter">查看全部</el-button>
      </div>
      <div class="task-list">
        <div v-for="task in previewTasks" :key="task.id" class="task-item">
          <div>
            <div class="file-name">{{ task.name }}</div>
            <div class="file-meta">{{ taskLabel(task) }}</div>
          </div>
          <el-progress :percentage="task.progress" :status="taskStatus(task)" />
        </div>
        <div v-if="previewTasks.length === 0" class="task-empty">暂无任务</div>
      </div>
    </div>

    <div v-if="loading" class="loading">加载中...</div>

    <div v-if="!loading && groups.length === 0" class="empty">暂无文件</div>

    <div v-for="group in groups" :key="group.date" class="group">
      <div class="group-title">{{ group.date }}</div>
      <div v-for="file in group.files" :key="file.file_id" class="file-card glass">
        <div class="file-info">
          <div class="file-icon">
            <el-icon><Document /></el-icon>
          </div>
          <div>
            <div class="file-name">{{ file.original_name }}</div>
            <div class="file-meta">{{ formatSize(file.size) }} · {{ formatTime(file.created_at) }}</div>
          </div>
        </div>
        <div class="file-actions">
          <el-button text @click="copyLink(file)">复制链接</el-button>
          <el-button text @click="goDetail(file)">查看</el-button>
          <el-button text type="danger" @click="removeFile(file)">删除</el-button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Document } from '@element-plus/icons-vue'
import { listFiles, deleteFile } from '../api/files'
import { openTaskCenter, useTaskCenter } from '../store/taskCenter'
import { ElMessage, ElMessageBox } from 'element-plus'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const files = ref([])
const total = ref(0)

const state = useTaskCenter()
const previewTasks = computed(() => state.tasks.slice(0, 3))

const taskLabel = (task) => {
  const stateText = task.status === 'done' ? '已完成' : task.status === 'error' ? '失败' : task.type === 'upload' ? '上传中' : '下载中'
  return `${stateText}${task.sizeLabel ? ' · ' + task.sizeLabel : ''}`
}

const taskStatus = (task) => (task.status === 'done' ? 'success' : task.status === 'error' ? 'exception' : '')

const fetchFiles = async () => {
  loading.value = true
  try {
    const response = await listFiles({
      limit: 50,
      offset: 0,
      order: 'desc',
      keyword: route.query.keyword || undefined,
    })
    files.value = response.data?.data?.files || []
    total.value = response.data?.data?.total || 0
  } catch {
    ElMessage.error('获取文件失败')
  } finally {
    loading.value = false
  }
}

const groups = computed(() => {
  const map = new Map()
  files.value.forEach((file) => {
    const date = file.created_at?.slice(0, 10) || '未知日期'
    if (!map.has(date)) map.set(date, [])
    map.get(date).push(file)
  })
  return Array.from(map.entries()).map(([date, items]) => ({ date, files: items }))
})

const formatSize = (size) => {
  if (!size) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let value = size
  let idx = 0
  while (value > 1024 && idx < units.length - 1) {
    value /= 1024
    idx += 1
  }
  return `${value.toFixed(1)} ${units[idx]}`
}

const formatTime = (value) => value?.slice(11, 16) || '--'

const copyLink = async (file) => {
  const text = `filehub://${file.file_id}`
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制链接')
  } catch {
    ElMessage.error('复制失败')
  }
}

const goDetail = (file) => router.push(`/file/${file.file_id}`)

const removeFile = async (file) => {
  try {
    await ElMessageBox.confirm(`确认删除 ${file.original_name} ?`, '删除文件', { type: 'warning' })
    await deleteFile(file.file_id)
    ElMessage.success('已删除')
    fetchFiles()
  } catch {}
}

const reload = () => fetchFiles()

watch(
  () => route.query.keyword,
  () => fetchFiles(),
)

onMounted(fetchFiles)
</script>
