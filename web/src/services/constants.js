export const SEVERITY = {
  critical: 'critical',
  warning: 'warning',
  info: 'info',
}

export const EVENT_SEVERITY = {
  slip_fall: 'critical',
  cash_drawer: 'critical',
  after_hours: 'critical',
  buddy_punch: 'critical',
  blocked_exit: 'critical',
  uniform_violation: 'warning',
  hygiene_violation: 'warning',
  phone_usage: 'warning',
  cleanliness_alert: 'warning',
  checkout_bottleneck: 'warning',
  stockroom_anomaly: 'warning',
  loitering: 'warning',
  camera_degraded: 'warning',
  loyalty_recognized: 'info',
  occupancy_update: 'info',
  activity_update: 'info',
}

export const CRITICAL_TYPES = Object.keys(EVENT_SEVERITY).filter((k) => EVENT_SEVERITY[k] === 'critical')
export const WARNING_TYPES = Object.keys(EVENT_SEVERITY).filter((k) => EVENT_SEVERITY[k] === 'warning')
export const INFO_TYPES = Object.keys(EVENT_SEVERITY).filter((k) => EVENT_SEVERITY[k] === 'info')

export const ALL_EVENT_TYPES = Object.keys(EVENT_SEVERITY)

export const ZONE_TYPES = [
  'checkout', 'aisles', 'stockroom', 'back_office',
  'entrance', 'restroom', 'restricted', 'privacy_mask',
]

export function sevColor(sev) {
  return sev === 'critical' ? '#ef4444' : sev === 'warning' ? '#f59e0b' : '#3b82f6'
}

export function severityOf(eventType) {
  return EVENT_SEVERITY[eventType] || 'info'
}

export function fmtTime(ts) {
  if (!ts) return '—'
  const d = new Date(ts)
  if (isNaN(d.getTime())) return String(ts)
  return d.toLocaleString()
}

export function fmtDate(ts) {
  if (!ts) return '—'
  const d = new Date(ts)
  if (isNaN(d.getTime())) return String(ts)
  return d.toLocaleDateString()
}