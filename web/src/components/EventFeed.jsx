import { useEffect, useRef, useState, useCallback } from 'react'
import { useLang } from '../contexts/LanguageContext'
import { apiGet, connectWebSocket } from '../services/api'
import { reportError } from '../services/errors'
import { sevColor, severityOf, fmtTime } from '../services/constants'
import { Activity } from 'lucide-react'

export default function EventFeed({ limit = 20, cameraId }) {
  const { t } = useLang()
  const [events, setEvents] = useState([])
  const [connecting, setConnecting] = useState(false)
  const wsRef = useRef(null)
  const pollRef = useRef(null)
  const reconnectRef = useRef(null)
  const reconnectAttempts = useRef(0)

  const fetchEvents = useCallback(async () => {
    try {
      const data = await apiGet.detections({ camera_id: cameraId, limit })
      const arr = Array.isArray(data) ? data : data.items || data.detections || []
      setEvents(arr.slice(0, limit))
    } catch (e) { reportError(e, 'fetching events') }
  }, [cameraId, limit])

  useEffect(() => {
    fetchEvents()

    // Try WS with reconnection; fall back to polling if WS unavailable
    const connect = () => {
      setConnecting(true)
      wsRef.current = connectWebSocket(
        (msg) => {
          // On message: reset reconnect attempts, clear connecting state
          reconnectAttempts.current = 0
          setConnecting(false)
          if (msg && (msg.type === 'detection' || msg.event_type)) {
            setEvents((prev) => [msg, ...prev].slice(0, limit))
          }
        },
        () => {
          // On open: reset reconnect, clear connecting
          reconnectAttempts.current = 0
          setConnecting(false)
          // Clear any polling fallback
          if (pollRef.current) {
            clearInterval(pollRef.current)
            pollRef.current = null
          }
        },
        () => {
          // On close: attempt reconnection with exponential backoff
          setConnecting(false)
          wsRef.current = null
          if (reconnectAttempts.current < 5) {
            // Add random jitter (0-2s) to prevent thundering herd on reconnection
            const jitter = Math.random() * 2000
            const delay = Math.min(1000 * 2 ** reconnectAttempts.current, 30000) + jitter
            reconnectAttempts.current++
            reconnectRef.current = setTimeout(connect, delay)
          } else {
            // Max reconnects exceeded — fall back to polling
            if (!pollRef.current) {
              pollRef.current = setInterval(fetchEvents, 15000)
            }
          }
        }
      )

      if (!wsRef.current) {
        // WS unavailable — poll every 15s
        setConnecting(false)
        if (!pollRef.current) {
          pollRef.current = setInterval(fetchEvents, 15000)
        }
      }
    }

    connect()

    return () => {
      if (wsRef.current) { try { wsRef.current.close() } catch {} }
      if (pollRef.current) clearInterval(pollRef.current)
      if (reconnectRef.current) clearTimeout(reconnectRef.current)
    }
  }, [fetchEvents, limit])

  return (
    <div className="bg-bg-card border border-border rounded-xl flex flex-col h-full">
      <div className="flex items-center gap-2 px-4 py-3 border-b border-border">
        <Activity size={16} className="text-accent" />
        <h3 className="text-sm font-semibold text-text-primary">{t('dashboard.recentEvents')}</h3>
        {connecting && (
          <span className="text-xs text-text-muted ml-auto">{t('common.loading')}</span>
        )}
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
    </div>
  )
}
