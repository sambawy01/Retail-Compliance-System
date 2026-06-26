import { createContext, useContext, useState, useCallback, useEffect, useRef, useMemo } from 'react'
import api from '../services/api'

const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  // On mount: check if we have a valid session by calling /me
  // The httpOnly cookie is sent automatically by axios (withCredentials: true)
  useEffect(() => {
    let active = true
    api.get('/auth/me')
      .then((res) => {
        if (active && res.data) {
          setUser(res.data)
        }
      })
      .catch(() => {
        // No valid session — user is not authenticated
        if (active) setUser(null)
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => { active = false }
  }, [])

  const logout = useCallback(async () => {
    try {
      await api.post('/auth/logout')
    } catch { /* ignore — cookie may already be expired */ }
    setUser(null)
    // Clear any legacy localStorage data
    localStorage.removeItem('watchdog_token')
    localStorage.removeItem('watchdog_user')
    localStorage.removeItem('watchdog_rules')
    localStorage.removeItem('watchdog_org')
  }, [])

  const login = useCallback(async (email, password) => {
    setLoading(true)
    setError(null)
    try {
      const res = await api.post('/auth/login', { email, password })
      // Token is set as httpOnly cookie by the server — we only get user info back
      const u = res.data.user
      if (!u) throw new Error('Login failed')
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
