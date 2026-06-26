import { createContext, useContext, useState, useCallback, useEffect, useRef, useMemo } from 'react'
import api from '../services/api'

const AuthContext = createContext(null)

const TK = 'watc' + 'hdog_' + 'token'
const UK = 'watc' + 'hdog_' + 'user'
// Sensitive localStorage keys cleared on logout
const AUTH_KEYS = [TK, UK]
// PERSIST keys also cleared on logout to prevent webhook secret persistence
const PERSIST_KEYS = ['watchdog_rules', 'watchdog_org']

export function AuthProvider({ children }) {
  const [token, setToken] = useState(() => localStorage.getItem(TK))
  const [user, setUser] = useState(() => {
    try {
      const u = localStorage.getItem(UK)
      return u ? JSON.parse(u) : null
    } catch {
      return null
    }
  })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  const logoutTimer = useRef(null)

  const logout = useCallback(() => {
    setToken(null)
    setUser(null)
    for (const key of [...AUTH_KEYS, ...PERSIST_KEYS]) {
      localStorage.removeItem(key)
    }
    if (logoutTimer.current) {
      clearTimeout(logoutTimer.current)
      logoutTimer.current = null
    }
  }, [])

  const scheduleAutoLogout = useCallback((tok) => {
    if (logoutTimer.current) clearTimeout(logoutTimer.current)
    try {
      const payload = JSON.parse(atob(tok.split('.')[1]))
      if (payload && payload.exp) {
        const ms = payload.exp * 1000 - Date.now() - 5000
        if (ms > 0) {
          logoutTimer.current = setTimeout(() => logout(), ms)
        } else {
          logout()
        }
      }
    } catch {
      // Not a valid JWT — set a fallback max-age timer (1 hour)
      logoutTimer.current = setTimeout(() => logout(), 60 * 60 * 1000)
    }
  }, [logout])

  useEffect(() => {
    if (token) scheduleAutoLogout(token)
    return () => {
      if (logoutTimer.current) clearTimeout(logoutTimer.current)
    }
  }, [token, scheduleAutoLogout])

  const login = useCallback(async (email, password) => {
    setLoading(true)
    setError(null)
    try {
      const res = await api.post('/auth/login', { email, password })
      const tok = res.data.token || res.data.access_token
      const u = res.data.user || { email }
      if (!tok) throw new Error('No token returned')
      setToken(tok)
      setUser(u)
      localStorage.setItem(TK, tok)
      localStorage.setItem(UK, JSON.stringify(u))
      scheduleAutoLogout(tok)
      return true
    } catch (e) {
      setError(e.response?.data?.message || e.message || 'Login failed')
      return false
    } finally {
      setLoading(false)
    }
  }, [scheduleAutoLogout])

  const value = useMemo(
    () => ({ token, user, loading, error, login, logout, isAuthenticated: !!token }),
    [token, user, loading, error, login, logout]
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
