// Centralized error handling for the Watch Dog frontend.
// Instead of silent catch blocks, use these helpers to report errors
// to the console and optionally to an error tracking service.

// Report an error that was caught in a catch block.
// In production, this could send to Sentry or similar.
export function reportError(error, context = '') {
  if (error && error.message) {
    console.error(`[WatchDog]${context ? ' ' + context : ''}:`, error.message)
  } else if (error) {
    console.error(`[WatchDog]${context ? ' ' + context : ''}:`, error)
  }
}

// Safe async wrapper — catches errors and reports them.
// Use: safeAsync(() => apiCall(), 'loading cameras')
export function safeAsync(asyncFn, context = '') {
  return asyncFn().catch((error) => {
    reportError(error, context)
    return null
  })
}

// Safe effect helper — for use in useEffect with async API calls.
// Returns a cleanup-aware wrapper.
export function safeEffect(asyncFn, context = '') {
  let active = true
  const promise = asyncFn()
  promise.catch((error) => {
    if (active) reportError(error, context)
  })
  return () => { active = false }
}