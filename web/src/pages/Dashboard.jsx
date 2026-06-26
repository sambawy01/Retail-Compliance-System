import { useEffect, useState, useMemo } from 'react'
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid, Cell } from 'recharts'
import { useLang } from '../contexts/LanguageContext'
import { apiGet } from '../services/api'
import { severityOf, sevColor, CRITICAL_TYPES, WARNING_TYPES } from '../services/constants'
import ComplianceGauge from '../components/ComplianceGauge'
import EventFeed from '../components/EventFeed'
import { ShieldCheck, AlertTriangle, AlertCircle, Info, Camera, CameraOff, Activity, TrendingUp } from 'lucide-react'

function StatCard({ icon: Icon, label, value, color }) {
  return (
    <div className="bg-bg-card border border-border rounded-xl p-4 flex items-center gap-4">
      <div className="w-12 h-12 rounded-lg flex items-center justify-center flex-shrink-0" style={{ backgroundColor: color + '20', color }}>
        <Icon size={24} />
      </div>
      <div className="min-w-0">
        <div className="text-2xl font-bold text-text-primary leading-tight">{value}</div>
        <div className="text-xs text-text-secondary truncate">{label}</div>
      </div>
    </div>
  )
}

export default function Dashboard() {
  const { t } = useLang()
  const [cameras, setCameras] = useState([])
  const [detections, setDetections] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let active = true
    const load = async () => {
      setLoading(true)
      try {
        const cams = await apiGet.cameras()
        if (active) {
          const arr = Array.isArray(cams) ? cams : cams.items || []
          setCameras(arr)
        }
      } catch { /* ignore */ }
      try {
        const det = await apiGet.detections({ limit: 200 })
        if (active) {
          const arr = Array.isArray(det) ? det : det.items || det.detections || []
          setDetections(arr)
        }
      } catch { /* ignore */ }
      if (active) setLoading(false)
    }
    load()
    const id = setInterval(load, 30000)
    return () => { active = false; clearInterval(id) }
  }, [])

  const stats = useMemo(() => {
    const today = new Date()
    today.setHours(0, 0, 0, 0)
    const todayEvents = detections.filter((d) => {
      const ts = new Date(d.timestamp || d.created_at)
      return ts >= today
    })
    const critical = todayEvents.filter((d) => severityOf(d.event_type) === 'critical').length
    const warning = todayEvents.filter((d) => severityOf(d.event_type) === 'warning').length
    const info = todayEvents.filter((d) => severityOf(d.event_type) === 'info').length
    return { critical, warning, info, total: todayEvents.length }
  }, [detections])

  const complianceScore = useMemo(() => {
    const total = stats.critical + stats.warning + stats.info
    if (total === 0) return 100
    // Weighted: critical=0 (worst), warning=0.5, info=1
    const weighted = stats.critical * 0 + stats.warning * 0.5 + stats.info * 1
    return (weighted / total) * 100
  }, [stats])

  const cameraStatus = useMemo(() => {
    const online = cameras.filter((c) => c.status === 'online').length
    const offline = cameras.filter((c) => c.status === 'offline').length
    const degraded = cameras.filter((c) => c.status === 'degraded').length
    return { online, offline, degraded, total: cameras.length }
  }, [cameras])

  const topViolations = useMemo(() => {
    const weekAgo = new Date()
    weekAgo.setDate(weekAgo.getDate() - 7)
    const weekEvents = detections.filter((d) => {
      const ts = new Date(d.timestamp || d.created_at)
      return ts >= weekAgo
    })
    const counts = {}
    for (const d of weekEvents) {
      counts[d.event_type] = (counts[d.event_type] || 0) + 1
    }
    return Object.entries(counts)
      .sort((a, b) => b[1] - a[1])
      .slice(0, 8)
      .map(([type, count]) => ({ type: t(`eventTypes.${type}`, type), count, sev: severityOf(type), raw: type }))
  }, [detections, t])

  const camStatusList = useMemo(() => {
    return cameras.map((c) => ({ name: c.name || `#${c.id}`, status: c.status || 'offline' }))
  }, [cameras])

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64 text-text-muted">
        <Activity size={20} className="animate-pulse me-2" /> {t('common.loading')}
      </div>
    )
  }

  return (
    <div className="space-y-5">
      <h1 className="text-xl font-bold text-text-primary">{t('dashboard.title')}</h1>

      {/* Top row: gauge + summary */}
      <div className="grid grid-cols-1 lg:grid-cols-4 gap-4">
        <ComplianceGauge score={complianceScore} />
        <div className="lg:col-span-3 grid grid-cols-2 md:grid-cols-4 gap-4">
          <StatCard icon={AlertTriangle} label={t('dashboard.critical')} value={stats.critical} color="#ef4444" />
          <StatCard icon={AlertCircle} label={t('dashboard.warning')} value={stats.warning} color="#f59e0b" />
          <StatCard icon={Info} label={t('dashboard.info')} value={stats.info} color="#3b82f6" />
          <StatCard icon={Camera} label={t('dashboard.camerasOnline')} value={`${cameraStatus.online}/${cameraStatus.total}`} color="#22c55e" />
        </div>
      </div>

      {/* Camera status row */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <div className="bg-bg-card border border-border rounded-xl p-5">
          <h3 className="text-sm font-semibold text-text-secondary mb-4">{t('dashboard.cameraStatus')}</h3>
          <div className="grid grid-cols-3 gap-3 mb-4">
            <div className="text-center">
              <div className="text-2xl font-bold text-success">{cameraStatus.online}</div>
              <div className="text-xs text-text-secondary">{t('dashboard.online')}</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-degraded">{cameraStatus.degraded}</div>
              <div className="text-xs text-text-secondary">{t('dashboard.degraded')}</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-offline">{cameraStatus.offline}</div>
              <div className="text-xs text-text-secondary">{t('dashboard.offline')}</div>
            </div>
          </div>
          <div className="space-y-1.5 max-h-48 overflow-y-auto">
            {camStatusList.map((c, i) => (
              <div key={i} className="flex items-center justify-between text-xs">
                <span className="text-text-secondary truncate">{c.name}</span>
                <span className="flex items-center gap-1.5">
                  <span className="w-1.5 h-1.5 rounded-full" style={{
                    backgroundColor: c.status === 'online' ? '#22c55e' : c.status === 'degraded' ? '#f59e0b' : '#6b7280'
                  }} />
                  <span className="text-text-muted">{t(`common.${c.status}`, c.status)}</span>
                </span>
              </div>
            ))}
          </div>
        </div>

        {/* Top violations */}
        <div className="lg:col-span-2 bg-bg-card border border-border rounded-xl p-5">
          <div className="flex items-center gap-2 mb-4">
            <TrendingUp size={16} className="text-accent" />
            <h3 className="text-sm font-semibold text-text-secondary">{t('dashboard.topViolations')}</h3>
          </div>
          {topViolations.length === 0 ? (
            <div className="text-sm text-text-muted text-center py-12">{t('dashboard.noData')}</div>
          ) : (
            <ResponsiveContainer width="100%" height={260}>
              <BarChart data={topViolations} layout="vertical" margin={{ left: 0, right: 16 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#2a2e3a" />
                <XAxis type="number" stroke="#6b7280" fontSize={11} />
                <YAxis dataKey="type" type="category" stroke="#9ca3af" fontSize={11} width={120} tick={{ fill: '#9ca3af' }} />
                <Tooltip cursor={{ fill: '#1e222c' }} contentStyle={{ background: '#171a21', border: '1px solid #2a2e3a', borderRadius: 8, fontSize: 12 }} />
                <Bar dataKey="count" radius={[0, 4, 4, 0]}>
                  {topViolations.map((v, i) => (
                    <Cell key={i} fill={sevColor(v.sev)} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          )}
        </div>
      </div>

      {/* Recent events feed */}
      <div className="h-96">
        <EventFeed limit={20} />
      </div>
    </div>
  )
}