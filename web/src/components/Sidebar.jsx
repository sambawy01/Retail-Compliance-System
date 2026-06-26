import { NavLink, useNavigate } from 'react-router-dom'
import { LayoutDashboard, Camera, ListTree, ScanFace, Settings, ShieldCheck, LogOut } from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import { useLang } from '../contexts/LanguageContext'
import { useState } from 'react'

export default function Sidebar() {
  const { logout } = useAuth()
  const { t } = useLang()
  const navigate = useNavigate()
  const [mobileOpen, setMobileOpen] = useState(false)

  const links = [
    { to: '/', icon: LayoutDashboard, label: t('nav.dashboard'), end: true },
    { to: '/cameras', icon: Camera, label: t('nav.cameras') },
    { to: '/events', icon: ListTree, label: t('nav.events') },
    { to: '/faceid', icon: ScanFace, label: t('nav.faceid') },
    { to: '/settings', icon: Settings, label: t('nav.settings') },
  ]

  const handleLogout = () => { logout(); navigate('/login') }
  const closeMobile = () => setMobileOpen(false)

  return (
    <>
      {/* Mobile toggle */}
      <button
        onClick={() => setMobileOpen(!mobileOpen)}
        className="lg:hidden fixed top-3 z-50 p-2 rounded-md bg-bg-card border border-border text-text-primary"
        style={{ insetInlineStart: 12 }}
        aria-label="menu"
      >
        <LayoutDashboard size={20} />
      </button>

      {/* Backdrop */}
      {mobileOpen && <div onClick={closeMobile} className="lg:hidden fixed inset-0 bg-black/50 z-30" />}

      <aside
        className={`${mobileOpen ? 'translate-x-0' : '-translate-x-full rtl:translate-x-full'} lg:translate-x-0 rtl:lg:translate-x-0 fixed lg:sticky top-0 z-40 h-screen w-64 bg-bg-card border border-border flex flex-col transition-transform duration-200`}
        style={{ insetInlineStart: 0 }}
      >
        <div className="h-16 flex items-center gap-2 px-5 border-b border-border">
          <ShieldCheck className="text-accent" size={28} />
          <div className="flex flex-col leading-tight">
            <span className="font-bold text-text-primary">{t('app.name')}</span>
            <span className="text-xs text-text-muted">{t('app.tagline')}</span>
          </div>
        </div>

        <nav className="flex-1 p-3 space-y-1 overflow-y-auto">
          {links.map((l) => (
            <NavLink
              key={l.to}
              to={l.to}
              end={l.end}
              onClick={closeMobile}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-accent text-white'
                    : 'text-text-secondary hover:bg-bg-hover hover:text-text-primary'
                }`
              }
            >
              <l.icon size={20} />
              <span>{l.label}</span>
            </NavLink>
          ))}
        </nav>

        <div className="p-3 border-t border-border">
          <button
            onClick={handleLogout}
            className="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium text-text-secondary hover:bg-bg-hover hover:text-text-primary transition-colors"
          >
            <LogOut size={20} />
            <span>{t('nav.logout')}</span>
          </button>
        </div>
      </aside>
    </>
  )
}