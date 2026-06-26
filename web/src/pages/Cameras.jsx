import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useLang } from '../contexts/LanguageContext'
import { apiGet } from '../services/api'
import { severityOf, fmtTime } from '../services/constants'
import CameraGrid from '../components/CameraGrid'
import SeverityBadge from '../components/SeverityBadge'
import { ArrowLeft, Video } from 'lucide-react'

export default function Cameras() {
  const { id } = useParams()
  const navigate = useNavigate()
  const { t, isRTL } = useLang()
  const [cameras, setCameras] = useState([])
  const [selected, setSelected] = useState(null)
  const [events, setEvents] = useState([])
  const [loading, setLoading] = useState(true)

  // Load all cameras
  useEffect(() => {
    let active = true
    setLoading(true)
    apiGet.cameras()
      .then((d) => {
        if (active) {
          const arr = Array.isArray(d) ? d : d.items || []
          setCameras(arr)
        }
      })
      .catch(() => {})
      .finally(() => { if (active) setLoading(false) })
    return () => { active = false }
  }, [])

  // If a camera id is in the URL, load its detail
  useEffect(() => {
    if (!id) { setSelected(null); return }
    let active = true
    apiGet.camera(id).then((d) => { if (active) setSelected(d) }).catch(() => {})
    apiGet.detections({ camera_id: id, limit: 100 })
      .then((d) => {
        if (active) {
          const arr = Array.isArray(d) ? d : d.items || d.detections || []
          setEvents(arr)
        }
      })
      .catch(() => {})
    return () => { active = false }
  }, [id])

  if (loading) {
    return <div className="flex items-center justify-center h-64 text-text-muted">{t('common.loading')}</div>
  }

  // Detail view
  if (id && selected) {
    return (
      <div className="space-y-5">
        <button
          onClick={() => navigate('/cameras')}
          className="flex items-center gap-2 text-sm text-text-secondary hover:text-text-primary"
        >
          <ArrowLeft size={18} className={isRTL ? 'rtl-flip' : ''} />
          {t('cameras.backToList')}
        </button>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
          {/* Live stream */}
          <div className="lg:col-span-2 bg-bg-card border border-border rounded-xl overflow-hidden">
            <div className="flex items-center gap-2 px-4 py-3 border-b border-border">
              <Video size={16} className="text-accent" />
              <h2 className="font-semibold text-text-primary">{selected.name || `Camera ${selected.id}`}</h2>
              <span className="text-xs text-text-muted ms-auto">{t('cameras.liveStream')}</span>
            </div>
            <div className="relative aspect-video bg-black">
              <CameraLive cameraId={selected.id} />
            </div>
          </div>

          {/* Camera info */}
          <div className="bg-bg-card border border-border rounded-xl p-5 space-y-3">
            <h3 className="font-semibold text-text-primary">{t('cameras.title')}</h3>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between"><span className="text-text-secondary">{t('cameras.status')}</span><span>{selected.status}</span></div>
              <div className="flex justify-between"><span className="text-text-secondary">ID</span><span className="text-text-muted">{selected.id}</span></div>
              {selected.location && <div className="flex justify-between"><span className="text-text-secondary">{t('cameras.zone')}</span><span>{selected.location}</span></div>}
            </div>
          </div>
        </div>

        {/* Full event history */}
        <div className="bg-bg-card border border-border rounded-xl">
          <div className="px-4 py-3 border-b border-border">
            <h3 className="font-semibold text-text-primary">{t('cameras.eventHistory')}</h3>
          </div>
          <div className="divide-y divide-border">
            {events.length === 0 && <div className="p-4 text-center text-text-muted text-sm">{t('events.noEvents')}</div>}
            {events.map((e, i) => (
              <div key={e.id || i} className="p-3 flex items-center justify-between gap-3 card-hover">
                <div className="min-w-0">
                  <div className="text-sm font-medium text-text-primary">{t(`eventTypes.${e.event_type}`, e.event_type)}</div>
                  <div className="text-xs text-text-muted">{fmtTime(e.timestamp || e.created_at)}</div>
                </div>
                <SeverityBadge severity={severityOf(e.event_type)} />
              </div>
            ))}
          </div>
        </div>
      </div>
    )
  }

  // Grid view
  return (
    <div className="space-y-5">
      <h1 className="text-xl font-bold text-text-primary">{t('cameras.title')}</h1>
      <CameraGrid cameras={cameras} onCameraClick={(c) => navigate(`/cameras/${c.id}`)} showStream />
    </div>
  )
}

// Inline live stream component for detail view
import { useRef } from 'react'
import { apiPost } from '../services/api'

function CameraLive({ cameraId }) {
  const videoRef = useRef(null)
  const pcRef = useRef(null)
  const [state, setState] = useState('loading') // loading | ok | error

  useEffect(() => {
    let active = true
    const start = async () => {
      try {
        const pc = new RTCPeerConnection({ iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] })
        pcRef.current = pc
        pc.ontrack = (ev) => { if (videoRef.current && ev.streams[0]) videoRef.current.srcObject = ev.streams[0] }
        pc.addTransceiver('video', { direction: 'recvonly' })
        pc.addTransceiver('audio', { direction: 'recvonly' })
        const offer = await pc.createOffer()
        await pc.setLocalDescription(offer)
        const resp = await apiPost.webrtcOffer({ camera_id: cameraId, sdp: offer.sdp, type: 'offer' })
        if (!active) { try { pc.close() } catch {} return }
        await pc.setRemoteDescription(new RTCSessionDescription({ type: 'answer', sdp: resp.sdp || resp.answer }))
        if (active) setState('ok')
      } catch {
        if (active) setState('error')
      }
    }
    start()
    return () => { active = false; if (pcRef.current) { try { pcRef.current.close() } catch {} pcRef.current = null } }
  }, [cameraId])

  return (
    <>
      <video ref={videoRef} autoPlay playsInline muted className="w-full h-full object-cover" />
      {state === 'loading' && (
        <div className="absolute inset-0 flex items-center justify-center text-text-muted text-sm bg-black/60">
          <Video size={20} className="animate-pulse me-2" /> Loading stream...
        </div>
      )}
      {state === 'error' && (
        <div className="absolute inset-0 flex items-center justify-center text-critical text-sm bg-black/70">
          Stream connection failed
        </div>
      )}
    </>
  )
}