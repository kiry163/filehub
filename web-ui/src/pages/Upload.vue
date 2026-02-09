<template>
  <section class="page">
    <div class="section-header">
      <div>
        <h1>上传文件</h1>
        <p>拖拽或选择文件上传，上传完成后复制 filehub:// 链接。</p>
      </div>
    </div>

    <div class="upload-grid">
      <div class="upload-drop glass" @dragover.prevent @drop.prevent="onDrop">
        <div class="drop-icon">
          <el-icon><UploadFilled /></el-icon>
        </div>
        <div class="drop-title">拖拽文件到这里</div>
        <div class="drop-subtitle">支持 PDF / PNG / TXT / MP4 / ZIP</div>
        <el-button type="primary" @click="selectFiles">选择文件</el-button>
        <input ref="fileInput" type="file" class="hidden-input" multiple @change="onSelect" />
      </div>
      <div class="upload-queue">
        <div class="queue-title">上传队列</div>
        <div v-for="task in uploadTasks" :key="task.id" class="queue-item glass">
          <div>
            <div class="file-name">{{ task.name }}</div>
            <div class="file-meta">{{ task.sizeLabel }}</div>
          </div>
          <el-progress :percentage="task.progress" :status="task.status === 'error' ? 'exception' : ''" />
        </div>
        <div v-if="uploadTasks.length === 0" class="queue-empty">暂无上传任务</div>
        <div v-if="lastLink" class="result-card glass">
          <div class="result-title">上传完成</div>
          <div class="result-link">{{ lastLink }}</div>
          <el-button text @click="copyLink(lastLink)">复制链接</el-button>
        </div>
        <div class="result-card glass">
          <div class="result-title">下载任务</div>
          <div class="file-meta">在任务中心查看实时进度</div>
          <el-button text @click="openTaskCenter">查看任务中心</el-button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed, ref } from 'vue'
import { UploadFilled } from '@element-plus/icons-vue'
import { uploadFile } from '../api/files'
import { addTask, updateTask, completeTask, failTask, openTaskCenter, useTaskCenter } from '../store/taskCenter'
import { ElMessage } from 'element-plus'

const fileInput = ref(null)
const lastLink = ref('')
const state = useTaskCenter()

const uploadTasks = computed(() => state.tasks.filter((task) => task.type === 'upload'))

const selectFiles = () => fileInput.value?.click()

const onSelect = (event) => {
  const files = Array.from(event.target.files || [])
  handleFiles(files)
  event.target.value = ''
}

const onDrop = (event) => {
  const files = Array.from(event.dataTransfer.files || [])
  handleFiles(files)
}

const handleFiles = (files) => {
  files.forEach((file) => startUpload(file))
}

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

const startUpload = async (file) => {
  const id = `${Date.now()}-${file.name}`
  addTask({
    id,
    name: file.name,
    type: 'upload',
    progress: 0,
    status: 'running',
    sizeLabel: formatSize(file.size),
  })
  try {
    const response = await uploadFile(file, (event) => {
      const percent = event.total ? Math.round((event.loaded / event.total) * 100) : 0
      updateTask(id, { progress: percent })
    })
    const fileID = response.data?.data?.file_id
    lastLink.value = fileID ? `filehub://${fileID}` : ''
    completeTask(id)
    ElMessage.success('上传完成')
  } catch {
    failTask(id)
    ElMessage.error('上传失败')
  }
}

const copyLink = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制链接')
  } catch {
    ElMessage.error('复制失败')
  }
}
</script>
