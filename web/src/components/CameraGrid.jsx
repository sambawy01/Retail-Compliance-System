import { useLang } from '../contexts/LanguageContext'
import CameraCard from './CameraCard'

export default function CameraGrid({ cameras, onCameraClick, showStream = false }) {
  const { t } = useLang()
  if (!cameras || cameras.length === 0) {
    return (
      <div className="text-center py-12 text-text-muted bg-bg-card border border-border rounded-xl">
        {t('cameras.noCameras')}
      </div>
    )
  }
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
      {cameras.map((c) => (
        <CameraCard
          key={c.id}
          camera={c}
          onClick={onCameraClick ? () => onCameraClick(c) : undefined}
          showStream={showStream}
        />
      ))}
    </div>
  )
}