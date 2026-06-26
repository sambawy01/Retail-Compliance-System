import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import { useLang } from '../contexts/LanguageContext'
import { ShieldCheck, Languages, Loader2, Mail, Lock } from 'lucide-react'

export default function Login() {
  const { login, error, isAuthenticated } = useAuth()
  const [submitting, setSubmitting] = useState(false)
  const { t, toggle, lang } = useLang()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')

  useEffect(() => { if (isAuthenticated) navigate('/', { replace: true }) }, [isAuthenticated, navigate])

  const handleSubmit = async (e) => {
    e.preventDefault()
    setSubmitting(true)
    const ok = await login(email, password)
    setSubmitting(false)
    if (ok) navigate('/', { replace: true })
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-bg p-4">
      <div className="absolute top-4 end-4">
        <button onClick={toggle} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-text-secondary hover:bg-bg-hover border border-border">
          <Languages size={18} /> {lang === 'en' ? 'ع' : 'EN'}
        </button>
      </div>

      <div className="w-full max-w-sm">
        <div className="flex flex-col items-center mb-8">
          <div className="w-16 h-16 rounded-2xl bg-accent/15 flex items-center justify-center mb-3">
            <ShieldCheck size={36} className="text-accent" />
          </div>
          <h1 className="text-2xl font-bold text-text-primary">{t('login.title')}</h1>
          <p className="text-sm text-text-secondary mt-1">{t('login.subtitle')}</p>
        </div>

        <form onSubmit={handleSubmit} className="bg-bg-card border border-border rounded-xl p-6 space-y-4">
          <div>
            <label className="block text-xs font-medium text-text-secondary mb-1.5">{t('login.email')}</label>
            <div className="relative">
              <Mail size={16} className="absolute top-1/2 -translate-y-1/2 text-text-muted" style={{ insetInlineStart: 12 }} />
              <input
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full bg-bg border border-border rounded-lg ps-9 pe-3 py-2.5 text-sm text-text-primary focus:outline-none focus:border-accent"
                placeholder="you@company.com"
                autoComplete="email"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-medium text-text-secondary mb-1.5">{t('login.password')}</label>
            <div className="relative">
              <Lock size={16} className="absolute top-1/2 -translate-y-1/2 text-text-muted" style={{ insetInlineStart: 12 }} />
              <input
                type="password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full bg-bg border border-border rounded-lg ps-9 pe-3 py-2.5 text-sm text-text-primary focus:outline-none focus:border-accent"
                placeholder="••••••••"
                autoComplete="current-password"
              />
            </div>
          </div>

          {error && (
            <div className="text-sm text-critical bg-critical/10 border border-critical/30 rounded-lg px-3 py-2">
              {t('login.error')}
            </div>
          )}

          <button
            type="submit"
            disabled={submitting}
            className="w-full bg-accent hover:bg-accent-hover text-white font-semibold py-2.5 rounded-lg transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
          >
            {loading && <Loader2 size={16} className="animate-spin" />}
            {loading ? t('login.loading') : t('login.submit')}
          </button>
        </form>
      </div>
    </div>
  )
}