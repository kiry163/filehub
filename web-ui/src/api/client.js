import axios from 'axios'
import { getAccessToken, getRefreshToken, setTokens, clearTokens } from '../store/auth'

const client = axios.create({
  baseURL: '/',
  timeout: 30000,
})

client.interceptors.request.use((config) => {
  const token = getAccessToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

let refreshing = false
let pending = []

const resolvePending = (token) => {
  pending.forEach((callback) => callback(token))
  pending = []
}

client.interceptors.response.use(
  (response) => response,
  async (error) => {
    const original = error.config
    if (error.response?.status !== 401 || original?._retry) {
      return Promise.reject(error)
    }

    const refreshToken = getRefreshToken()
    if (!refreshToken) {
      clearTokens()
      return Promise.reject(error)
    }

    if (refreshing) {
      return new Promise((resolve) => {
        pending.push((token) => {
          original.headers.Authorization = `Bearer ${token}`
          resolve(client(original))
        })
      })
    }

    refreshing = true
    original._retry = true
    try {
      const response = await client.post('/api/v1/auth/refresh', {
        refresh_token: refreshToken,
      })
      const data = response.data?.data
      if (!data?.access_token) {
        throw new Error('refresh failed')
      }
      setTokens(data.access_token, data.refresh_token)
      resolvePending(data.access_token)
      original.headers.Authorization = `Bearer ${data.access_token}`
      return client(original)
    } catch (err) {
      clearTokens()
      resolvePending('')
      return Promise.reject(err)
    } finally {
      refreshing = false
    }
  },
)

export default client
