import { useEffect, useState, useMemo } from 'react'
import { useLang } from '../contexts/LanguageContext'
import { apiGet } from '../services/api'
import { ALL_EVENT_TYPES, severityOf, fmtTime } from '../services/constants'
import SeverityBadge from '../components/SeverityBadge'
import { Filter, ChevronLeft, ChevronRight, X, ChevronUp, ChevronDown } from 'lucide-react'

const PAGE_SIZE = 20
const SEVERITIES = ['critical', 'warning', 'info']

export default function Events() {
  const { t, isRTL } = useLang()
  const [cameras, setCameras] = useState([])
  const [detections, setDetections] = useState([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [filters, setFilters] = useState({ camera: '', eventType: '', severity: '', dateFrom: '', dateTo: '' })
  const [applied, setApplied] = useState(filters)
  const [selected, setSelected] = useState(null)
  const [clip, setClip] = useState(null)
  const [clipLoading, setClipLoading] = useState(false)

  useEffect(() => {
    apiGet.cameras().then((d) => setCameras(Array.isArray(d) ? d : d.items || [])).catch(() => {})
  }, [])

  useEffect(() => {
    let active = true
    setLoading(true)
    const params = { limit: 500 }
    if (applied.camera) params.camera_id = applied.camera
    if (applied.eventType) params.event_type = applied.eventType
    apiGet.detections(params)
      .then((d) => { if (active) setDetections(Array.isArray(d) ? d : d.items || d.detections || []) })
      .catch(() => { if (active) setDetections([]) })
      .finally(() => { if (active) setLoading(false) })
    return () => { active = false }
  }, [applied])

  const filtered = useMemo(() => {
    return detections.filter((d) => {
      const sev = severityOf(d.event_type)
      if (applied.severity && sev !== applied.severity) return false
      const ts = new Date(d.timestamp || d.created_at)
      if (applied.dateFrom && ts < new Date(applied.dateFrom + 'T00:00:00')) return false
      if (applied.dateTo && ts > new Date(applied.dateTo + 'T23:59:59')) return false
      return true
    })
  }, [detections, applied])

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE))
  const safePage = Math.min(page, totalPages)
  const pageItems = filtered.slice((safePage - 1) * PAGE_SIZE, safePage * PAGE_SIZE)

  const applyFilters = () => { setPage(1); setApplied(filters) }
  const resetFilters = () => { setFilters({ camera: '', eventType: '', severity: '', dateFrom: '', dateTo: '' }); setApplied({ camera: '', eventType: '', severity: '', dateFrom: '', dateTo: '' }); setPage(1) }

  const openDetail = async (e) => {
    setSelected(e)
    setClip(null)
    if (e.camera_id || e.cameraId) {
      setClipLoading(true)
      apiGet.clips(e.camera_id || e.cameraId)
        .then((d) => {
          const arr = Array.isArray(d) ? d : d.items || d.clips || []
          const match = arr.find((c) => c.detection_id === e.id) || arr[0]
          if (match) setClip(match)
        })
        .catch(() => {})
        .finally(() => setClipLoading(false))
    }
  }

  const FlipIcon = isRTL ? ChevronRight : ChevronLeft
  const NextIcon = isRTL ? ChevronLeft : ChevronRight
  const PrevIcon = isRTL ? ChevronRight : ChevronLeft

  return (
    <div className="space-y-5">
      <h1 className="text-xl font-bold text-text-primary">{t('events.title')}</h1>

      {/* Filters */}
      <div className="bg-bg-card border border-border rounded-xl p-4">
        <div className="flex items-center gap-2 mb-3">
          <Filter size={16} className="text-accent" />
          <span className="text-sm font-semibold text-text-primary">{t('events.filters')}</span>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3">
          <select value={filters.camera} onChange={(e) => setFilters({ ...filters, camera: e.target.value })}
            className="bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-accent">
            <option value="">{t('events.all')} — {t('events.camera')}</option>
            {cameras.map((c) => <option key={c.id} value={c.id}>{c.name || `#${c.id}`}</option>)}
          </select>
          <select value={filters.eventType} onChange={(e) => setFilters({ ...filters, eventType: e.target.value })}
            className="bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-accent">
            <option value="">{t('events.all')} — {t('events.eventType')}</option>
            {ALL_EVENT_TYPES.map((et) => <option key={et} value={et}>{t(`eventTypes.${et}`, et)}</option>)}
          </select>
          <select value={filters.severity} onChange={(e) => setFilters({ ...filters, severity: e.target.value })}
            className="bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-accent">
            <option value="">{t('events.all')} — {t('events.severity')}</option>
            {SEVERITIES.map((s) => <option key={s} value={s}>{t(`severity.${s}`, s)}</option>)}
          </select>
          <input type="date" value={filters.dateFrom} onChange={(e) => setFilters({ ...filters, dateFrom: e.target.value })}
            className="bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-accent" />
          <input type="date" value={filters.dateTo} onChange={(e) => setFilters({ ...filters, dateTo: e.target.value })}
            className="bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text-primary focus:outline-none focus:border-accent" />
        </div>
        <div className="flex gap-2 mt-3">
          <button onClick={applyFilters} className="px-4 py-2 text-sm font-medium bg-accent hover:bg-accent-hover text-white rounded-lg">{t('events.apply')}</button>
          <button onClick={resetFilters} className="px-4 py-2 text-sm font-medium border border-border text-text-secondary hover:bg-bg-hover rounded-lg">{t('events.reset')}</button>
        </div>
      </div>

      {/* Table */}
      <div className="bg-bg-card border border-border rounded-xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="bg-bg-hover text-text-secondary text-xs uppercase">
              <tr>
                <th className="text-start px-4 py-3 font-medium">{t('events.timestamp')}</th>
                <th className="text-start px-4 py-3 font-medium">{t('events.eventType')}</th>
                <th className="text-start px-4 py-3 font-medium">{t('events.severity')}</th>
                <th className="text-start px-4 py-3 font-medium">{t('events.cameraName')}</th>
                <th className="text-start px-4 py-3 font-medium">{t('events.confidence')}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {loading && (
                <tr><td colSpan={5} className="px-4 py-8 text-center text-text-muted">{t('common.loading')}</td></tr>
              )}
              {!loading && pageItems.length === 0 && (
                <tr><td colSpan={5} className="px-4 py-8 text-center text-text-muted">{t('events.noEvents')}</td></tr>
              )}
              {pageItems.map((e, i) => (
                <tr key={e.id || i} onClick={() => openDetail(e)} className="cursor-pointer card-hover">
                  <td className="px-4 py-3 text-text-secondary whitespace-nowrap">{fmtTime(e.timestamp || e.created_at)}</td>
                  <td className="px-4 py-3 text-text-primary">{t(`eventTypes.${e.event_type}`, e.event_type)}</td>
                  <td className="px-4 py-3"><SeverityBadge severity={severityOf(e.event_type)} /></td>
                  <td className="px-4 py-3 text-text-secondary">{e.camera_name || e.camera_id || '—'}</td>
                  <td className="px-4 py-3 text-text-muted">{e.confidence != null ? `${Math.round(e.confidence * 100)}%` : '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between px-4 py-3 border-t border-border">
            <span className="text-xs text-text-muted">{t('events.page')} {safePage} {t('events.of')} {totalPages}</span>
            <div className="flex gap-2">
              <button disabled={safePage <= 1} onClick={() => setPage(safePage - 1)}
                className="p-1.5 rounded-lg border border-border text-text-secondary hover:bg-bg-hover disabled:opacity-40">
                <PrevIcon size={16} className={isRTL ? 'rtl-flip' : ''} />
              </button>
              <button disabled={safePage >= totalPages} onClick={() => setPage(safePage + 1)}
                className="p-1.5 rounded-lg border border-border text-text-secondary hover:bg-bg-hover disabled:opacity-40">
                <NextIcon size={16} className={isRTL ? 'rtl-flip' : ''} />
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Detail modal */}
      {selected && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60" onClick={() => setSelected(null)}>
          <div className="bg-bg-card border border-border rounded-xl w-full max-w-lg max-h-[90vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-5 py-3 border-b border-border sticky top-0 bg-bg-card">
              <h3 className="font-semibold text-text-primary">{t('events.detail')}</h3>
              <button onClick={() => setSelected(null)} className="text-text-muted hover:text-text-primary"><X size={18} /></button>
            </div>
            <div className="p-5 space-y-4">
              <div className="space-y-2 text-sm">
                <Row label={t('events.eventType')} value={t(`eventTypes.${selected.event_type}`, selected.event_type)} />
                <Row label={t('events.severity')} value={<SeverityBadge severity={severityOf(selected.event_type)} />} />
                <Row label={t('events.timestamp')} value={fmtTime(selected.timestamp || selected.created_at)} />
                <Row label={t('events.cameraName')} value={selected.camera_name || selected.camera_id || '—'} />
                <Row label={t('events.confidence')} value={selected.confidence != null ? `${Math.round(selected.confidence * 100)}%` : '—'} />
              </div>

              {/* Clip thumbnail */}
              <div>
                <div className="text-xs text-text-secondary mb-2">{t('events.clip')}</div>
                {clipLoading ? (
                  <div className="aspect-video bg-black/40 rounded-lg flex items-center justify-center text-text-muted text-sm">{t('common.loading')}</div>
                ) : clip ? (
                  <div className="aspect-video bg-black rounded-lg overflow-hidden">
                    {clip.thumbnail_url ? (
                      <img src={clip.thumbnail_url} alt="clip" className="w-full h-full object-cover" />
                    ) : clip.url ? (
                      <video src={clip.url} controls className="w-full h-full object-cover" />
                    ) : (
                      <div className="w-full h-full flex items-center justify-center text-text-muted text-sm">{t('events.noClip')}</div>
                    )}
                  </div>
                ) : (
                  <div className="aspect-video bg-black/40 rounded-lg flex items-center justify-center text-text-muted text-sm">{t('events.noClip')}</div>
                )}
              </div>

              {/* Payload */}
              <div>
                <div className="text-xs text-text-secondary mb-2">{t('events.payload')}</div>
                <pre className="bg-bg border border-border rounded-lg p-3 text-xs text-text-secondary overflow-x-auto max-h-48">
{JSON.stringify(selected.payload || selected.metadata || selected, null, 2)}
                </pre>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function Row({ label, value }) {
  return (
    <div className="flex justify-between gap-4">
      <span className="text-text-secondary">{label}</span>
      <span className="text-text-primary text-end">{value}</span>
    </div>
  )
}