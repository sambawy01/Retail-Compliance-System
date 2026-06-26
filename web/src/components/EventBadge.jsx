import { useLang } from '../contexts/LanguageContext'
import { severityOf } from '../services/constants'
import SeverityBadge from './SeverityBadge'

export default function EventBadge({ eventType }) {
  const { t } = useLang()
  const sev = severityOf(eventType)
  return (
    <div className="inline-flex items-center gap-2">
      <span className="text-sm text-text-primary font-medium">
        {t(`eventTypes.${eventType}`, eventType)}
      </span>
      <SeverityBadge severity={sev} />
    </div>
  )
}