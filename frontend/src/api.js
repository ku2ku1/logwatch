import axios from 'axios'

// Production mein same domain use karo, development mein localhost
const BASE = import.meta.env.PROD
  ? ''
  : 'http://127.0.0.1:8080'

const api = axios.create({ baseURL: BASE, withCredentials: true })

api.interceptors.request.use(cfg => {
  const token = localStorage.getItem('token')
  if (token) cfg.headers.Authorization = `Bearer ${token}`
  return cfg
})

api.interceptors.response.use(
  r => r,
  err => {
    if (err.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)

export const login = (u, p) => api.post('/api/auth/login', { username:u, password:p }).then(r => r.data)
export const logout = () => api.post('/api/auth/logout').then(r => r.data)
export const getMe = () => api.get('/api/auth/me').then(r => r.data)
export const getStats = () => api.get('/api/v1/stats').then(r => r.data)
export const getTopPaths = () => api.get('/api/v1/top/paths').then(r => r.data)
export const getTopIPs = () => api.get('/api/v1/top/ips').then(r => r.data)
export const getStatusCodes = () => api.get('/api/v1/status-codes').then(r => r.data)
export const getSecurityStats = () => api.get('/api/v1/security/stats').then(r => r.data)
export const getRecentThreats = () => api.get('/api/v1/security/threats').then(r => r.data)
export const getTopAttackers = () => api.get('/api/v1/security/attackers').then(r => r.data)
export const listUsers = () => api.get('/api/v1/users').then(r => r.data)
export const createUser = (u, p, r) => api.post('/api/v1/users', { username:u, password:p, role:r }).then(r => r.data)
export const deleteUser = (id) => api.delete(`/api/v1/users/${id}`).then(r => r.data)
