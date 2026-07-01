// useWebRTCStream — custom hook for WebRTC camera streaming.
// Extracted from CameraCard.jsx and Cameras.jsx to eliminate duplication.
import { useEffect, useRef, useState } from 'react'
import { apiPost } from './api'

export function useWebRTCStream(cameraId, enabled = true) {
  const videoRef = useRef(null)
  const pcRef = useRef(null)
  const [loading, setLoading] = useState(enabled)
  const [error, setError] = useState(false)

  useEffect(() => {
    if (!enabled || !cameraId) return
    let active = true
    setLoading(true)
    setError(false)

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

        const resp = await apiPost.webrtcOffer({ camera_id: cameraId, sdp: offer.sdp, type: 'offer' })
        if (!active) { try { pc.close() } catch {} return }
        const answer = resp.sdp || resp.answer
        await pc.setRemoteDescription(new RTCSessionDescription({ type: 'answer', sdp: answer }))
        if (active) setLoading(false)
      } catch (e) {
        if (active) { setError(true); setLoading(false) }
      }
    }
    start()

    return () => {
      active = false
      if (pcRef.current) { try { pcRef.current.close() } catch {} pcRef.current = null }
    }
  }, [cameraId, enabled])

  return { videoRef, loading, error }
}