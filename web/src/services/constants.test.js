import { describe, it, expect } from 'vitest'
import {
  severityOf,
  sevColor,
  EVENT_SEVERITY,
  CRITICAL_TYPES,
  WARNING_TYPES,
  INFO_TYPES,
  ALL_EVENT_TYPES,
  ZONE_TYPES,
  fmtTime,
  fmtDate,
} from '../services/constants'

describe('severityOf', () => {
  it('returns critical for slip_fall', () => {
    expect(severityOf('slip_fall')).toBe('critical')
  })

  it('returns warning for uniform_violation', () => {
    expect(severityOf('uniform_violation')).toBe('warning')
  })

  it('returns info for loyalty_recognized', () => {
    expect(severityOf('loyalty_recognized')).toBe('info')
  })

  it('returns info for unknown event type', () => {
    expect(severityOf('unknown_event')).toBe('info')
  })

  it('returns info for undefined input', () => {
    expect(severityOf(undefined)).toBe('info')
  })
})

describe('sevColor', () => {
  it('returns red for critical', () => {
    expect(sevColor('critical')).toBe('#ef4444')
  })

  it('returns amber for warning', () => {
    expect(sevColor('warning')).toBe('#f59e0b')
  })

  it('returns blue for info', () => {
    expect(sevColor('info')).toBe('#3b82f6')
  })
})

describe('event type groups', () => {
  it('CRITICAL_TYPES contains slip_fall, cash_drawer, after_hours, buddy_punch, blocked_exit', () => {
    expect(CRITICAL_TYPES).toContain('slip_fall')
    expect(CRITICAL_TYPES).toContain('cash_drawer')
    expect(CRITICAL_TYPES).toContain('after_hours')
    expect(CRITICAL_TYPES).toContain('buddy_punch')
    expect(CRITICAL_TYPES).toContain('blocked_exit')
  })

  it('WARNING_TYPES contains uniform_violation, phone_usage', () => {
    expect(WARNING_TYPES).toContain('uniform_violation')
    expect(WARNING_TYPES).toContain('phone_usage')
  })

  it('INFO_TYPES contains loyalty_recognized, occupancy_update', () => {
    expect(INFO_TYPES).toContain('loyalty_recognized')
    expect(INFO_TYPES).toContain('occupancy_update')
  })

  it('ALL_EVENT_TYPES has 16 types', () => {
    expect(ALL_EVENT_TYPES).toHaveLength(16)
  })

  it('every event type maps to a severity in EVENT_SEVERITY', () => {
    for (const t of ALL_EVENT_TYPES) {
      expect(EVENT_SEVERITY[t]).toBeDefined()
    }
  })
})

describe('ZONE_TYPES', () => {
  it('contains checkout, aisles, stockroom, back_office, entrance, restroom, restricted, privacy_mask', () => {
    expect(ZONE_TYPES).toContain('checkout')
    expect(ZONE_TYPES).toContain('aisles')
    expect(ZONE_TYPES).toContain('privacy_mask')
    expect(ZONE_TYPES).toHaveLength(8)
  })
})

describe('fmtTime', () => {
  it('returns dash for null/undefined', () => {
    expect(fmtTime(null)).toBe('—')
    expect(fmtTime(undefined)).toBe('—')
  })

  it('returns the input string for invalid dates', () => {
    expect(fmtTime('not-a-date')).toBe('not-a-date')
  })

  it('returns a locale string for valid dates', () => {
    const d = new Date('2025-01-15T10:30:00Z')
    const result = fmtTime(d.toISOString())
    expect(result).not.toBe('—')
    expect(typeof result).toBe('string')
  })
})

describe('fmtDate', () => {
  it('returns dash for null/undefined', () => {
    expect(fmtDate(null)).toBe('—')
    expect(fmtDate(undefined)).toBe('—')
  })

  it('returns the input string for invalid dates', () => {
    expect(fmtDate('not-a-date')).toBe('not-a-date')
  })

  it('returns a locale date string for valid dates', () => {
    const d = new Date('2025-01-15T10:30:00Z')
    const result = fmtDate(d.toISOString())
    expect(result).not.toBe('—')
    expect(typeof result).toBe('string')
  })
})