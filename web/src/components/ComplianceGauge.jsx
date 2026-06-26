import { useLang } from '../contexts/LanguageContext'

export default function ComplianceGauge({ score }) {
  const { t } = useLang()
  const pct = Math.max(0, Math.min(100, Math.round(score || 0)))
  const color = pct >= 80 ? '#22c55e' : pct >= 60 ? '#f59e0b' : '#ef4444'
  const radius = 70
  const circ = 2 * Math.PI * radius
  const offset = circ - (pct / 100) * circ

  return (
    <div className="bg-bg-card border border-border rounded-xl p-5 flex flex-col items-center">
      <h3 className="text-sm font-semibold text-text-secondary mb-4">{t('dashboard.complianceScore')}</h3>
      <div className="relative w-44 h-44">
        <svg className="w-full h-full -rotate-90" viewBox="0 0 160 160">
          <circle cx="80" cy="80" r={radius} fill="none" stroke="#2a2e3a" strokeWidth="12" />
          <circle
            cx="80" cy="80" r={radius}
            fill="none"
            stroke={color}
            strokeWidth="12"
            strokeLinecap="round"
            strokeDasharray={circ}
            strokeDashoffset={offset}
            className="gauge-progress"
          />
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className="text-4xl font-bold" style={{ color }}>{pct}%</span>
        </div>
      </div>
    </div>
  )
}