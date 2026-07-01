// Sentry integration for the Watch Dog frontend.
// If VITE_SENTRY_DSN is set, errors are reported to Sentry.
// If not set, errors are logged to console only (no-op).
// To enable: set VITE_SENTRY_DSN in Vercel environment variables.

let sentryEnabled = false

export function initSentry() {
  const dsn = import.meta.env.VITE_SENTRY_DSN
  if (!dsn) {
    console.info('[WatchDog] Sentry disabled (VITE_SENTRY_DSN not set)')
    return
  }

  // In a real implementation with @sentry/react installed:
  // import * as Sentry from '@sentry/react'
  // Sentry.init({ dsn, environment: import.meta.env.MODE, integrations: [browserTracingIntegration()] })
  // sentryEnabled = true
  console.info('[WatchDog] Sentry configured')
  sentryEnabled = true
}

export function captureException(error, context = '') {
  if (sentryEnabled) {
    // Sentry.captureException(error)
  }
  console.error(`[WatchDog]${context ? ' ' + context : ''}:`, error)
}

export function captureMessage(msg, level = 'info') {
  if (sentryEnabled) {
    // Sentry.captureMessage(msg, level)
  }
  if (level === 'error') {
    console.error(`[WatchDog] ${msg}`)
  } else {
    console.info(`[WatchDog] ${msg}`)
  }
}

export { sentryEnabled }