import client from './client'

export const login = (username, password) =>
  client.post('/api/v1/auth/login', { username, password })

export const logout = () => client.post('/api/v1/auth/logout')
