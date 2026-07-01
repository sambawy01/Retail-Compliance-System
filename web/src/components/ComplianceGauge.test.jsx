import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import ComplianceGauge from '../components/ComplianceGauge'
import { LanguageProvider } from '../contexts/LanguageContext'

function renderWithProvider(ui) {
  return render(<LanguageProvider>{ui}</LanguageProvider>)
}

describe('ComplianceGauge', () => {
  it('renders the score percentage', () => {
    renderWithProvider(<ComplianceGauge score={75} />)
    expect(screen.getByText('75%')).toBeInTheDocument()
  })

  it('renders 100% for perfect score', () => {
    renderWithProvider(<ComplianceGauge score={100} />)
    expect(screen.getByText('100%')).toBeInTheDocument()
  })

  it('renders 0% for zero score', () => {
    renderWithProvider(<ComplianceGauge score={0} />)
    expect(screen.getByText('0%')).toBeInTheDocument()
  })

  it('clamps scores above 100', () => {
    renderWithProvider(<ComplianceGauge score={150} />)
    expect(screen.getByText('100%')).toBeInTheDocument()
  })

  it('clamps scores below 0', () => {
    renderWithProvider(<ComplianceGauge score={-20} />)
    expect(screen.getByText('0%')).toBeInTheDocument()
  })

  it('rounds decimal scores', () => {
    renderWithProvider(<ComplianceGauge score={73.7} />)
    expect(screen.getByText('74%')).toBeInTheDocument()
  })

  it('renders the compliance score label', () => {
    renderWithProvider(<ComplianceGauge score={80} />)
    expect(screen.getByText('Compliance Score')).toBeInTheDocument()
  })

  it('uses green colour for score >= 80', () => {
    const { container } = renderWithProvider(<ComplianceGauge score={85} />)
    const circle = container.querySelector('.gauge-progress')
    expect(circle.getAttribute('stroke')).toBe('#22c55e')
  })

  it('uses amber colour for score 60-79', () => {
    const { container } = renderWithProvider(<ComplianceGauge score={65} />)
    const circle = container.querySelector('.gauge-progress')
    expect(circle.getAttribute('stroke')).toBe('#f59e0b')
  })

  it('uses red colour for score < 60', () => {
    const { container } = renderWithProvider(<ComplianceGauge score={45} />)
    const circle = container.querySelector('.gauge-progress')
    expect(circle.getAttribute('stroke')).toBe('#ef4444')
  })
})