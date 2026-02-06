import { describe, expect, it } from 'vitest'
import { formatCost, formatCostSimple, formatTokens } from './format'

describe('formatTokens', () => {
  it('formats small values as raw numbers', () => {
    expect(formatTokens(0)).toBe('0')
    expect(formatTokens(999)).toBe('999')
  })

  it('formats thousands with K suffix', () => {
    expect(formatTokens(1000)).toBe('1.0K')
    expect(formatTokens(12500)).toBe('12.5K')
  })

  it('formats millions with M suffix', () => {
    expect(formatTokens(1_000_000)).toBe('1.0M')
    expect(formatTokens(2_350_000)).toBe('2.4M')
  })
})

describe('formatCost', () => {
  it('formats USD by default', () => {
    expect(formatCost(12.3)).toBe('$12.30')
  })

  it('formats other currencies', () => {
    expect(formatCost(10, 'EUR')).toContain('10.00')
  })
})

describe('formatCostSimple', () => {
  it('uses fixed two decimal places', () => {
    expect(formatCostSimple(5)).toBe('$5.00')
    expect(formatCostSimple(5.678)).toBe('$5.68')
  })
})
