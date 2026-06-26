import { useEffect, useRef, useState } from 'react'
import { useLang } from '../contexts/LanguageContext'
import { apiGet, apiPost } from '../services/api'
import { severityOf, sevColor, fmtTime } from '../services/constants'
import { Video, VideoOff, Camera } from 'lucide-react'

function StatusPill({ status }) {
  const { t } = useLang()
  const s = status || 'offline'
  const map = {
    online: { c: 'bg-success/15 text-success border-success/40' },
    offline: { c: 'bg-offline/15 text-offline border-offline/40' },
    degraded: { c: 'bg-degraded/15 text-degraded border-degraded/40' },
  }
  const m = map[s] || map.offline
  return (
    <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-semibold border ${m.c}`}>
      <span className="w-1.5 h-1.5 rounded-full bg-current" />
      {t(`common.${s}`, s)}
    </span>
  )
}

function ZoneOverlay({ zones }) {
  if (!zones || zones.length === 0) return null
  return (
    <svg className="absolute inset-0 w-full h-full pointer-events-none" preserveAspectRatio="none" viewBox="0 0 100 100">
      {zones.map((z, i) => {
        const pts = z.polygon || z.points || z.coordinates
        if (!Array.isArray(pts) || pts.length < 3) return null
        const poly = pts.map((p) => `${p[0]},${p[1]}`).join(' ')
        return (
          <polygon
            key={z.id || i}
            points={poly}
            fill="rgba(59,130,246,0.10)"
            stroke="rgba(59,130,246,0.85)"
            strokeWidth="0.4"
          />
        )
      })}
    </svg>
  )
}

export default function CameraCard({ camera, onClick, showStream = false }) {
  const { t } = useLang()
  const videoRef = useRef(null)
  const pcRef = useRef(null)
  const [streamError, setStreamError] = useState(false)
  const [streamLoading, setStreamLoading] = useState(showStream)
  const [zones, setZones] = useState([])
  const [recent, setRecent] = useState([])

  // Load zones + last 5 events for this camera
  useEffect(() => {
    let active = true
    apiGet.zones(camera.id)
      .then((d) => { if (active) setZones(Array.isArray(d) ? d : d.items || []) })
      .catch(() => {})
    apiGet.detections({ camera_id: camera.id, limit: 5 })
      .then((d) => {
        if (active) {
          const arr = Array.isArray(d) ? d : d.items || d.detections || []
          setRecent(arr.slice(0, 5))
        }
      })
      .catch(() => {})
    return () => { active = false }
  }, [camera.id])

  // WebRTC: start stream when showStream is true
  useEffect(() => {
    if (!showStream) return
    let active = true
    setStreamLoading(true)
    setStreamError(false)

    const start = async () => {
      try {
        const pc = new RTCPeerConnection({ iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] })
        pcRef.current = pc
        pc.ontrack = (ev) => {
          if (videoRef.current && ev.streams && ev.streams[0]) {
            videoRef.current.srcObject = ev.streams[0]
          }
        }
        pc.addTransceiver('video', { direction: 'recvonly' })
        pc.addTransceiver('audio', { direction: 'recvonly' })
        const offer = await pc.createOffer()
        await pc.setLocalDescription(offer)

        const resp = await apiPost.webrtcOffer({
          camera_id: camera.id,
          sdp: offer.sdp,
          type: 'offer',
        })
        if (!active) { try { pc.close() } catch {} return }
        const answer = resp.sdp || resp.answer
        await pc.setRemoteDescription(new RTCSessionDescription({ type: 'answer', sdp: answer }))
        setStreamLoading(false)
      } catch (e) {
        if (active) { setStreamError(true); setStreamLoading(false) }
      }
    }
    start()

    return () => {
      active = false
      if (pcRef.current) { try { pcRef.current.close() } catch {} pcRef.current = null }
    }
  }, [showStream, camera.id])

  return (
    <div
      className={`bg-bg-card border border-border rounded-xl overflow-hidden card-hover ${onClick ? 'cursor-pointer' : ''}`}
      onClick={onClick}
    >
      {/* Video / stream area */}
      <div className="relative aspect-video bg-black">
        {showStream ? (
          <>
            <video ref={videoRef} autoPlay playsInline muted className="w-full h-full object-cover" />
            <ZoneOverlay zones={zones} />
            {streamLoading && (
              <div className="absolute inset-0 flex items-center justify-center text-text-muted text-sm bg-black/60">
                <span className="flex items-center gap-2"><Video size={18} className="animate-pulse" /> {t('cameras.loadingStream')}</span>
              </div>
            )}
            {streamError && (
              <div className="absolute inset-0 flex items-center justify-center text-critical text-sm bg-black/70">
                <span className="flex items-center gap-2"><VideoOff size={18} /> {t('cameras.streamFailed')}</span>
              </div>
            )}
          </>
        ) : (
          <div className="absolute inset-0 flex items-center justify-center text-text-muted">
            <Camera size={36} />
          </div>
        )}
        <div className="absolute top-2 end-2">
          <StatusPill status={camera.status} />
        </div>
      </div>

      {/* Info */}
      <div className="p-4">
        <div className="flex items-center justify-between gap-2 mb-2">
          <h3 className="font-semibold text-text-primary truncate">{camera.name || `Camera ${camera.id}`}</h3>
        </div>
        {zones.length > 0 && (
          <div className="text-xs text-text-secondary mb-2">
            <span className="text-text-muted">{t('cameras.zones')}:</span>{' '}
            {zones.map((z) => z.type || z.name).filter(Boolean).join(', ')}
          </div>
        )}
        {recent.length > 0 && (
          <div className="mt-2">
            <div className="text-xs text-text-muted mb-1">{t('cameras.lastEvents')}</div>
            <div className="space-y-1">
              {recent.map((e, i) => {
                const sev = severityOf(e.event_type)
                return (
                  <div key={e.id || i} className="flex items-center justify-between text-xs">
                    <span className="text-text-secondary truncate">{t(`eventTypes.${e.event_type}`, e.event_type)}</span>
                    <span className="text-text-muted flex-shrink-0 ms-2">{fmtTime(e.timestamp || e.created_at)}</span>
                  </div>
                )
              })}
            </div>
          </div>
        )}
        {onClick && (
          <button className="mt-3 w-full text-xs font-medium text-accent hover:text-accent-hover border border-border rounded-lg py-1.5">
            {t('cameras.viewDetails')}
          </button>
        )}
      </div>
    </div>
  )
}