import { useEffect, useRef, useState, useCallback } from 'react'
import { useLang } from '../contexts/LanguageContext'
import { apiGet, connectWebSocket } from '../services/api'
import { sevColor, severityOf, fmtTime } from '../services/constants'
import { Activity } from 'lucide-react'

export default function EventFeed({ limit = 20, cameraId }) {
  const { t } = useLang()
  const [events, setEvents] = useState([])
  const [polling, setPolling] = useState(false)
  const wsRef = useRef(null)
  const pollRef = useRef(null)

  const fetchEvents = useCallback(async () => {
    try {
      const data = await apiGet.detections({ camera_id: cameraId, limit })
      const arr = Array.isArray(data) ? data : data.items || data.detections || []
      setEvents(arr.slice(0, limit))
    } catch { /* ignore — keep last state */ }
  }, [cameraId, limit])

  useEffect(() => {
    fetchEvents()
    setPolling(true)

    // Try WS; fall back to polling
    let closed = false
    wsRef.current = connectWebSocket(
      (msg) => {
        if (msg && (msg.type === 'detection' || msg.event_type)) {
          setEvents((prev) => [msg, ...prev].slice(0, limit))
        }
      },
      () => { closed = false },
      () => { closed = true }
    )

    if (!wsRef.current) {
      // WS unavailable — poll every 15s
      pollRef.current = setInterval(fetchEvents, 15000)
    }

    return () => {
      if (wsRef.current && !closed) { try { wsRef.current.close() } catch {} }
      if (pollRef.current) clearInterval(pollRef.current)
    }
  }, [fetchEvents, limit])

  return (
    <div className="bg-bg-card border border-border rounded-xl flex flex-col h-full">
      <div className="flex items-center gap-2 px-4 py-3 border-b border-border">
        <Activity size={16} className="text-accent" />
        <h3 className="text-sm font-semibold text-text-primary">{t('dashboard.recentEvents')}</h3>
      </div>
      <div className="flex-1 overflow-y-auto divide-y divide-border">
        {events.length === 0 && (
          <div className="p-4 text-sm text-text-muted text-center">{t('dashboard.noEvents')}</div>
        )}
        {events.map((e, i) => {
          const sev = severityOf(e.event_type)
          const c = sevColor(sev)
          return (
            <div key={e.id || i} className="p-3 flex items-start gap-3 card-hover">
              <span className="mt-1 w-2 h-2 rounded-full flex-shrink-0" style={{ backgroundColor: c }} />
              <div className="flex-1 min-w-0">
                <div className="flex items-center justify-between gap-2">
                  <span className="text-sm font-medium text-text-primary truncate">
                    {t(`eventTypes.${e.event_type}`, e.event_type)}
                  </span>
                  <span className="text-xs text-text-muted flex-shrink-0">{fmtTime(e.timestamp || e.created_at)}</span>
                </div>
                {e.camera_name && (
                  <span className="text-xs text-text-secondary truncate block">{e.camera_name}</span>
                )}
              </div>
            </div>
          )
        })}
      </div>
      {polling && (
        <div className="px-4 py-2 border-t border-border text-xs text-text-muted text-center">
          {t('common.loading')}
        </div>
      )}
    </div>
  )
}