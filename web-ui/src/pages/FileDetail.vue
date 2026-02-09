<template>
  <section class="page">
    <div class="section-header">
      <div>
        <h1>文件详情</h1>
        <p>自动识别文件类型并展示可预览内容。</p>
      </div>
      <button class="btn ghost" @click="goBack">返回文件</button>
    </div>

    <div v-if="loading" class="loading">加载中...</div>

    <div v-else class="detail-grid">
      <div class="detail-preview">
        <div class="preview-header">
          <div>
            <div class="file-title">{{ file?.original_name }}</div>
            <div class="file-sub">{{ metaLine }}</div>
          </div>
          <div class="preview-meta">
            <span class="pill active">{{ previewLabel }}</span>
          </div>
        </div>
        <div class="preview-body" :class="{ unsupported: previewType === 'none' }">
          <div v-if="previewType === 'image'" class="preview-pane active">
            <div class="media-frame" v-if="previewUrl">
              <img :src="previewUrl" alt="文件图片预览" />
            </div>
            <div v-else class="preview-empty">图片加载中...</div>
          </div>
          <div v-else-if="previewType === 'video'" class="preview-pane active">
            <div class="media-frame" v-if="previewUrl">
              <video controls preload="metadata" :src="previewUrl"></video>
            </div>
            <div v-else class="preview-empty">视频加载中...</div>
          </div>
          <div v-else-if="previewType === 'markdown'" class="preview-pane active">
            <article class="markdown" v-html="markdownHtml"></article>
          </div>
          <div v-else class="preview-placeholder">
            <div>
              <div class="preview-title">不支持预览的类型</div>
              <div class="file-sub">请直接下载查看</div>
            </div>
          </div>
        </div>
      </div>
      <div class="detail-actions">
        <div class="action-title">操作</div>
        <button class="btn primary full" @click="download">下载文件</button>
        <button class="btn ghost full" @click="copyFilehub">复制 filehub://</button>
        <button class="btn ghost full" @click="copyShare">复制分享链接</button>
        <button class="btn danger full" @click="remove">删除文件</button>
        <div class="mini-info">
          <div>类型：{{ file?.mime_type || '-' }}</div>
          <div>Key：{{ file?.file_id || '-' }}</div>
        </div>
      </div>
    </div>
  </section>

  <el-dialog v-model="copyDialogOpen" title="手动复制" width="420px" append-to-body>
    <div class="dialog-tip">自动复制失败，请手动复制：</div>
    <el-input :model-value="copyDialogText" readonly />
  </el-dialog>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import MarkdownIt from 'markdown-it'
import { getFile, downloadFile, deleteFile, getPreviewUrl, shareFile } from '../api/files'
import { tryCopyText } from '../utils/copy'
import { addTask, updateTask, completeTask, failTask } from '../store/taskCenter'
import { ElMessage, ElMessageBox } from 'element-plus'

const route = useRoute()
const router = useRouter()
const file = ref(null)
const loading = ref(false)
const markdownHtml = ref('')
const markdown = new MarkdownIt({ html: false, linkify: true })
const previewUrl = ref('')
const copyDialogOpen = ref(false)
const copyDialogText = ref('')

const metaLine = computed(() => {
  if (!file.value) return ''
  return `${file.value.size ? formatSize(file.value.size) : '--'} · ${formatDate(file.value.created_at)}`
})

const previewType = computed(() => {
  if (!file.value) return 'none'
  const name = file.value.original_name?.toLowerCase() || ''
  const mime = file.value.mime_type || ''
  if (mime.startsWith('image/')) return 'image'
  if (mime.startsWith('video/')) return 'video'
  if (mime.includes('markdown') || name.endsWith('.md') || name.endsWith('.markdown') || name.endsWith('.txt')) {
    return 'markdown'
  }
  return 'none'
})

const previewLabel = computed(() => {
  if (previewType.value === 'image') return '图片'
  if (previewType.value === 'video') return '视频'
  if (previewType.value === 'markdown') return 'Markdown'
  return '不支持预览'
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

const formatDate = (value) => value?.replace('T', ' ').slice(0, 16) || '--'

const fetchFile = async () => {
  loading.value = true
  try {
    const response = await getFile(route.params.id)
    file.value = response.data?.data
    if (!file.value) throw new Error('not found')
    await loadPreview()
  } catch {
    ElMessage.error('文件不存在或加载失败')
  } finally {
    loading.value = false
  }
}

const loadPreview = async () => {
  if (!file.value?.file_id) return
  if (previewType.value === 'markdown') {
    const response = await downloadFile(file.value.file_id)
    const text = await response.data.text()
    markdownHtml.value = markdown.render(text)
    return
  }
  if (previewType.value === 'image' || previewType.value === 'video') {
    const response = await getPreviewUrl(file.value.file_id)
    previewUrl.value = response.data?.data?.url || ''
  }
}

const showCopyDialog = (text) => {
  copyDialogText.value = text
  copyDialogOpen.value = true
}

const copyFilehub = async () => {
  const text = `filehub://${file.value?.file_id}`
  const ok = await tryCopyText(text)
  if (ok) {
    ElMessage.success('已复制 filehub://')
  } else {
    showCopyDialog(text)
  }
}

const copyShare = async () => {
  try {
    const response = await shareFile(file.value?.file_id)
    const url = response.data?.data?.url
    if (!url) throw new Error('share failed')
    const ok = await tryCopyText(url)
    if (ok) {
      ElMessage.success('已复制分享链接')
    } else {
      showCopyDialog(url)
    }
  } catch {
    ElMessage.error('分享链接生成失败')
  }
}

const download = async () => {
  if (!file.value?.file_id) return
  const id = `${Date.now()}-${file.value.file_id}`
  addTask({
    id,
    name: file.value.original_name,
    type: 'download',
    progress: 0,
    status: 'running',
    sizeLabel: formatSize(file.value.size),
  })
  try {
    const response = await downloadFile(file.value.file_id, (event) => {
      if (event.total) {
        const percent = Math.round((event.loaded / event.total) * 100)
        updateTask(id, { progress: percent })
      }
    })
    const blob = new Blob([response.data])
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = file.value.original_name
    link.click()
    window.URL.revokeObjectURL(url)
    completeTask(id)
  } catch {
    failTask(id)
    ElMessage.error('下载失败')
  }
}

const remove = async () => {
  try {
    await ElMessageBox.confirm('确认删除该文件？', '删除文件', { type: 'warning' })
    await deleteFile(file.value.file_id)
    ElMessage.success('已删除')
    router.push('/')
  } catch {}
}

const goBack = () => router.push('/')

onMounted(fetchFile)
</script>
