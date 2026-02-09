<template>
  <section class="page">
    <div class="section-header">
      <div>
        <h1>文件</h1>
        <p>管理文件、搜索与批量操作，复制 filehub:// 与分享链接。</p>
      </div>
      <div class="storage-card">
        <div class="storage-title">存储使用</div>
        <div class="storage-value">{{ formatSize(usedBytes) }} / {{ formatSize(capacityBytes) }}</div>
        <div class="progress"><span :style="{ width: `${usagePercent}%` }"></span></div>
      </div>
    </div>

    <div class="toolbar">
      <label class="search" aria-label="搜索文件">
        <span class="icon" aria-hidden="true">
          <el-icon><Search /></el-icon>
        </span>
        <input v-model="keyword" type="text" placeholder="搜索文件名 / ID / 类型" @input="onSearch" />
      </label>
      <div class="view-toggle" role="group" aria-label="视图切换">
        <button class="btn subtle" :class="{ active: layout === 'list' }" @click="layout = 'list'">列表</button>
        <button class="btn subtle" :class="{ active: layout === 'grid' }" @click="layout = 'grid'">网格</button>
      </div>
      <button class="btn danger" :disabled="selectedIds.length === 0" @click="batchRemove">批量删除</button>
    </div>

    <div v-if="loading" class="loading">加载中...</div>
    <div v-else-if="files.length === 0" class="empty">暂无文件</div>

    <div v-else class="file-board" :data-layout="layout">
      <div class="file-list">
        <div class="list-head">
          <label class="checkbox">
            <input type="checkbox" :checked="allSelected" @change="toggleAll" />
            <span></span>
          </label>
          <div>文件名</div>
          <div>大小</div>
          <div>创建时间</div>
          <div>操作</div>
        </div>

        <div v-for="file in files" :key="file.file_id" class="file-row">
          <label class="checkbox">
            <input type="checkbox" :value="file.file_id" v-model="selectedIds" />
            <span></span>
          </label>
          <div class="file-main">
            <div class="file-title">{{ file.original_name }}</div>
            <div class="file-sub">{{ fileLabel(file) }}</div>
          </div>
          <div>{{ formatSize(file.size) }}</div>
          <div>{{ formatDate(file.created_at) }}</div>
          <div class="file-actions">
            <button class="btn ghost" @click="copyFilehub(file)">复制 filehub://</button>
            <button class="btn ghost" @click="copyShare(file)">复制分享链接</button>
            <button class="btn ghost" @click="goDetail(file)">预览</button>
            <button class="btn ghost" @click="download(file)">下载</button>
            <button class="btn ghost danger" @click="removeFile(file)">删除</button>
          </div>
        </div>
      </div>

      <div class="file-grid">
        <article v-for="file in files" :key="file.file_id" class="file-tile">
          <div class="file-tile-header">
            <div>
              <div class="file-title">{{ file.original_name }}</div>
              <div class="file-sub">{{ formatSize(file.size) }} · {{ formatDate(file.created_at) }}</div>
            </div>
            <label class="checkbox">
              <input type="checkbox" :value="file.file_id" v-model="selectedIds" />
              <span></span>
            </label>
          </div>
          <div class="file-tile-body">
            <div class="file-tag">{{ fileType(file) }}</div>
            <div class="file-id">filehub://{{ file.file_id }}</div>
          </div>
          <div class="file-actions">
            <button class="btn ghost" @click="copyFilehub(file)">复制 filehub://</button>
            <button class="btn ghost" @click="copyShare(file)">复制分享链接</button>
            <button class="btn ghost" @click="goDetail(file)">预览</button>
          </div>
          <div class="file-actions">
            <button class="btn ghost" @click="download(file)">下载</button>
            <button class="btn ghost danger" @click="removeFile(file)">删除</button>
          </div>
          <div v-if="!isPreviewable(file)" class="file-note">此文件类型不支持预览</div>
        </article>
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
import { useRouter } from 'vue-router'
import { Search } from '@element-plus/icons-vue'
import { listFiles, deleteFile, downloadFile, shareFile } from '../api/files'
import { tryCopyText } from '../utils/copy'
import { ElMessage, ElMessageBox } from 'element-plus'

const router = useRouter()
const loading = ref(false)
const files = ref([])
const keyword = ref('')
const layout = ref('list')
const selectedIds = ref([])
const copyDialogOpen = ref(false)
const copyDialogText = ref('')
let searchTimer

const capacityBytes = 50 * 1024 * 1024 * 1024
const usedBytes = computed(() => files.value.reduce((sum, file) => sum + (file.size || 0), 0))
const usagePercent = computed(() => Math.min(100, Math.round((usedBytes.value / capacityBytes) * 100)))

const allSelected = computed(() => files.value.length > 0 && selectedIds.value.length === files.value.length)

const fetchFiles = async () => {
  loading.value = true
  try {
    const response = await listFiles({
      limit: 200,
      offset: 0,
      order: 'desc',
      keyword: keyword.value || undefined,
    })
    files.value = response.data?.data?.files || []
    selectedIds.value = []
  } catch {
    ElMessage.error('获取文件失败')
  } finally {
    loading.value = false
  }
}

const onSearch = () => {
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => fetchFiles(), 300)
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

const formatDate = (value) => value?.replace('T', ' ').slice(0, 16) || '--'

const fileType = (file) => {
  const ext = file.original_name?.split('.').pop()?.toUpperCase()
  return ext || file.mime_type || 'FILE'
}

const fileLabel = (file) => `${fileType(file)} · filehub://${file.file_id}`

const isPreviewable = (file) => {
  const name = file.original_name?.toLowerCase() || ''
  const mime = file.mime_type || ''
  return (
    mime.startsWith('image/') ||
    mime.startsWith('video/') ||
    mime.includes('markdown') ||
    name.endsWith('.md') ||
    name.endsWith('.markdown') ||
    name.endsWith('.txt')
  )
}

const showCopyDialog = (text) => {
  copyDialogText.value = text
  copyDialogOpen.value = true
}

const copyFilehub = async (file) => {
  const text = `filehub://${file.file_id}`
  const ok = await tryCopyText(text)
  if (ok) {
    ElMessage.success('已复制 filehub://')
  } else {
    showCopyDialog(text)
  }
}

const copyShare = async (file) => {
  try {
    const response = await shareFile(file.file_id)
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

const goDetail = (file) => router.push(`/file/${file.file_id}`)

const download = async (file) => {
  try {
    const response = await downloadFile(file.file_id)
    const blob = new Blob([response.data])
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = file.original_name
    link.click()
    window.URL.revokeObjectURL(url)
  } catch {
    ElMessage.error('下载失败')
  }
}

const removeFile = async (file) => {
  try {
    await ElMessageBox.confirm(`确认删除 ${file.original_name} ?`, '删除文件', { type: 'warning' })
    await deleteFile(file.file_id)
    ElMessage.success('已删除')
    fetchFiles()
  } catch {}
}

const batchRemove = async () => {
  try {
    await ElMessageBox.confirm(`确认删除 ${selectedIds.value.length} 个文件？`, '批量删除', { type: 'warning' })
    await Promise.all(selectedIds.value.map((id) => deleteFile(id)))
    ElMessage.success('已删除')
    fetchFiles()
  } catch {}
}

const toggleAll = (event) => {
  if (event.target.checked) {
    selectedIds.value = files.value.map((file) => file.file_id)
  } else {
    selectedIds.value = []
  }
}

onMounted(fetchFiles)
</script>
