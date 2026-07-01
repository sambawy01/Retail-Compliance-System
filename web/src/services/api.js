import axios from 'axios'

const baseURL = (import.meta.env.VITE_API_URL || '') + '/api/v1'
// Shared localStorage key for the JWT — AuthContext imports this to avoid drift.
export const TOKEN_KEY = 'watchdog_token'

const api = axios.create({
  baseURL,
  timeout: 30000,
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true,
})

// Request interceptor: attach Bearer token from localStorage to every request
// Cookie may not work cross-site (third-party cookie blocking by browsers),
// so we use Bearer token auth as the primary mechanism
api.interceptors.request.use((config) => {
  const token = localStorage.getItem(TOKEN_KEY)
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Auto-logout on 401 — but skip redirect for auth endpoints.
// On 401 from a non-auth endpoint, attempt a single token refresh before
// redirecting to login. If the refresh also fails, clear the token.
let refreshAttempted = false
api.interceptors.response.use(
  (res) => res,
  async (err) => {
    if (err.response && err.response.status === 401) {
      const url = err.config?.url || ''
      const isAuthEndpoint = url.includes('/auth/me') || url.includes('/auth/login') || url.includes('/auth/logout') || url.includes('/auth/refresh')
      if (!isAuthEndpoint && !refreshAttempted) {
        refreshAttempted = true
        try {
          const token = localStorage.getItem(TOKEN_KEY)
          if (token) {
            const res = await api.post('/auth/refresh', {}, {
              headers: { Authorization: `Bearer ${token}` },
            })
            if (res.data.token) {
              localStorage.setItem(TOKEN_KEY, res.data.token)
              refreshAttempted = false
              // Retry the original request with the new token
              err.config.headers.Authorization = `Bearer ${res.data.token}`
              return api.request(err.config)
            }
          }
        } catch {
          // refresh failed — fall through to logout
        }
        refreshAttempted = false
      }
      // Clear token and redirect to login
      localStorage.removeItem(TOKEN_KEY)
      if (!window.location.pathname.endsWith('/login')) {
        window.location.href = '/login'
      }
    }
    return Promise.reject(err)
  }
)

export const healthCheck = () =>
  axios.get((import.meta.env.VITE_API_URL || '') + '/health')

export function connectWebSocket(onMessage, onOpen, onClose) {
  const wsBase = (import.meta.env.VITE_API_URL || '').replace(/^http/, 'ws')
  if (!wsBase) return null
  try {
    const ws = new WebSocket(`${wsBase}/ws/events`)
    ws.onopen = () => { onOpen && onOpen() }
    ws.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data)
        onMessage && onMessage(data)
      } catch { /* ignore non-JSON */ }
    }
    ws.onclose = () => { onClose && onClose() }
    ws.onerror = () => { try { ws.close() } catch {} }
    return ws
  } catch {
    return null
  }
}

export const apiGet = {
  cameras: () => api.get('/vision/cameras').then((r) => r.data),
  camera: (id) => api.get(`/vision/cameras/${id}`).then((r) => r.data),
  zones: (cameraId) => api.get('/vision/zones', { params: { camera_id: cameraId } }).then((r) => r.data),
  detections: (params) => api.get('/vision/detections', { params }).then((r) => r.data),
  clips: (cameraId) => api.get('/vision/clips', { params: { camera_id: cameraId } }).then((r) => r.data),
  persons: (kind) => api.get('/identity/persons', { params: { kind } }).then((r) => r.data),
  notificationRules: () => api.get('/notifications/rules').then((r) => r.data),
}

export const apiPost = {
  person: (body) => api.post('/identity/persons', body).then((r) => r.data),
  match: (body) => api.post('/identity/match', body).then((r) => r.data),
  webrtcOffer: (body) => api.post('/webrtc/offer', body).then((r) => r.data),
  notificationRule: (body) => api.post('/notifications/rules', body).then((r) => r.data),
}

export const apiPatch = {
  notificationRule: (id, body) => api.patch(`/notifications/rules/${id}`, body).then((r) => r.data),
}

export const apiDelete = {
  notificationRule: (id) => api.delete(`/notifications/rules/${id}`).then((r) => r.data),
}

export { api }
export default api
