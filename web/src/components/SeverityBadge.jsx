import { sevColor } from '../services/constants'
import { useLang } from '../contexts/LanguageContext'
import { AlertTriangle, AlertCircle, Info } from 'lucide-react'

export default function SeverityBadge({ severity }) {
  const { t } = useLang()
  const color = sevColor(severity)
  const Icon = severity === 'critical' ? AlertTriangle : severity === 'warning' ? AlertCircle : Info
  const label = t(`severity.${severity}`, severity)
  return (
    <span
      className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold border"
      style={{ color, borderColor: color + '55', backgroundColor: color + '15' }}
    >
      <Icon size={12} />
      {label}
    </span>
  )
}