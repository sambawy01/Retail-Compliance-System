import { useLang } from '../contexts/LanguageContext'
import { severityOf, sevColor, fmtTime, fmtDate } from '../services/constants'
import { AlertTriangle, AlertCircle, Info, Clock, MapPin, Camera, User, Gauge, ShieldCheck, Video, Zap } from 'lucide-react'

export default function EventDetailCard({ event, camera, zone, clip }) {
  const { t, isRTL } = useLang()
  const sev = severityOf(event.event_type)
  const color = sevColor(sev)
  const Icon = sev === 'critical' ? AlertTriangle : sev === 'warning' ? AlertCircle : Info

  // Extract data from event and payload
  const payload = event.payload || {}
  const confidence = event.confidence != null ? Math.round(event.confidence * 100) : null
  const detectedAt = event.detected_at || event.timestamp || event.created_at
  const createdAt = event.created_at
  const cameraName = camera?.name || event.camera_name || payload.camera_name || '—'
  const zoneName = zone?.name || payload.zone_name || '—'
  const locationName = payload.location_name || '—'

  // Employee/face match data from payload
  const employeeName = payload.matched_person || payload.employee_name || payload.person_name
  const employeeId = payload.person_id || payload.employee_id
  const employeeKind = payload.person_kind || payload.employee_kind
  const matchSimilarity = payload.similarity != null ? Math.round(payload.similarity * 100) : null

  // Performance/compliance score from payload
  const performanceScore = payload.performance_score != null ? Math.round(payload.performance_score * 100) : null
  const complianceImpact = payload.compliance_impact

  // Duration if available
  const durationSecs = payload.duration_secs || payload.duration

  // Recommended action based on severity
  const actions = {
    critical: t('eventDetail.actionCritical', 'Immediate review required. Notify store manager and security team.'),
    warning: t('eventDetail.actionWarning', 'Review within 1 hour. Coach employee if applicable.'),
    info: t('eventDetail.actionInfo', 'Informational. No immediate action required.'),
  }
  const recommendedAction = actions[sev] || actions.info

  // Incident description
  const description = t(`eventDescriptions.${event.event_type}`, '')

  // Zone type label
  const zoneType = zone?.type || zone?.kind || payload.zone_type

  return (
    <div className="space-y-4">
      {/* Header: severity + event type */}
      <div className="flex items-start gap-3 pb-4 border-b border-border">
        <div className="w-12 h-12 rounded-lg flex items-center justify-center flex-shrink-0"
          style={{ backgroundColor: color + '20', color }}>
          <Icon size={24} />
        </div>
        <div className="min-w-0 flex-1">
          <h3 className="text-lg font-bold text-text-primary">
            {t(`eventTypes.${event.event_type}`, event.event_type)}
          </h3>
          <div className="flex items-center gap-2 mt-1">
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold border"
              style={{ color, borderColor: color + '55', backgroundColor: color + '15' }}>
              <Icon size={10} /> {t(`severity.${sev}`, sev)}
            </span>
            {confidence != null && (
              <span className="text-xs text-text-muted">
                {t('eventDetail.confidence')}: {confidence}%
              </span>
            )}
          </div>
        </div>
      </div>

      {/* Description */}
      {description && (
        <div className="bg-bg rounded-lg p-3 border border-border">
          <div className="text-xs font-semibold text-text-secondary mb-1">{t('eventDetail.description')}</div>
          <p className="text-sm text-text-primary leading-relaxed">{description}</p>
        </div>
      )}

      {/* Timing section */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <DetailRow icon={Clock} label={t('eventDetail.detectedAt')} value={fmtTime(detectedAt)} sub={fmtDate(detectedAt)} />
        <DetailRow icon={Clock} label={t('eventDetail.recordedAt')} value={fmtTime(createdAt)} sub={fmtDate(createdAt)} />
        {durationSecs != null && (
          <DetailRow icon={Zap} label={t('eventDetail.duration')} value={`${durationSecs}s`} />
        )}
      </div>

      {/* Location section */}
      <div className="bg-bg rounded-lg p-3 border border-border space-y-2">
        <div className="text-xs font-semibold text-text-secondary mb-2 flex items-center gap-1.5">
          <MapPin size={12} /> {t('eventDetail.location')}
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
          <DetailMini icon={Camera} label={t('eventDetail.camera')} value={cameraName} />
          <DetailMini icon={MapPin} label={t('eventDetail.zone')} value={zoneName} />
          {locationName !== '—' && (
            <DetailMini icon={MapPin} label={t('eventDetail.storeLocation')} value={locationName} />
          )}
          {zoneType && (
            <DetailMini icon={MapPin} label={t('eventDetail.zoneType')} value={t(`zoneTypes.${zoneType}`, zoneType)} />
          )}
        </div>
      </div>

      {/* Employee / Face Match section */}
      {(employeeName || matchSimilarity != null) && (
        <div className="bg-bg rounded-lg p-3 border border-border space-y-2">
          <div className="text-xs font-semibold text-text-secondary mb-2 flex items-center gap-1.5">
            <User size={12} /> {t('eventDetail.person')}
          </div>
          <div className="flex items-center gap-3">
            {payload.person_photo || payload.employee_photo ? (
              <img src={payload.person_photo || payload.employee_photo} alt={employeeName || ''}
                className="w-12 h-12 rounded-full object-cover border border-border" />
            ) : (
              <div className="w-12 h-12 rounded-full bg-accent/15 text-accent flex items-center justify-center">
                <User size={20} />
              </div>
            )}
            <div className="min-w-0 flex-1">
              <div className="text-sm font-medium text-text-primary truncate">{employeeName || t('eventDetail.unknownPerson')}</div>
              {employeeKind && (
                <div className="text-xs text-text-muted">{t(`faceid.kind${employeeKind.charAt(0).toUpperCase() + employeeKind.slice(1)}`, employeeKind)}</div>
              )}
              {matchSimilarity != null && (
                <div className="mt-1">
                  <div className="flex items-center gap-2">
                    <div className="flex-1 h-1.5 bg-border rounded-full overflow-hidden">
                      <div className="h-full rounded-full" style={{ width: `${matchSimilarity}%`, backgroundColor: '#3b82f6' }} />
                    </div>
                    <span className="text-xs text-text-muted">{matchSimilarity}% match</span>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Performance / Compliance score */}
      {performanceScore != null && (
        <div className="bg-bg rounded-lg p-3 border border-border">
          <div className="text-xs font-semibold text-text-secondary mb-2 flex items-center gap-1.5">
            <Gauge size={12} /> {t('eventDetail.performanceScore')}
          </div>
          <div className="flex items-center gap-3">
            <div className="flex-1">
              <div className="h-2 bg-border rounded-full overflow-hidden">
                <div className="h-full rounded-full transition-all"
                  style={{
                    width: `${performanceScore}%`,
                    backgroundColor: performanceScore >= 80 ? '#22c55e' : performanceScore >= 60 ? '#f59e0b' : '#ef4444'
                  }} />
              </div>
            </div>
            <span className="text-sm font-bold"
              style={{ color: performanceScore >= 80 ? '#22c55e' : performanceScore >= 60 ? '#f59e0b' : '#ef4444' }}>
              {performanceScore}%
            </span>
          </div>
          {complianceImpact && (
            <div className="text-xs text-text-muted mt-2">{t('eventDetail.complianceImpact')}: {complianceImpact}</div>
          )}
        </div>
      )}

      {/* Confidence bar */}
      {confidence != null && (
        <div className="bg-bg rounded-lg p-3 border border-border">
          <div className="text-xs font-semibold text-text-secondary mb-2 flex items-center gap-1.5">
            <Gauge size={12} /> {t('eventDetail.confidenceScore')}
          </div>
          <div className="flex items-center gap-3">
            <div className="flex-1">
              <div className="h-2 bg-border rounded-full overflow-hidden">
                <div className="h-full rounded-full" style={{ width: `${confidence}%`, backgroundColor: color }} />
              </div>
            </div>
            <span className="text-sm font-bold" style={{ color }}>{confidence}%</span>
          </div>
        </div>
      )}

      {/* Clip evidence */}
      {clip && (clip.thumbnail_url || clip.url) && (
        <div className="bg-bg rounded-lg p-3 border border-border">
          <div className="text-xs font-semibold text-text-secondary mb-2 flex items-center gap-1.5">
            <Video size={12} /> {t('eventDetail.evidenceClip')}
          </div>
          <div className="aspect-video bg-black rounded-lg overflow-hidden">
            {clip.thumbnail_url ? (
              <img src={clip.thumbnail_url} alt="clip" className="w-full h-full object-cover" />
            ) : clip.url ? (
              <video src={clip.url} controls className="w-full h-full object-cover" />
            ) : null}
          </div>
        </div>
      )}

      {/* Recommended action */}
      <div className="rounded-lg p-3 border" style={{ borderColor: color + '40', backgroundColor: color + '08' }}>
        <div className="flex items-start gap-2">
          <ShieldCheck size={16} className="flex-shrink-0 mt-0.5" style={{ color }} />
          <div>
            <div className="text-xs font-semibold mb-1" style={{ color }}>{t('eventDetail.recommendedAction')}</div>
            <p className="text-sm text-text-secondary leading-relaxed">{recommendedAction}</p>
          </div>
        </div>
      </div>

      {/* Raw payload (collapsible) */}
      {payload && Object.keys(payload).length > 0 && (
        <details className="bg-bg rounded-lg border border-border">
          <summary className="px-3 py-2 text-xs font-semibold text-text-secondary cursor-pointer hover:bg-bg-hover">
            {t('eventDetail.rawData')}
          </summary>
          <pre className="p-3 text-xs text-text-muted overflow-x-auto max-h-48 border-t border-border">
{JSON.stringify(event, null, 2)}
          </pre>
        </details>
      )}
    </div>
  )
}

function DetailRow({ icon: Icon, label, value, sub }) {
  return (
    <div className="flex items-start gap-2.5">
      <Icon size={16} className="text-text-muted flex-shrink-0 mt-0.5" />
      <div className="min-w-0">
        <div className="text-xs text-text-muted">{label}</div>
        <div className="text-sm text-text-primary truncate">{value}</div>
        {sub && sub !== '—' && <div className="text-xs text-text-muted">{sub}</div>}
      </div>
    </div>
  )
}

function DetailMini({ icon: Icon, label, value }) {
  return (
    <div className="flex items-center gap-2">
      <Icon size={14} className="text-text-muted flex-shrink-0" />
      <div className="min-w-0">
        <span className="text-xs text-text-muted">{label}: </span>
        <span className="text-xs text-text-secondary truncate">{value}</span>
      </div>
    </div>
  )
}