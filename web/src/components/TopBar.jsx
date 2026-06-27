import { useEffect, useState } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { useLang } from '../contexts/LanguageContext'
import { healthCheck } from '../services/api'
import { Activity, Languages, User } from 'lucide-react'

export default function TopBar() {
  const { user } = useAuth()
  const { lang, toggle, t } = useLang()
  const [health, setHealth] = useState('unknown')

  useEffect(() => {
    let active = true
    const check = async () => {
      try {
        await healthCheck()
        if (active) setHealth('ok')
      } catch {
        if (active) setHealth('down')
      }
    }
    check()
    const id = setInterval(check, 30000)
    return () => { active = false; clearInterval(id) }
  }, [])

  const dotColor = health === 'ok' ? 'bg-success' : health === 'down' ? 'bg-critical' : 'bg-offline'

  return (
    <header className="h-16 sticky top-0 z-20 bg-bg-card border-b border-border flex items-center justify-between px-4 md:px-6 gap-4">
      <div className="flex items-center gap-2 ms-10 lg:ms-0">
        <span className={`w-2.5 h-2.5 rounded-full ${dotColor}`} />
        <span className="text-sm text-text-secondary hidden sm:inline">{t('topbar.health')}</span>
      </div>

      <div className="flex items-center gap-3">
        <button
          onClick={toggle}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-text-secondary hover:bg-bg-hover hover:text-text-primary transition-colors border border-border"
        >
          <Languages size={18} />
          <span>{lang === 'en' ? 'EN | ع' : 'ع | EN'}</span>
        </button>

        {user && (
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-bg-hover border border-border">
            <div className="w-7 h-7 rounded-full bg-accent text-white flex items-center justify-center text-xs font-bold">
              {(user.display_name || user.email || '?').charAt(0).toUpperCase()}
            </div>
            <span className="text-sm text-text-secondary hidden sm:inline max-w-[140px] truncate">
              {user.display_name || user.email}
            </span>
          </div>
        )}
      </div>
    </header>
  )
}