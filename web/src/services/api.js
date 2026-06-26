import axios from 'axios'

const baseURL = (import.meta.env.VITE_API_URL || '') + '/api/v1'

const api = axios.create({
  baseURL,
  timeout: 30000,
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true,
})

// Auto-logout on 401 — but skip redirect for auth endpoints
// (AuthContext catches /auth/me 401s gracefully and sets user=null)
api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response && err.response.status === 401) {
      const url = err.config?.url || ''
      if (!url.includes('/auth/me') && !url.includes('/auth/login') && !url.includes('/auth/logout')) {
        localStorage.removeItem('watchdog_token')
        localStorage.removeItem('watchdog_user')
        localStorage.removeItem('watchdog_rules')
        localStorage.removeItem('watchdog_org')
        if (!window.location.pathname.endsWith('/login')) {
          window.location.href = '/login'
        }
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
}

export const apiPost = {
  person: (body) => api.post('/identity/persons', body).then((r) => r.data),
  match: (body) => api.post('/identity/match', body).then((r) => r.data),
  webrtcOffer: (body) => api.post('/webrtc/offer', body).then((r) => r.data),
}

export { api }
export default api
