import { useState, useEffect } from 'react'
import { useLang } from '../contexts/LanguageContext'
import { ALL_EVENT_TYPES } from '../services/constants'
import { Bell, Plus, Trash2, Save, Settings as Cog } from 'lucide-react'

const CHANNELS = ['email', 'sms', 'webhook', 'slack', 'push']

export default function Settings() {
  const { t } = useLang()
  const [rules, setRules] = useState([])
  const [org, setOrg] = useState({ name: '', timezone: '' })
  const [savedMsg, setSavedMsg] = useState('')

  // Load from localStorage (the backend may not have these endpoints yet)
  useEffect(() => {
    try {
      const r = JSON.parse(localStorage.getItem('watchdog_rules') || '[]')
      setRules(Array.isArray(r) ? r : [])
      const o = JSON.parse(localStorage.getItem('watchdog_org') || '{}')
      setOrg({ name: o.name || '', timezone: o.timezone || '' })
    } catch { /* ignore */ }
  }, [])

  const persist = (r) => {
    setRules(r)
    localStorage.setItem('watchdog_rules', JSON.stringify(r))
  }

  const addRule = () => {
    persist([...rules, { id: Date.now(), event_type: '', channel: 'email', target: '' }])
  }
  const updateRule = (id, field, val) => {
    persist(rules.map((r) => (r.id === id ? { ...r, [field]: val } : r)))
  }
  const deleteRule = (id) => {
    persist(rules.filter((r) => r.id !== id))
  }

  const saveOrg = () => {
    localStorage.setItem('watchdog_org', JSON.stringify(org))
    setSavedMsg(t('settings.saved'))
    setTimeout(() => setSavedMsg(''), 2000)
  }

  return (
    <div className="space-y-5">
      <h1 className="text-xl font-bold text-text-primary">{t('settings.title')}</h1>

      {/* Notification rules */}
      <div className="bg-bg-card border border-border rounded-xl">
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <div className="flex items-center gap-2">
            <Bell size={16} className="text-accent" />
            <h2 className="font-semibold text-text-primary">{t('settings.notificationRules')}</h2>
          </div>
          <button onClick={addRule} className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium bg-accent hover:bg-accent-hover text-white rounded-lg">
            <Plus size={14} /> {t('settings.addRule')}
          </button>
        </div>
        <div className="divide-y divide-border">
          {rules.length === 0 && (
            <div className="p-6 text-center text-text-muted text-sm">{t('settings.noRules')}</div>
          )}
          {rules.map((r) => (
            <div key={r.id} className="p-4 grid grid-cols-1 sm:grid-cols-4 gap-2 items-center">
              <select value={r.event_type} onChange={(e) => updateRule(r.id, 'event_type', e.target.value)}
                className="bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-accent">
                <option value="">— {t('settings.eventType')} —</option>
                {ALL_EVENT_TYPES.map((et) => <option key={et} value={et}>{t(`eventTypes.${et}`, et)}</option>)}
              </select>
              <select value={r.channel} onChange={(e) => updateRule(r.id, 'channel', e.target.value)}
                className="bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-accent">
                {CHANNELS.map((c) => <option key={c} value={c}>{t(`settings.channel${c.charAt(0).toUpperCase() + c.slice(1)}`, c)}</option>)}
              </select>
              <input value={r.target} onChange={(e) => updateRule(r.id, 'target', e.target.value)}
                placeholder={t('settings.target')}
                className="bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-accent" />
              <button onClick={() => deleteRule(r.id)}
                className="flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium border border-critical/40 text-critical hover:bg-critical/10 rounded-lg">
                <Trash2 size={14} /> {t('settings.delete')}
              </button>
            </div>
          ))}
        </div>
      </div>

      {/* Org settings */}
      <div className="bg-bg-card border border-border rounded-xl">
        <div className="flex items-center gap-2 px-4 py-3 border-b border-border">
          <Cog size={16} className="text-accent" />
          <h2 className="font-semibold text-text-primary">{t('settings.orgSettings')}</h2>
        </div>
        <div className="p-4 grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-xs font-medium text-text-secondary mb-1.5">{t('settings.orgName')}</label>
            <input value={org.name} onChange={(e) => setOrg({ ...org, name: e.target.value })}
              className="w-full bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-accent" />
          </div>
          <div>
            <label className="block text-xs font-medium text-text-secondary mb-1.5">{t('settings.timezone')}</label>
            <input value={org.timezone} onChange={(e) => setOrg({ ...org, timezone: e.target.value })}
              placeholder="e.g. Asia/Dubai"
              className="w-full bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-accent" />
          </div>
        </div>
        <div className="px-4 pb-4 flex items-center gap-3">
          <button onClick={saveOrg} className="flex items-center gap-1.5 px-4 py-2 text-sm font-medium bg-accent hover:bg-accent-hover text-white rounded-lg">
            <Save size={14} /> {t('settings.save')}
          </button>
          {savedMsg && <span className="text-sm text-success">{savedMsg}</span>}
        </div>
      </div>
    </div>
  )
}