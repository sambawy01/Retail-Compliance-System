import { createContext, useContext, useState, useCallback, useEffect } from 'react'

const LanguageContext = createContext(null)

const STORAGE_KEY = 'watchdog_lang'

export function LanguageProvider({ children }) {
  const [lang, setLang] = useState(() => {
    try {
      return localStorage.getItem(STORAGE_KEY) || 'en'
    } catch {
      return 'en'
    }
  })

  const dir = lang === 'ar' ? 'rtl' : 'ltr'

  useEffect(() => {
    document.documentElement.lang = lang
    document.documentElement.dir = dir
    try {
      localStorage.setItem(STORAGE_KEY, lang)
    } catch { /* ignore */ }
  }, [lang, dir])

  const toggle = useCallback(() => {
    setLang((p) => (p === 'en' ? 'ar' : 'en'))
  }, [])

  const t = useCallback((path, fallback) => {
    const dict = lang === 'ar' ? arDict : enDict
    const parts = path.split('.')
    let cur = dict
    for (const p of parts) {
      if (cur && typeof cur === 'object' && p in cur) {
        cur = cur[p]
      } else {
        // fallback to en
        let fb = enDict
        for (const fp of parts) {
          if (fb && typeof fb === 'object' && fp in fb) fb = fb[fp]
          else { cur = undefined; break }
        }
        cur = fb
        break
      }
    }
    if (cur === undefined) return fallback !== undefined ? fallback : path
    return cur
  }, [lang])

  const value = { lang, dir, isRTL: lang === 'ar', toggle, setLang, t }

  return (
    <LanguageContext.Provider value={value}>
      {children}
    </LanguageContext.Provider>
  )
}

// import dictionaries at bottom to keep render code clean
import enDict from '../i18n/en'
import arDict from '../i18n/ar'

export function useLang() {
  const ctx = useContext(LanguageContext)
  if (!ctx) throw new Error('useLang must be used within LanguageProvider')
  return ctx
}