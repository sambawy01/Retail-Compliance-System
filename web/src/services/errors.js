// Centralized error handling for the Watch Dog frontend.
// Reports errors to console and optionally to Sentry (if initialized).
import { captureException } from './sentry'

// Report an error that was caught in a catch block.
export function reportError(error, context = '') {
  captureException(error, context)
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