import { useEffect, useState, useCallback } from 'react'
import { useLang } from '../contexts/LanguageContext'
import { apiGet } from '../services/api'
import { reportError } from '../services/errors'
import { fmtDate } from '../services/constants'
import { Users, Gauge, Clock, Calendar, FileText, Phone, Briefcase, TrendingUp, AlertTriangle, CheckCircle, XCircle, Loader2, ChevronDown, ChevronUp, Award, UserX } from 'lucide-react'

export default function Staff() {
  const { t } = useLang()
  const [staff, setStaff] = useState([])
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState(null)
  const [report, setReport] = useState(null)
  const [reportLoading, setReportLoading] = useState(false)
  const [expanded, setExpanded] = useState(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const d = await apiGet.staff()
      const arr = Array.isArray(d) ? d : d.staff || []
      setStaff(arr)
    } catch (e) { reportError(e, 'loading staff') }
    setLoading(false)
  }, [])

  useEffect(() => { load() }, [load])

  const loadReport = async (personId) => {
    setReportLoading(true)
    setReport(null)
    try {
      const d = await apiGet.staffReport(personId)
      setReport(d.report || '')
    } catch (e) { reportError(e, 'loading staff') }
    setReportLoading(false)
  }

  const openDetail = (person) => {
    setSelected(person)
    loadReport(person.person_id)
  }

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold text-text-primary">{t('staff.title')}</h1>
        <span className="text-sm text-text-muted">{staff.length} {t('staff.members')}</span>
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64 text-text-muted">
          <Loader2 size={20} className="animate-spin me-2" /> {t('common.loading')}
        </div>
      ) : staff.length === 0 ? (
        <div className="text-center py-12 text-text-muted bg-bg-card border border-border rounded-xl">
          <Users size={36} className="mx-auto mb-3 opacity-40" />
          {t('staff.noStaff')}
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {staff.map((p) => (
            <StaffCard key={p.person_id} person={p} t={t} onClick={() => openDetail(p)} />
          ))}
        </div>
      )}

      {/* Detail modal with AI report */}
      {selected && (
        <StaffDetailModal
          person={selected}
          report={report}
          reportLoading={reportLoading}
          onClose={() => { setSelected(null); setReport(null) }}
          t={t}
        />
      )}
    </div>
  )
}

function ScoreRing({ score }) {
  const pct = Math.max(0, Math.min(100, Math.round(score || 0)))
  const color = pct >= 80 ? '#22c55e' : pct >= 60 ? '#f59e0b' : '#ef4444'
  const radius = 28
  const circ = 2 * Math.PI * radius
  const offset = circ - (pct / 100) * circ
  return (
    <div className="relative w-16 h-16 flex-shrink-0">
      <svg className="w-full h-full -rotate-90" viewBox="0 0 70 70">
        <circle cx="35" cy="35" r={radius} fill="none" stroke="#2a2e3a" strokeWidth="6" />
        <circle cx="35" cy="35" r={radius} fill="none" stroke={color} strokeWidth="6"
          strokeLinecap="round" strokeDasharray={circ} strokeDashoffset={offset}
          style={{ transition: 'stroke-dashoffset 0.6s ease' }} />
      </svg>
      <div className="absolute inset-0 flex items-center justify-center">
        <span className="text-sm font-bold" style={{ color }}>{pct}</span>
      </div>
    </div>
  )
}

function StaffCard({ person, t, onClick }) {
  const score = person.performance_score || 100
  const photo = person.photo_url
  const statusColor = person.status === 'active' ? '#22c55e' : person.status === 'on_leave' ? '#f59e0b' : '#6b7280'

  return (
    <div className="bg-bg-card border border-border rounded-xl p-4 card-hover cursor-pointer" onClick={onClick}>
      <div className="flex items-start gap-3">
        {photo ? (
          <img src={photo} alt={person.display_name} className="w-14 h-14 rounded-full object-cover border border-border" />
        ) : (
          <div className="w-14 h-14 rounded-full bg-accent/15 text-accent flex items-center justify-center">
            <Users size={22} />
          </div>
        )}
        <div className="min-w-0 flex-1">
          <div className="text-sm font-bold text-text-primary truncate">{person.display_name}</div>
          {person.job_role && <div className="text-xs text-text-secondary flex items-center gap-1"><Briefcase size={10} /> {person.job_role}</div>}
          {person.department && <div className="text-xs text-text-muted">{person.department}</div>}
          <div className="flex items-center gap-1.5 mt-1">
            <span className="w-1.5 h-1.5 rounded-full" style={{ backgroundColor: statusColor }} />
            <span className="text-xs text-text-muted">{t(`staff.status.${person.status || 'active'}`, person.status || 'active')}</span>
          </div>
        </div>
        <ScoreRing score={score} />
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-4 gap-2 mt-3 pt-3 border-t border-border">
        <Stat icon={AlertTriangle} value={person.critical_events || 0} label={t('staff.critical')} color="#ef4444" />
        <Stat icon={AlertTriangle} value={person.warning_events || 0} label={t('staff.warnings')} color="#f59e0b" />
        <Stat icon={Clock} value={person.attendance?.late_days || 0} label={t('staff.late')} color="#f59e0b" />
        <Stat icon={Calendar} value={person.holidays?.length || 0} label={t('staff.leave')} color="#3b82f6" />
      </div>
    </div>
  )
}

