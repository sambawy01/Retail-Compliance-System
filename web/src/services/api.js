import axios from 'axios'

const baseURL = (import.meta.env.VITE_API_URL || '') + '/api/v1'

const api = axios.create({
  baseURL,
  timeout: 30000,
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true, // send httpOnly cookies with every request
})

// Auto-logout on 401
api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response && err.response.status === 401) {
      // Clear any lingering localStorage data from the old token-based auth
      localStorage.removeItem('watchdog_token')
      localStorage.removeItem('watchdog_user')
      localStorage.removeItem('watchdog_rules')
      localStorage.removeItem('watchdog_org')
      // Redirect to login if not already there
      if (!window.location.pathname.endsWith('/login')) {
        window.location.href = '/login'
      }
    }
    return Promise.reject(err)
  }
)

// Helper for raw health check (not under /api/v1)
export const healthCheck = () =>
  axios.get((import.meta.env.VITE_API_URL || '') + '/health')

// WebSocket connection helper
// With httpOnly cookies, WS can't read the token from document.cookie (httpOnly).
// For WS auth, the backend should issue a short-lived WS ticket via /api/v1/auth/ws-ticket.
// For now, WS falls back to polling if no ticket is available.
export function connectWebSocket(onMessage, onOpen, onClose) {
  const wsBase = (import.meta.env.VITE_API_URL || '').replace(/^http/, 'ws')
  if (!wsBase) return null
  // Try WS without token — cookies aren't sent over WS, so this may fail.
  // The proper fix is a WS ticket endpoint, but for now we attempt and let
  // onClose trigger the polling fallback.
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

// Domain-specific helpers
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
