import { createContext, useContext, useState, useCallback, useEffect } from 'react'

const LanguageContext = createContext(null)

const STORAGE_KEY = 'watchdog_lang'

// Arabic plural rules: 0=zero, 1=one, 2=two, 3-10=few, 11+=many, 100+=other
function arabicPluralForm(n) {
  if (n === 0) return 'zero'
  if (n === 1) return 'one'
  if (n === 2) return 'two'
  if (n >= 3 && n <= 10) return 'few'
  if (n >= 11 && n <= 99) return 'many'
  return 'other'
}

// English plural rules: 1=one, everything else=other
function englishPluralForm(n) {
  if (n === 1) return 'one'
  return 'other'
}

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

  // t(path, fallbackOrCount, count?) — supports pluralization
  // If the second arg is a number, it's treated as count for pluralization.
  // t('dashboard.cameras', 3) → looks for dashboard.cameras.few (Arabic) or dashboard.cameras.other (English)
  // t('login.title') → returns the string directly
  const t = useCallback((path, fallback, count) => {
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

    // If count is provided or fallback is a number, resolve plural form
    const n = count !== undefined ? count : (typeof fallback === 'number' ? fallback : undefined)
    if (n !== undefined && cur && typeof cur === 'object' && cur !== null) {
      const formFn = lang === 'ar' ? arabicPluralForm : englishPluralForm
      const form = formFn(n)
      // Try the specific form, then 'other', then any string value
      if (form in cur) cur = cur[form]
      else if ('other' in cur) cur = cur['other']
      else if (typeof cur === 'object') cur = path // can't resolve
    }

    if (cur === undefined || typeof cur === 'object') {
      return typeof fallback === 'string' ? fallback : path
    }
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
