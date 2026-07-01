import { describe, it, expect, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { LanguageProvider, useLang } from '../contexts/LanguageContext'

function wrapper({ children }) {
  return <LanguageProvider>{children}</LanguageProvider>
}

describe('LanguageContext', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('defaults to English', () => {
    const { result } = renderHook(() => useLang(), { wrapper })
    expect(result.current.lang).toBe('en')
    expect(result.current.isRTL).toBe(false)
    expect(result.current.dir).toBe('ltr')
  })

  it('toggle switches en to ar', () => {
    const { result } = renderHook(() => useLang(), { wrapper })
    act(() => result.current.toggle())
    expect(result.current.lang).toBe('ar')
    expect(result.current.isRTL).toBe(true)
    expect(result.current.dir).toBe('rtl')
  })

  it('toggle switches ar back to en', () => {
    const { result } = renderHook(() => useLang(), { wrapper })
    act(() => result.current.toggle())
    act(() => result.current.toggle())
    expect(result.current.lang).toBe('en')
  })

  it('t() returns English string for known path', () => {
    const { result } = renderHook(() => useLang(), { wrapper })
    expect(result.current.t('app.name')).toBe('Watch Dog')
    expect(result.current.t('nav.dashboard')).toBe('Dashboard')
  })

  it('t() returns Arabic string when lang is ar', () => {
    const { result } = renderHook(() => useLang(), { wrapper })
    act(() => result.current.toggle())
    expect(result.current.t('app.name')).toBe('واتش دوغ')
    expect(result.current.t('nav.dashboard')).toBe('لوحة التحكم')
  })

  it('t() returns fallback for unknown path', () => {
    const { result } = renderHook(() => useLang(), { wrapper })
    expect(result.current.t('nonexistent.path', 'Fallback')).toBe('Fallback')
  })

  it('t() returns path itself for unknown path without fallback', () => {
    const { result } = renderHook(() => useLang(), { wrapper })
    expect(result.current.t('nonexistent.path')).toBe('nonexistent.path')
  })

  it('t() resolves English plural form: 1 = one', () => {
    const { result } = renderHook(() => useLang(), { wrapper })
    const val = result.current.t('dashboard.critical', 1)
    expect(val).toBe('1 Critical')
  })

  it('t() resolves English plural form: 0 = zero', () => {
    const { result } = renderHook(() => useLang(), { wrapper })
    const val = result.current.t('dashboard.critical', 0)
    expect(val).toBe('No critical')
  })

  it('t() resolves English plural form: 5 = other', () => {
    const { result } = renderHook(() => useLang(), { wrapper })
    const val = result.current.t('dashboard.critical', 5)
    expect(val).toBe('5 Critical')
  })

  it('t() resolves Arabic plural form: 1 = one', () => {
    const { result } = renderHook(() => useLang(), { wrapper })
    act(() => result.current.toggle())
    const val = result.current.t('dashboard.critical', 1)
    expect(val).toBe('حدث حرج واحد')
  })

  it('t() resolves Arabic plural form: 0 = zero', () => {
    const { result } = renderHook(() => useLang(), { wrapper })
    act(() => result.current.toggle())
    const val = result.current.t('dashboard.critical', 0)
    expect(val).toBe('لا أحداث حرجة')
  })

  it('t() resolves Arabic plural form: 3 = few', () => {
    const { result } = renderHook(() => useLang(), { wrapper })
    act(() => result.current.toggle())
    const val = result.current.t('dashboard.critical', 3)
    expect(val).toBe('أحداث حرجة')
  })

  it('persists lang to localStorage', () => {
    const { result } = renderHook(() => useLang(), { wrapper })
    act(() => result.current.toggle())
    expect(localStorage.getItem('watchdog_lang')).toBe('ar')
  })
})