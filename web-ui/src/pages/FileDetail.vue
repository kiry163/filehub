<template>
  <section class="page">
    <div class="section-header">
      <div>
        <h1>文件详情</h1>
        <p>查看内容预览、下载或分享 filehub:// 链接。</p>
      </div>
    </div>

    <div v-if="loading" class="loading">加载中...</div>

    <div v-else class="detail-grid">
      <div class="detail-preview glass">
        <div class="preview-header">
          <div>
            <div class="file-name">{{ file?.original_name }}</div>
            <div class="file-meta">{{ metaLine }}</div>
          </div>
          <el-button text @click="copyLink">复制链接</el-button>
        </div>
        <div class="preview-tabs">
          <el-button text :class="activeTab === 'image' ? 'active' : ''" @click="activeTab = 'image'">图片预览</el-button>
          <el-button text :class="activeTab === 'markdown' ? 'active' : ''" @click="activeTab = 'markdown'">Markdown</el-button>
          <el-button text :class="activeTab === 'video' ? 'active' : ''" @click="activeTab = 'video'">视频</el-button>
          <el-button text @click="openFullscreen">全屏</el-button>
        </div>
        <div class="preview-body">
          <div v-if="activeTab === 'image'" class="preview-pane active">
            <div class="media-frame" v-if="previewUrl">
              <img :src="previewUrl" alt="文件图片预览" />
            </div>
            <div v-else class="preview-empty">图片加载中...</div>
          </div>
          <div v-if="activeTab === 'markdown'" class="preview-pane active">
            <article class="markdown" v-html="markdownHtml"></article>
          </div>
          <div v-if="activeTab === 'video'" class="preview-pane active">
            <div class="media-frame" v-if="previewUrl">
              <video controls preload="metadata" :src="previewUrl"></video>
            </div>
            <div v-else class="preview-empty">视频加载中...</div>
          </div>
        </div>
      </div>
      <div class="detail-actions glass">
        <div class="action-title">操作</div>
        <el-button type="primary" @click="download">下载文件</el-button>
        <div class="download-progress" v-if="downloadProgress > 0 && downloadProgress < 100">
          <div class="file-meta">下载中 · {{ downloadProgress }}%</div>
          <el-progress :percentage="downloadProgress" />
        </div>
        <el-button text @click="copyLink">复制 filehub:// 链接</el-button>
        <el-button text type="danger" @click="remove">删除文件</el-button>
        <div class="mini-info">
          <div>类型：{{ file?.mime_type || '-' }}</div>
          <div>Key：{{ file?.file_id || '-' }}</div>
        </div>
      </div>
    </div>
  </section>

  <el-dialog v-model="fullscreen" width="90%" top="5vh" append-to-body>
    <template #header>
      <div class="fullscreen-header">预览</div>
    </template>
    <div class="fullscreen-body">
      <div v-if="activeTab === 'image'" class="media-frame">
        <img :src="previewUrl" alt="全屏预览" />
      </div>
      <div v-if="activeTab === 'markdown'" class="markdown" v-html="markdownHtml"></div>
      <div v-if="activeTab === 'video'" class="media-frame">
        <video controls preload="metadata" :src="previewUrl"></video>
      </div>
    </div>
  </el-dialog>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import MarkdownIt from 'markdown-it'
import { getFile, downloadFile, deleteFile, getPreviewUrl } from '../api/files'
import { addTask, updateTask, completeTask, failTask } from '../store/taskCenter'
import { ElMessage, ElMessageBox } from 'element-plus'

const route = useRoute()
const router = useRouter()
const file = ref(null)
const loading = ref(false)
const activeTab = ref('image')
const markdownHtml = ref('')
const markdown = new MarkdownIt({ html: false, linkify: true })
const previewUrl = ref('')
const fullscreen = ref(false)
const downloadProgress = ref(0)

const metaLine = computed(() => {
  if (!file.value) return ''
  return `${file.value.size ? formatSize(file.value.size) : '--'} · ${file.value.created_at?.slice(0, 16) || '--'}`
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

const fetchFile = async () => {
  loading.value = true
  try {
    const response = await getFile(route.params.id)
    file.value = response.data?.data
    if (!file.value) throw new Error('not found')
    const mime = file.value.mime_type || ''
    if (mime.startsWith('image/')) {
      activeTab.value = 'image'
      await loadPreviewUrl()
    } else if (mime.startsWith('video/')) {
      activeTab.value = 'video'
      await loadPreviewUrl()
    } else if (file.value.original_name?.endsWith('.md') || mime.includes('markdown')) {
      activeTab.value = 'markdown'
      await loadMarkdown()
    }
  } catch {
    ElMessage.error('文件不存在或加载失败')
  } finally {
    loading.value = false
  }
}

const loadMarkdown = async () => {
  if (!file.value?.file_id) return
  const response = await downloadFile(file.value.file_id)
  const text = await response.data.text()
  markdownHtml.value = markdown.render(text)
}

const loadPreviewUrl = async () => {
  if (!file.value?.file_id) return
  if (previewUrl.value) return
  const response = await getPreviewUrl(file.value.file_id)
  previewUrl.value = response.data?.data?.url || ''
}

const copyLink = async () => {
  const text = `filehub://${file.value?.file_id}`
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制链接')
  } catch {
    ElMessage.error('复制失败')
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
    downloadProgress.value = 0
    const response = await downloadFile(file.value.file_id, (event) => {
      if (event.total) {
        const percent = Math.round((event.loaded / event.total) * 100)
        downloadProgress.value = percent
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

const openFullscreen = () => {
  fullscreen.value = true
}

watch(activeTab, async (tab) => {
  if (tab === 'markdown' && !markdownHtml.value) {
    await loadMarkdown()
  }
  if ((tab === 'image' || tab === 'video') && !previewUrl.value) {
    await loadPreviewUrl()
  }
})

onMounted(fetchFile)

</script>
