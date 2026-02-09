import client from './client'

export const listFiles = (params) => client.get('/api/v1/files', { params })

export const getFile = (id) => client.get(`/api/v1/files/${id}`)

export const uploadFile = (file, onProgress) => {
  const form = new FormData()
  form.append('file', file)
  return client.post('/api/v1/files', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    onUploadProgress: onProgress,
    timeout: 0,
  })
}

export const deleteFile = (id) => client.delete(`/api/v1/files/${id}`)

export const shareFile = (id) => client.get(`/api/v1/files/${id}/share`)

export const getPreviewUrl = (id) => client.get(`/api/v1/files/${id}/preview`)

export const downloadFile = (id, onProgress) =>
  client.get(`/api/v1/files/${id}/download`, {
    responseType: 'blob',
    onDownloadProgress: onProgress,
    timeout: 0,
  })
