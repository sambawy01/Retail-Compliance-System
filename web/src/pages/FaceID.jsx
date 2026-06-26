import { useEffect, useState, useCallback } from 'react'
import { useLang } from '../contexts/LanguageContext'
import { apiGet, apiPost, api } from '../services/api'
import { fmtDate } from '../services/constants'
import { ScanFace, UserPlus, X, ShieldCheck, FileText, Ban, ChevronDown, ChevronUp } from 'lucide-react'

export default function FaceID() {
  const { t } = useLang()
  const [persons, setPersons] = useState([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({ name: '', kind: 'employee' })
  const [formError, setFormError] = useState(null)
  const [formMsg, setFormMsg] = useState(null)
  const [expanded, setExpanded] = useState(null) // person id for consent/audit
  const [audit, setAudit] = useState({})
  const [consent, setConsent] = useState({})

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const d = await apiGet.persons('employee')
      const arr = Array.isArray(d) ? d : d.items || d.persons || []
      setPersons(arr)
    } catch { /* ignore */ }
    setLoading(false)
  }, [])

  useEffect(() => { load() }, [load])

  const enroll = async (e) => {
    e.preventDefault()
    setFormError(null); setFormMsg(null)
    try {
      await apiPost.person({ name: form.name, kind: form.kind })
      setFormMsg(t('faceid.form.success'))
      setForm({ name: '', kind: 'employee' })
      setShowForm(false)
      load()
    } catch {
      setFormError(t('faceid.form.error'))
    }
  }

  const revoke = async (p) => {
    if (!confirm(t('faceid.revokeConfirm'))) return
    try {
      await api.delete(`/identity/persons/${p.id}`)
      load()
    } catch { /* ignore */ }
  }

  const toggleConsent = async (p) => {
    if (expanded === p.id) { setExpanded(null); return }
    setExpanded(p.id)
    try {
      const res = await api.get(`/identity/persons/${p.id}/consent`)
      setConsent((s) => ({ ...s, [p.id]: res.data }))
    } catch { setConsent((s) => ({ ...s, [p.id]: null })) }
    try {
      const res = await api.get(`/identity/persons/${p.id}/audit`)
      setAudit((s) => ({ ...s, [p.id]: Array.isArray(res.data) ? res.data : res.data.items || [] }))
    } catch { setAudit((s) => ({ ...s, [p.id]: [] })) }
  }

  const kindLabel = (k) => t(`faceid.kind${k ? k.charAt(0).toUpperCase() + k.slice(1) : ''}`, k || '')
  const consentBadge = (p) => {
    const s = p.consent_status || p.consent || 'pending'
    const map = {
      given: { c: 'bg-success/15 text-success border-success/40' },
      revoked: { c: 'bg-critical/15 text-critical border-critical/40' },
      pending: { c: 'bg-warning/15 text-warning border-warning/40' },
    }
    const m = map[s] || map.pending
    return <span className={`inline-block px-2 py-0.5 rounded-full text-xs font-semibold border ${m.c}`}>{t(`faceid.consent${s.charAt(0).toUpperCase() + s.slice(1)}`, s)}</span>
  }

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold text-text-primary">{t('faceid.title')}</h1>
        <button onClick={() => setShowForm(true)}
          className="flex items-center gap-2 px-4 py-2 text-sm font-medium bg-accent hover:bg-accent-hover text-white rounded-lg">
          <UserPlus size={16} /> {t('faceid.enrollNew')}
        </button>
      </div>

      {/* Person list */}
      <div className="bg-bg-card border border-border rounded-xl overflow-hidden">
        <div className="px-4 py-3 border-b border-border">
          <h2 className="font-semibold text-text-primary">{t('faceid.enrolledPersons')}</h2>
        </div>
        {loading ? (
          <div className="p-8 text-center text-text-muted">{t('common.loading')}</div>
        ) : persons.length === 0 ? (
          <div className="p-8 text-center text-text-muted">{t('faceid.noPersons')}</div>
        ) : (
          <div className="divide-y divide-border">
            {persons.map((p) => (
              <div key={p.id}>
                <div className="p-4 flex items-center justify-between gap-4 card-hover">
                  <div className="flex items-center gap-3 min-w-0">
                    <div className="w-10 h-10 rounded-full bg-accent/15 text-accent flex items-center justify-center flex-shrink-0">
                      <ScanFace size={20} />
                    </div>
                    <div className="min-w-0">
                      <div className="text-sm font-medium text-text-primary truncate">{p.name}</div>
                      <div className="text-xs text-text-muted">{kindLabel(p.kind)} · {fmtDate(p.enrolled_at || p.created_at)}</div>
                    </div>
                  </div>
                  <div className="flex items-center gap-2 flex-shrink-0">
                    {consentBadge(p)}
                    <button onClick={() => toggleConsent(p)}
                      className="p-1.5 rounded-lg border border-border text-text-secondary hover:bg-bg-hover"
                      title={t('faceid.viewConsent')}>
                      {expanded === p.id ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
                    </button>
                    <button onClick={() => revoke(p)}
                      className="p-1.5 rounded-lg border border-critical/40 text-critical hover:bg-critical/10"
                      title={t('faceid.revoke')}>
                      <Ban size={16} />
                    </button>
                  </div>
                </div>
                {expanded === p.id && (
                  <div className="px-4 pb-4 grid grid-cols-1 md:grid-cols-2 gap-4">
                    {/* Consent record */}
                    <div className="bg-bg border border-border rounded-lg p-4">
                      <div className="flex items-center gap-2 mb-2">
                        <FileText size={14} className="text-text-secondary" />
                        <span className="text-xs font-semibold text-text-secondary">{t('faceid.viewConsent')}</span>
                      </div>
                      {consent[p.id] ? (
                        <pre className="text-xs text-text-secondary overflow-x-auto max-h-40">
{JSON.stringify(consent[p.id], null, 2)}
                        </pre>
                      ) : <div className="text-xs text-text-muted">—</div>}
                    </div>
                    {/* Audit log */}
                    <div className="bg-bg border border-border rounded-lg p-4">
                      <div className="flex items-center gap-2 mb-2">
                        <ShieldCheck size={14} className="text-text-secondary" />
                        <span className="text-xs font-semibold text-text-secondary">{t('faceid.auditLog')}</span>
                      </div>
                      {audit[p.id] && audit[p.id].length > 0 ? (
                        <div className="space-y-1 max-h-40 overflow-y-auto">
                          {audit[p.id].map((a, i) => (
                            <div key={i} className="text-xs text-text-secondary">
                              <span className="text-text-muted">{fmtDate(a.timestamp || a.created_at)}</span> — {a.action || a.event || 'access'}
                            </div>
                          ))}
                        </div>
                      ) : <div className="text-xs text-text-muted">—</div>}
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Enroll form modal */}
      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60" onClick={() => setShowForm(false)}>
          <div className="bg-bg-card border border-border rounded-xl w-full max-w-md" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-5 py-3 border-b border-border">
              <h3 className="font-semibold text-text-primary">{t('faceid.enrollNew')}</h3>
              <button onClick={() => setShowForm(false)} className="text-text-muted hover:text-text-primary"><X size={18} /></button>
            </div>
            <form onSubmit={enroll} className="p-5 space-y-4">
              <div>
                <label className="block text-xs font-medium text-text-secondary mb-1.5">{t('faceid.form.name')}</label>
                <input required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })}
                  className="w-full bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-accent" />
              </div>
              <div>
                <label className="block text-xs font-medium text-text-secondary mb-1.5">{t('faceid.form.kind')}</label>
                <select value={form.kind} onChange={(e) => setForm({ ...form, kind: e.target.value })}
                  className="w-full bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-accent">
                  <option value="employee">{t('faceid.kindEmployee')}</option>
                  <option value="contractor">{t('faceid.kindContractor')}</option>
                  <option value="visitor">{t('faceid.kindVisitor')}</option>
                </select>
              </div>
              {formError && <div className="text-sm text-critical">{formError}</div>}
              {formMsg && <div className="text-sm text-success">{formMsg}</div>}
              <div className="flex gap-2 justify-end">
                <button type="button" onClick={() => setShowForm(false)} className="px-4 py-2 text-sm font-medium border border-border text-text-secondary hover:bg-bg-hover rounded-lg">{t('faceid.form.cancel')}</button>
                <button type="submit" className="px-4 py-2 text-sm font-medium bg-accent hover:bg-accent-hover text-white rounded-lg">{t('faceid.form.submit')}</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}