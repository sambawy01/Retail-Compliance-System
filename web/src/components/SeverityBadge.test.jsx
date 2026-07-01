import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import SeverityBadge from '../components/SeverityBadge'
import { LanguageProvider } from '../contexts/LanguageContext'

function renderWithProvider(ui) {
  return render(<LanguageProvider>{ui}</LanguageProvider>)
}

describe('SeverityBadge', () => {
  it('renders critical label', () => {
    renderWithProvider(<SeverityBadge severity="critical" />)
    expect(screen.getByText(/critical/i)).toBeInTheDocument()
  })

  it('renders warning label', () => {
    renderWithProvider(<SeverityBadge severity="warning" />)
    expect(screen.getByText(/warning/i)).toBeInTheDocument()
  })

  it('renders info label', () => {
    renderWithProvider(<SeverityBadge severity="info" />)
    expect(screen.getByText(/info/i)).toBeInTheDocument()
  })

  it('applies critical colour style', () => {
    const { container } = renderWithProvider(<SeverityBadge severity="critical" />)
    const badge = container.firstChild
    expect(badge.style.color).toBe('rgb(239, 68, 68)')
  })

  it('applies warning colour style', () => {
    const { container } = renderWithProvider(<SeverityBadge severity="warning" />)
    const badge = container.firstChild
    expect(badge.style.color).toBe('rgb(245, 158, 11)')
  })

  it('applies info colour style', () => {
    const { container } = renderWithProvider(<SeverityBadge severity="info" />)
    const badge = container.firstChild
    expect(badge.style.color).toBe('rgb(59, 130, 246)')
  })
})