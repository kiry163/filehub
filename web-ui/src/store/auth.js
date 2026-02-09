const STORAGE_KEY = 'filehub.auth'

let state = {
  accessToken: '',
  refreshToken: '',
}

const load = () => {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return
    const parsed = JSON.parse(raw)
    state = {
      accessToken: parsed.accessToken || '',
      refreshToken: parsed.refreshToken || '',
    }
  } catch {
    state = { accessToken: '', refreshToken: '' }
  }
}

const persist = () => {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
}

load()

export const setTokens = (accessToken, refreshToken) => {
  state.accessToken = accessToken
  state.refreshToken = refreshToken
  persist()
}

export const clearTokens = () => {
  state.accessToken = ''
  state.refreshToken = ''
  persist()
}

export const getAccessToken = () => state.accessToken
export const getRefreshToken = () => state.refreshToken
