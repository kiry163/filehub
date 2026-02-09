<template>
  <section class="page">
    <div class="section-header">
      <div>
        <h1>上传文件</h1>
        <p>拖拽或选择文件上传，上传完成后复制 filehub:// 链接。</p>
      </div>
      <button class="btn ghost" @click="goBack">返回文件</button>
    </div>

    <div class="upload-grid">
      <div class="upload-drop" @dragover.prevent @drop.prevent="onDrop">
        <div class="drop-icon">
          <el-icon><UploadFilled /></el-icon>
        </div>
        <div class="drop-title">拖拽文件到这里</div>
        <div class="drop-subtitle">支持 PDF / PNG / TXT / MP4 / ZIP</div>
        <button class="btn primary" @click="selectFiles">选择文件</button>
        <input ref="fileInput" type="file" class="hidden-input" multiple @change="onSelect" />
      </div>
      <div class="upload-queue">
        <div class="queue-title">上传队列</div>
        <div v-for="task in uploadTasks" :key="task.id" class="queue-item">
          <div>
            <div class="file-title">{{ task.name }}</div>
            <div class="file-sub">{{ task.sizeLabel }}</div>
          </div>
          <el-progress :percentage="task.progress" :status="task.status === 'error' ? 'exception' : ''" />
        </div>
        <div v-if="uploadTasks.length === 0" class="queue-empty">暂无上传任务</div>
        <div v-if="lastLink" class="result-card">
          <div class="result-title">上传完成</div>
          <div class="result-link">{{ lastLink }}</div>
          <div class="file-actions">
            <button class="btn ghost" @click="copyText(lastLink)">复制 filehub://</button>
          </div>
        </div>
        <div class="result-card">
          <div class="result-title">任务中心</div>
          <div class="file-sub">下载与上传进度在任务中心查看</div>
          <button class="btn ghost" @click="openTaskCenter">查看任务中心</button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { UploadFilled } from '@element-plus/icons-vue'
import { uploadFile } from '../api/files'
import { addTask, updateTask, completeTask, failTask, openTaskCenter, useTaskCenter } from '../store/taskCenter'
import { ElMessage } from 'element-plus'

const router = useRouter()
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

const copyText = async (text) => {
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

const goBack = () => router.push('/')
</script>
