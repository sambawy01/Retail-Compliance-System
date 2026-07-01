// Loading skeleton components that match content layout to prevent layout shift.
export function CardSkeleton({ count = 1 }) {
  return (
    <>
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="bg-bg-card border border-border rounded-xl p-4 animate-pulse">
          <div className="flex items-start gap-3">
            <div className="w-14 h-14 rounded-full bg-border" />
            <div className="flex-1 space-y-2">
              <div className="h-4 bg-border rounded w-3/4" />
              <div className="h-3 bg-border rounded w-1/2" />
              <div className="h-3 bg-border rounded w-2/3" />
            </div>
            <div className="w-12 h-12 rounded-full bg-border" />
          </div>
        </div>
      ))}
    </>
  )
}

export function TableSkeleton({ rows = 5 }) {
  return (
    <div className="bg-bg-card border border-border rounded-xl overflow-hidden">
      <div className="divide-y divide-border">
        {Array.from({ length: rows }).map((_, i) => (
          <div key={i} className="p-3 flex items-center gap-3 animate-pulse">
            <div className="h-4 bg-border rounded flex-1" />
            <div className="h-4 bg-border rounded w-20" />
            <div className="h-4 bg-border rounded w-16" />
          </div>
        ))}
      </div>
    </div>
  )
}

export function GridSkeleton({ count = 6 }) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="bg-bg-card border border-border rounded-xl overflow-hidden animate-pulse">
          <div className="aspect-video bg-border" />
          <div className="p-4 space-y-2">
            <div className="h-4 bg-border rounded w-3/4" />
            <div className="h-3 bg-border rounded w-1/2" />
          </div>
        </div>
      ))}
    </div>
  )
}