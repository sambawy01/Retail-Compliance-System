import { createContext, useContext, useState, useCallback, useEffect, useMemo } from 'react'
import api from '../services/api'

const AuthContext = createContext(null)
const TOKEN_KEY = 'watchdog_token'

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  // On mount: check if we have a stored token and validate via /me
  useEffect(() => {
    let active = true
    const token = localStorage.getItem(TOKEN_KEY)
    if (!token) {
      setLoading(false)
      return
    }
    // Attach token as Bearer header for this request
    api.get('/auth/me', { headers: { Authorization: `Bearer ${token}` } })
      .then((res) => {
        if (active && res.data) {
          const u = res.data.user || res.data
          setUser(u)
        }
      })
      .catch(() => {
        if (active) {
          setUser(null)
          localStorage.removeItem(TOKEN_KEY)
        }
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => { active = false }
  }, [])

  const logout = useCallback(async () => {
    try { await api.post('/auth/logout') } catch {}
    setUser(null)
    localStorage.removeItem(TOKEN_KEY)
  }, [])

  const login = useCallback(async (email, password) => {
    setLoading(true)
    setError(null)
    try {
      const res = await api.post('/auth/login', { email, password })
      const u = res.data.user
      if (!u) throw new Error('Login failed')
      // Store the JWT token from the response body
      // Cookie may not work cross-site (third-party cookie blocking),
      // so we use Bearer token auth as the primary mechanism
      if (res.data.token) {
        localStorage.setItem(TOKEN_KEY, res.data.token)
      }
      setUser(u)
      return true
    } catch (e) {
      setError(e.response?.data?.error || e.message || 'Login failed')
      return false
    } finally {
      setLoading(false)
    }
  }, [])

  const value = useMemo(
    () => ({ user, loading, error, login, logout, isAuthenticated: !!user }),
    [user, loading, error, login, logout]
  )

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