function Stat({ icon: Icon, value, label, color }) {
  return (
    <div className="text-center">
      <Icon size={14} className="mx-auto mb-0.5" style={{ color }} />
      <div className="text-sm font-bold text-text-primary">{value}</div>
      <div className="text-xs text-text-muted truncate">{label}</div>
    </div>
  )
}

function StaffDetailModal({ person, report, reportLoading, onClose, t }) {
  const score = person.performance_score || 100
  const scoreColor = score >= 80 ? '#22c55e' : score >= 60 ? '#f59e0b' : '#ef4444'
  const scoreLabel = score >= 80 ? t('staff.excellent') : score >= 60 ? t('staff.good') : score >= 40 ? t('staff.needsImprovement') : t('staff.poor')
  const att = person.attendance || {}
  const photo = person.photo_url

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60" onClick={onClose}>
      <div className="bg-bg-card border border-border rounded-xl w-full max-w-2xl max-h-[90vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-3 border-b border-border sticky top-0 bg-bg-card z-10">
          <h3 className="font-semibold text-text-primary">{t('staff.profile')}</h3>
          <button onClick={onClose} className="text-text-muted hover:text-text-primary"><ChevronUp size={18} /></button>
        </div>

        <div className="p-5 space-y-5">
          {/* Profile header */}
          <div className="flex items-start gap-4">
            {photo ? (
              <img src={photo} alt={person.display_name} className="w-20 h-20 rounded-xl object-cover border border-border" />
            ) : (
              <div className="w-20 h-20 rounded-xl bg-accent/15 text-accent flex items-center justify-center">
                <Users size={32} />
              </div>
            )}
            <div className="flex-1 min-w-0">
              <h2 className="text-lg font-bold text-text-primary">{person.display_name}</h2>
              {person.job_role && <p className="text-sm text-text-secondary flex items-center gap-1"><Briefcase size={12} /> {person.job_role}</p>}
              {person.department && <p className="text-xs text-text-muted">{person.department}</p>}
              {person.phone && <p className="text-xs text-text-muted flex items-center gap-1 mt-0.5"><Phone size={10} /> {person.phone}</p>}
              {person.shift_start && person.shift_end && (
                <p className="text-xs text-text-muted flex items-center gap-1 mt-0.5"><Clock size={10} /> {person.shift_start} - {person.shift_end}</p>
              )}
              {person.hire_date && <p className="text-xs text-text-muted mt-0.5">{t('staff.hired')}: {fmtDate(person.hire_date)}</p>}
            </div>
            <div className="text-center">
              <ScoreRing score={score} />
              <div className="text-xs font-semibold mt-1" style={{ color: scoreColor }}>{scoreLabel}</div>
            </div>
          </div>

          {/* Performance score bar */}
          <div className="bg-bg rounded-lg p-3 border border-border">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-semibold text-text-secondary flex items-center gap-1.5"><Gauge size={12} /> {t('staff.performanceScore')}</span>
              <span className="text-sm font-bold" style={{ color: scoreColor }}>{Math.round(score)}%</span>
            </div>
            <div className="h-2.5 bg-border rounded-full overflow-hidden">
              <div className="h-full rounded-full transition-all" style={{ width: `${score}%`, backgroundColor: scoreColor }} />
            </div>
            <div className="grid grid-cols-3 gap-2 mt-3 text-center">
              <div><div className="text-lg font-bold text-critical">{person.critical_events || 0}</div><div className="text-xs text-text-muted">{t('staff.criticalEvents')}</div></div>
              <div><div className="text-lg font-bold text-warning">{person.warning_events || 0}</div><div className="text-xs text-text-muted">{t('staff.warningEvents')}</div></div>
              <div><div className="text-lg font-bold text-info">{person.info_events || 0}</div><div className="text-xs text-text-muted">{t('staff.infoEvents')}</div></div>
            </div>
          </div>

          {/* Event breakdown */}
          {person.event_breakdown && person.event_breakdown.length > 0 && (
            <div className="bg-bg rounded-lg p-3 border border-border">
              <div className="text-xs font-semibold text-text-secondary mb-2 flex items-center gap-1.5"><TrendingUp size={12} /> {t('staff.topViolations')}</div>
              <div className="space-y-1.5">
                {person.event_breakdown.map((e, i) => (
                  <div key={i} className="flex items-center justify-between text-xs">
                    <span className="text-text-secondary truncate">{t(`eventTypes.${e.event_type}`, e.event_type)}</span>
                    <span className="flex items-center gap-1.5 flex-shrink-0">
                      <span className="w-1.5 h-1.5 rounded-full" style={{ backgroundColor: sevColor(e.severity) }} />
                      <span className="text-text-muted">{e.count}x</span>
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Attendance summary */}
          <div className="bg-bg rounded-lg p-3 border border-border">
            <div className="text-xs font-semibold text-text-secondary mb-3 flex items-center gap-1.5"><Clock size={12} /> {t('staff.attendance')} ({t('staff.last30Days')})</div>
            <div className="grid grid-cols-5 gap-2">
              <AttStat value={att.present_days || 0} label={t('staff.present')} color="#22c55e" />
              <AttStat value={att.late_days || 0} label={t('staff.lateDays')} color="#f59e0b" />
              <AttStat value={att.absent_days || 0} label={t('staff.absent')} color="#ef4444" />
              <AttStat value={att.half_days || 0} label={t('staff.halfDay')} color="#3b82f6" />
              <AttStat value={att.remote_days || 0} label={t('staff.remote')} color="#8b5cf6" />
            </div>
            <div className="grid grid-cols-3 gap-2 mt-3 pt-3 border-t border-border">
              <div className="text-center"><div className="text-sm font-bold text-text-primary">{Math.round(att.attendance_rate || 100)}%</div><div className="text-xs text-text-muted">{t('staff.attendanceRate')}</div></div>
              <div className="text-center"><div className="text-sm font-bold text-text-primary">{att.late_minutes || 0}m</div><div className="text-xs text-text-muted">{t('staff.totalLate')}</div></div>
              <div className="text-center"><div className="text-sm font-bold text-text-primary">{att.overtime_mins || 0}m</div><div className="text-xs text-text-muted">{t('staff.overtime')}</div></div>
            </div>
          </div>

          {/* Holidays / Leave */}
          {person.holidays && person.holidays.length > 0 && (
            <div className="bg-bg rounded-lg p-3 border border-border">
              <div className="text-xs font-semibold text-text-secondary mb-2 flex items-center gap-1.5"><Calendar size={12} /> {t('staff.leaveHolidays')}</div>
              <div className="space-y-2">
                {person.holidays.map((h, i) => {
                  const statusIcon = h.status === 'approved' ? <CheckCircle size={12} className="text-success" /> : h.status === 'rejected' ? <XCircle size={12} className="text-critical" /> : <Clock size={12} className="text-warning" />
                  return (
                    <div key={i} className="flex items-center justify-between text-xs">
                      <div className="flex items-center gap-2">
                        {statusIcon}
                        <span className="text-text-secondary">{fmtDate(h.start_date)} - {fmtDate(h.end_date)}</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="text-text-muted">{t(`staff.leaveType.${h.type}`, h.type)}</span>
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          )}

          {/* AI Report */}
          <div className="bg-bg rounded-lg p-4 border border-accent/30">
            <div className="flex items-center gap-2 mb-3">
              <FileText size={16} className="text-accent" />
              <span className="text-sm font-semibold text-text-primary">{t('staff.aiReport')}</span>
              {reportLoading && <Loader2 size={14} className="animate-spin text-text-muted ms-1" />}
            </div>
            {reportLoading ? (
              <div className="text-sm text-text-muted text-center py-4">{t('common.loading')}</div>
            ) : report ? (
              <pre className="text-xs text-text-secondary whitespace-pre-wrap font-mono leading-relaxed">{report}</pre>
            ) : (
              <div className="text-sm text-text-muted text-center py-4">{t('staff.noReport')}</div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

function AttStat({ value, label, color }) {
  return (
    <div className="text-center">
      <div className="text-lg font-bold" style={{ color }}>{value}</div>
      <div className="text-xs text-text-muted truncate">{label}</div>
    </div>
  )
}