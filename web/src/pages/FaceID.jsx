import { useEffect, useState, useCallback, useRef } from 'react'
import { useLang } from '../contexts/LanguageContext'
import { apiGet, apiPost, api } from '../services/api'
import { reportError } from '../services/errors'
import { fmtDate } from '../services/constants'
import { ScanFace, UserPlus, X, ShieldCheck, FileText, Ban, ChevronDown, ChevronUp, Camera, Upload, Phone, Briefcase, CheckCircle, Loader2, AlertCircle } from 'lucide-react'

// Fields to redact from biometric consent/audit JSON before rendering to DOM
const SENSITIVE_FIELDS = ['embedding', 'template', 'image_hash', 'signature_sha256', 'password_hash', 'wrapped_key', 'kms_key_arn']

function redactBiometric(obj) {
  if (!obj || typeof obj !== 'object') return obj
  const out = Array.isArray(obj) ? [...obj] : { ...obj }
  for (const key of Object.keys(out)) {
    if (SENSITIVE_FIELDS.includes(key)) {
      out[key] = '[REDACTED]'
    } else if (typeof out[key] === 'object' && out[key] !== null) {
      out[key] = redactBiometric(out[key])
    }
  }
  return out
}

export default function FaceID() {
  const { t } = useLang()
  const [persons, setPersons] = useState([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({ display_name: '', kind: 'employee', phone: '', job_role: '', photo_url: '' })
  const [formError, setFormError] = useState(null)
  const [formMsg, setFormMsg] = useState(null)
  const [expanded, setExpanded] = useState(null)
  const [audit, setAudit] = useState({})
  const [consent, setConsent] = useState({})

  const load = useCallback(async (isRefresh = false) => {
    if (isRefresh) setRefreshing(true)
    else setLoading(true)
    try {
      const d = await apiGet.persons('')
      const arr = Array.isArray(d) ? d : d.items || d.persons || []
      setPersons(arr)
    } catch (e) { reportError(e, 'loading persons') }
    if (isRefresh) setRefreshing(false)
    else setLoading(false)
  }, [])

  useEffect(() => { load() }, [load])

  const enroll = async (e) => {
    e.preventDefault()
    setFormError(null); setFormMsg(null)
    if (!form.display_name.trim()) {
      setFormError(t('faceid.form.errorName'))
      return
    }
    if (!form.photo_url) {
      setFormError(t('faceid.form.errorPhoto'))
      return
    }
    try {
      await apiPost.person({
        display_name: form.display_name,
        kind: form.kind,
        phone: form.phone,
        job_role: form.job_role,
        photo_url: form.photo_url,
      })
      setFormMsg(t('faceid.form.success'))
      setForm({ display_name: '', kind: 'employee', phone: '', job_role: '', photo_url: '' })
      setShowForm(false)
      load(true)
    } catch {
      setFormError(t('faceid.form.error'))
    }
  }

  const revoke = async (p) => {
    if (!confirm(t('faceid.revokeConfirm'))) return
    try {
      await api.delete(`/identity/persons/${p.id || p.person_id}`)
      load(true)
    } catch (e) { reportError(e, 'loading persons') }
  }

  const toggleConsent = async (p) => {
    const pid = p.id || p.person_id
    if (expanded === pid) { setExpanded(null); return }
    setExpanded(pid)
    try {
      const res = await api.get(`/identity/persons/${pid}/consent`)
      setConsent((s) => ({ ...s, [pid]: res.data }))
    } catch (e) { reportError(e, 'fetching consent'); setConsent((s) => ({ ...s, [pid]: null })) }
    try {
      const res = await api.get(`/identity/persons/${pid}/audit`)
      setAudit((s) => ({ ...s, [pid]: Array.isArray(res.data) ? res.data : res.data.items || [] }))
    } catch (e) { reportError(e, 'fetching audit'); setAudit((s) => ({ ...s, [pid]: [] })) }
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
          <div className="p-8 text-center text-text-muted">
            <Loader2 size={20} className="animate-spin inline-block me-2" />
            {t('common.loading')}
          </div>
        ) : persons.length === 0 ? (
          <div className="p-8 text-center text-text-muted">
            <ScanFace size={36} className="mx-auto mb-3 opacity-40" />
            {t('faceid.noPersons')}
          </div>
        ) : (
          <div className="divide-y divide-border">
            {persons.map((p) => {
              const pid = p.person_id || p.id
              const photo = p.photo_url || p.photoURL
              return (
                <div key={pid}>
                  <div className="p-4 flex items-center justify-between gap-4 card-hover">
                    <div className="flex items-center gap-3 min-w-0">
                      {photo ? (
                        <img src={photo} alt={p.display_name || p.name}
                          className="w-10 h-10 rounded-full object-cover flex-shrink-0 border border-border" />
                      ) : (
                        <div className="w-10 h-10 rounded-full bg-accent/15 text-accent flex items-center justify-center flex-shrink-0">
                          <ScanFace size={20} />
                        </div>
                      )}
                      <div className="min-w-0">
                        <div className="text-sm font-medium text-text-primary truncate">{p.display_name || p.name}</div>
                        <div className="text-xs text-text-muted flex items-center gap-2">
                          <span>{kindLabel(p.kind)}</span>
                          {p.job_role && <span className="flex items-center gap-0.5"><Briefcase size={10} /> {p.job_role}</span>}
                          {p.phone && <span className="flex items-center gap-0.5"><Phone size={10} /> {p.phone}</span>}
                          <span>· {fmtDate(p.enrolled_at || p.created_at)}</span>
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 flex-shrink-0">
                      {consentBadge(p)}
                      <button onClick={() => toggleConsent({ id: pid, ...p })}
                        className="p-1.5 rounded-lg border border-border text-text-secondary hover:bg-bg-hover"
                        title={t('faceid.viewConsent')}>
                        {expanded === pid ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
                      </button>
                      <button onClick={() => revoke({ id: pid, ...p })}
                        className="p-1.5 rounded-lg border border-critical/40 text-critical hover:bg-critical/10"
                        title={t('faceid.revoke')}>
                        <Ban size={16} />
                      </button>
                    </div>
                  </div>
                  {expanded === pid && (
                    <div className="px-4 pb-4 grid grid-cols-1 md:grid-cols-2 gap-4">
                      {/* Consent record */}
                      <div className="bg-bg border border-border rounded-lg p-4">
                        <div className="flex items-center gap-2 mb-2">
                          <FileText size={14} className="text-text-secondary" />
                          <span className="text-xs font-semibold text-text-secondary">{t('faceid.viewConsent')}</span>
                        </div>
                        {consent[pid] ? (
                          <pre className="text-xs text-text-secondary overflow-x-auto max-h-40">
{JSON.stringify(redactBiometric(consent[pid]), null, 2)}
                          </pre>
                        ) : <div className="text-xs text-text-muted">—</div>}
                      </div>
                      {/* Audit log */}
                      <div className="bg-bg border border-border rounded-lg p-4">
                        <div className="flex items-center gap-2 mb-2">
                          <ShieldCheck size={14} className="text-text-secondary" />
                          <span className="text-xs font-semibold text-text-secondary">{t('faceid.auditLog')}</span>
                        </div>
                        {audit[pid] && audit[pid].length > 0 ? (
                          <div className="space-y-1 max-h-40 overflow-y-auto">
                            {audit[pid].map((a, i) => (
                              <div key={i} className="text-xs text-text-secondary">
                                <span className="text-text-muted">{fmtDate(a.timestamp || a.created_at || a.accessed_at)}</span> — {a.action || a.purpose || a.event || 'access'}
                              </div>
                            ))}
                          </div>
                        ) : <div className="text-xs text-text-muted">—</div>}
                      </div>
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </div>

      {/* Enroll form modal with webcam capture */}
      {showForm && (
        <FaceCaptureForm
          form={form}
          setForm={setForm}
          onSubmit={enroll}
          onClose={() => { setShowForm(false); setFormError(null); setFormMsg(null) }}
          error={formError}
          msg={formMsg}
          t={t}
        />
      )}
    </div>
  )
}

// --- Face Capture Form with Webcam ---
function FaceCaptureForm({ form, setForm, onSubmit, onClose, error, msg, t }) {
  const videoRef = useRef(null)
  const canvasRef = useRef(null)
  const [cameraOn, setCameraOn] = useState(false)
  const [cameraError, setCameraError] = useState(null)
  const [capturing, setCapturing] = useState(false)
  const streamRef = useRef(null)

  const startCamera = async () => {
    setCameraError(null)
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: 'user', width: { ideal: 640 }, height: { ideal: 480 } }
      })
      streamRef.current = stream
      if (videoRef.current) {
        videoRef.current.srcObject = stream
        await videoRef.current.play()
      }
      setCameraOn(true)
    } catch (e) {
      setCameraError(t('faceid.camera.error'))
    }
  }

  const stopCamera = () => {
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((track) => track.stop())
      streamRef.current = null
    }
    setCameraOn(false)
  }

  const capture = () => {
    if (!videoRef.current || !canvasRef.current) return
    setCapturing(true)
    const video = videoRef.current
    const canvas = canvasRef.current
    canvas.width = 320
    canvas.height = 320
    const ctx = canvas.getContext('2d')
    // Center-crop to square
    const minDim = Math.min(video.videoWidth, video.videoHeight)
    const sx = (video.videoWidth - minDim) / 2
    const sy = (video.videoHeight - minDim) / 2
    ctx.drawImage(video, sx, sy, minDim, minDim, 0, 0, 320, 320)
    const dataUrl = canvas.toDataURL('image/jpeg', 0.85)
    setForm((f) => ({ ...f, photo_url: dataUrl }))
    setCapturing(false)
    stopCamera()
  }

  const handleFileUpload = (e) => {
    const file = e.target.files?.[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = () => {
      setForm((f) => ({ ...f, photo_url: reader.result }))
    }
    reader.readAsDataURL(file)
  }

  useEffect(() => {
    return () => stopCamera()
  }, [])

  const retakePhoto = () => {
    setForm((f) => ({ ...f, photo_url: '' }))
    startCamera()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60" onClick={onClose}>
      <div className="bg-bg-card border border-border rounded-xl w-full max-w-md max-h-[90vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between px-5 py-3 border-b border-border sticky top-0 bg-bg-card z-10">
          <h3 className="font-semibold text-text-primary">{t('faceid.enrollNew')}</h3>
          <button onClick={onClose} className="text-text-muted hover:text-text-primary"><X size={18} /></button>
        </div>
        <form onSubmit={onSubmit} className="p-5 space-y-4">
          {/* Face capture area */}
          <div>
            <label className="block text-xs font-medium text-text-secondary mb-1.5">{t('faceid.form.photo')}</label>
            {form.photo_url ? (
              <div className="relative">
                <img src={form.photo_url} alt="captured" className="w-full aspect-square object-cover rounded-lg border border-border" />
                <button type="button" onClick={retakePhoto}
                  className="absolute top-2 end-2 px-2 py-1 text-xs bg-bg/80 border border-border rounded-lg text-text-primary hover:bg-bg">
                  {t('faceid.retake')}
                </button>
                <div className="absolute bottom-2 start-2 flex items-center gap-1 px-2 py-1 text-xs bg-success/80 text-white rounded-lg">
                  <CheckCircle size={12} /> {t('faceid.photoCaptured')}
                </div>
              </div>
            ) : cameraOn ? (
              <div className="relative aspect-square bg-black rounded-lg overflow-hidden">
                <video ref={videoRef} autoPlay playsInline muted className="w-full h-full object-cover" />
                <div className="absolute inset-0 pointer-events-none">
                  <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-48 h-48 border-2 border-accent/60 rounded-full" />
                </div>
                <button type="button" onClick={capture} disabled={capturing}
                  className="absolute bottom-3 left-1/2 -translate-x-1/2 flex items-center gap-2 px-4 py-2 bg-accent hover:bg-accent-hover text-white rounded-lg text-sm font-medium disabled:opacity-50">
                  {capturing ? <Loader2 size={16} className="animate-spin" /> : <Camera size={16} />}
                  {t('faceid.capture')}
                </button>
              </div>
            ) : (
              <div className="space-y-2">
                {cameraError && (
                  <div className="flex items-center gap-2 text-sm text-critical bg-critical/10 border border-critical/30 rounded-lg px-3 py-2">
                    <AlertCircle size={16} /> {cameraError}
                  </div>
                )}
                <div className="grid grid-cols-1 gap-2">
                  <button type="button" onClick={startCamera}
                    className="flex items-center justify-center gap-2 px-4 py-6 border-2 border-dashed border-border rounded-lg text-text-secondary hover:border-accent hover:text-accent transition-colors">
                    <Camera size={28} />
                    <span className="text-sm font-medium">{t('faceid.startCamera')}</span>
                  </button>
                  <div className="flex items-center gap-2">
                    <label className="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 border border-border rounded-lg text-text-secondary hover:bg-bg-hover text-sm cursor-pointer">
                      <Upload size={16} /> {t('faceid.uploadPhoto')}
                      <input type="file" accept="image/*" onChange={handleFileUpload} className="hidden" />
                    </label>
                  </div>
                </div>
              </div>
            )}
            <canvas ref={canvasRef} className="hidden" />
          </div>

          {/* Name */}
          <div>
            <label className="block text-xs font-medium text-text-secondary mb-1.5">{t('faceid.form.name')}</label>
            <input required value={form.display_name}
              onChange={(e) => setForm({ ...form, display_name: e.target.value })}
              className="w-full bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-accent"
              placeholder={t('faceid.form.namePlaceholder')} />
          </div>

          {/* Kind */}
          <div>
            <label className="block text-xs font-medium text-text-secondary mb-1.5">{t('faceid.form.kind')}</label>
            <select value={form.kind}
              onChange={(e) => setForm({ ...form, kind: e.target.value })}
              className="w-full bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-accent">
              <option value="employee">{t('faceid.kindEmployee')}</option>
              <option value="customer">{t('faceid.kindCustomer')}</option>
            </select>
          </div>

          {/* Phone */}
          <div>
            <label className="block text-xs font-medium text-text-secondary mb-1.5">{t('faceid.form.phone')}</label>
            <div className="relative">
              <Phone size={14} className="absolute top-1/2 -translate-y-1/2 text-text-muted" style={{ insetInlineStart: 12 }} />
              <input value={form.phone}
                onChange={(e) => setForm({ ...form, phone: e.target.value })}
                className="w-full bg-bg border border-border rounded-lg ps-9 pe-3 py-2 text-sm text-text-primary focus:outline-none focus:border-accent"
                placeholder="+20 1xx xxx xxxx" />
            </div>
          </div>

          {/* Job Role */}
          <div>
            <label className="block text-xs font-medium text-text-secondary mb-1.5">{t('faceid.form.jobRole')}</label>
            <div className="relative">
              <Briefcase size={14} className="absolute top-1/2 -translate-y-1/2 text-text-muted" style={{ insetInlineStart: 12 }} />
              <input value={form.job_role}
                onChange={(e) => setForm({ ...form, job_role: e.target.value })}
                className="w-full bg-bg border border-border rounded-lg ps-9 pe-3 py-2 text-sm text-text-primary focus:outline-none focus:border-accent"
                placeholder={t('faceid.form.jobRolePlaceholder')} />
            </div>
          </div>

          {/* Consent checkbox */}
          <label className="flex items-start gap-2 text-xs text-text-secondary cursor-pointer">
            <input type="checkbox" required className="mt-0.5" />
            <span>{t('faceid.form.consentText')}</span>
          </label>

          {error && <div className="text-sm text-critical flex items-center gap-2"><AlertCircle size={14} /> {error}</div>}
          {msg && <div className="text-sm text-success flex items-center gap-2"><CheckCircle size={14} /> {msg}</div>}
          <div className="flex gap-2 justify-end">
            <button type="button" onClick={onClose} className="px-4 py-2 text-sm font-medium border border-border text-text-secondary hover:bg-bg-hover rounded-lg">{t('faceid.form.cancel')}</button>
            <button type="submit" className="px-4 py-2 text-sm font-medium bg-accent hover:bg-accent-hover text-white rounded-lg">{t('faceid.form.submit')}</button>
          </div>
        </form>
      </div>
    </div>
  )
}