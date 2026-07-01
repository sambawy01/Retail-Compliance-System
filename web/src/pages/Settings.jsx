import { useState, useEffect, useCallback } from 'react'
import { useLang } from '../contexts/LanguageContext'
import { apiGet, apiPost, apiPatch, apiDelete } from '../services/api'
import { ALL_EVENT_TYPES, SEVERITY } from '../services/constants'
import { Bell, Plus, Trash2, Save, Settings as Cog, Loader2, AlertCircle } from 'lucide-react'

// Channels must match the CHECK constraint in migration 005:
// telegram, email, sms, dashboard
const CHANNELS = ['telegram', 'email', 'sms', 'dashboard']
const SEVERITIES = [SEVERITY.critical, SEVERITY.warning, SEVERITY.info]

export default function Settings() {
  const { t } = useLang()
  const [rules, setRules] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [org, setOrg] = useState({ name: '', timezone: '' })
  const [savedMsg, setSavedMsg] = useState('')

  // Load notification rules from backend
  const loadRules = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const d = await apiGet.notificationRules()
      const arr = Array.isArray(d) ? d : d.rules || []
      setRules(arr)
    } catch (e) {
      setError(e.response?.data?.error || 'Failed to load notification rules')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadRules()
    // Org settings still in localStorage until backend adds an org endpoint
    try {
      const o = JSON.parse(localStorage.getItem('watchdog_org') || '{}')
      setOrg({ name: o.name || '', timezone: o.timezone || '' })
    } catch { /* ignore */ }
  }, [loadRules])

  const addRule = async () => {
    // Optimistically add a placeholder row, then create on backend
    const newRule = {
      event_type: ALL_EVENT_TYPES[0],
      severity: SEVERITY.critical,
      channel: 'telegram',
      target: '',
    }
    try {
      setError(null)
      const created = await apiPost.notificationRule(newRule)
      setRules((prev) => [...prev, created])
    } catch (e) {
      setError(e.response?.data?.error || 'Failed to create rule')
    }
  }

  const updateRule = async (id, field, val) => {
    // Optimistic update
    setRules((prev) => prev.map((r) => (r.rule_id === id ? { ...r, [field]: val } : r)))
    try {
      setError(null)
      const rule = rules.find((r) => r.rule_id === id)
      if (!rule) return
      await apiPatch.notificationRule(id, { [field]: val })
    } catch (e) {
      setError(e.response?.data?.error || 'Failed to update rule')
      // Revert on failure
      loadRules()
    }
  }

  const deleteRule = async (id) => {
    // Optimistic delete
    const prev = rules
    setRules((cur) => cur.filter((r) => r.rule_id !== id))
    try {
      setError(null)
      await apiDelete.notificationRule(id)
    } catch (e) {
      setError(e.response?.data?.error || 'Failed to delete rule')
      setRules(prev) // revert
    }
  }

  const saveOrg = () => {
    localStorage.setItem('watchdog_org', JSON.stringify(org))
    setSavedMsg(t('settings.saved'))
    setTimeout(() => setSavedMsg(''), 2000)
  }

  return (
    <div className="space-y-5">
      <h1 className="text-xl font-bold text-text-primary">{t('settings.title')}</h1>

      {error && (
        <div className="flex items-center gap-2 text-sm text-critical bg-critical/10 border border-critical/30 rounded-lg px-3 py-2">
          <AlertCircle size={16} /> {error}
        </div>
      )}

      {/* Notification rules */}
      <div className="bg-bg-card border border-border rounded-xl">
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <div className="flex items-center gap-2">
            <Bell size={16} className="text-accent" />
            <h2 className="font-semibold text-text-primary">{t('settings.notificationRules')}</h2>
          </div>
          <button onClick={addRule}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium bg-accent hover:bg-accent-hover text-white rounded-lg">
            <Plus size={14} /> {t('settings.addRule')}
          </button>
        </div>
        {loading ? (
          <div className="p-8 text-center text-text-muted">
            <Loader2 size={20} className="animate-spin inline-block me-2" />
            {t('common.loading')}
          </div>
        ) : (
          <div className="divide-y divide-border">
            {rules.length === 0 && (
              <div className="p-6 text-center text-text-muted text-sm">{t('settings.noRules')}</div>
            )}
            {rules.map((r) => (
              <div key={r.rule_id} className="p-4 grid grid-cols-1 sm:grid-cols-5 gap-2 items-center">
                <select value={r.event_type}
                  onChange={(e) => updateRule(r.rule_id, 'event_type', e.target.value)}
                  className="bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-accent">
                  {ALL_EVENT_TYPES.map((et) => <option key={et} value={et}>{t(`eventTypes.${et}`, et)}</option>)}
                </select>
                <select value={r.severity}
                  onChange={(e) => updateRule(r.rule_id, 'severity', e.target.value)}
                  className="bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-accent">
                  {SEVERITIES.map((s) => <option key={s} value={s}>{t(`severity.${s}`, s)}</option>)}
                </select>
                <select value={r.channel}
                  onChange={(e) => updateRule(r.rule_id, 'channel', e.target.value)}
                  className="bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-accent">
                  {CHANNELS.map((c) => <option key={c} value={c}>{t(`settings.channel${c.charAt(0).toUpperCase() + c.slice(1)}`, c)}</option>)}
                </select>
                <input value={r.target}
                  onChange={(e) => updateRule(r.rule_id, 'target', e.target.value)}
                  placeholder={t('settings.target')}
                  className="bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-accent" />
                <button onClick={() => deleteRule(r.rule_id)}
                  className="flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium border border-critical/40 text-critical hover:bg-critical/10 rounded-lg">
                  <Trash2 size={14} /> {t('settings.delete')}
                </button>
              </div>
            ))}
          </div>
        )}
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
          <button onClick={saveOrg}
            className="flex items-center gap-1.5 px-4 py-2 text-sm font-medium bg-accent hover:bg-accent-hover text-white rounded-lg">
            <Save size={14} /> {t('settings.save')}
          </button>
          {savedMsg && <span className="text-sm text-success">{savedMsg}</span>}
        </div>
      </div>
    </div>
  )
}