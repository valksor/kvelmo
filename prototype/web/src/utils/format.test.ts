import { describe, expect, it } from 'vitest'
import { formatCost, formatCostSimple, formatDate, formatDateTime, formatTimestamp, formatTokens } from './format'

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

describe('formatDate', () => {
  it('formats date as dd.mm.yyyy', () => {
    // Use UTC date to avoid timezone issues in tests
    const date = new Date(Date.UTC(2026, 1, 8)) // Feb 8, 2026
    // Note: getDate/getMonth use local timezone, so we test with ISO string
    expect(formatDate('2026-02-08T12:00:00Z')).toBe('08.02.2026')
  })

  it('accepts Date object', () => {
    const date = new Date(2026, 1, 8, 12, 0, 0) // Feb 8, 2026 (local)
    expect(formatDate(date)).toBe('08.02.2026')
  })

  it('returns empty string for invalid date', () => {
    expect(formatDate('invalid')).toBe('')
  })

  it('pads single-digit day and month', () => {
    expect(formatDate('2026-01-05T12:00:00Z')).toBe('05.01.2026')
  })
})

describe('formatDateTime', () => {
  it('formats datetime as dd.mm.yyyy hh:mm', () => {
    const date = new Date(2026, 1, 8, 15, 4, 0) // Feb 8, 2026 15:04 (local)
    expect(formatDateTime(date)).toBe('08.02.2026 15:04')
  })

  it('pads single-digit hours and minutes', () => {
    const date = new Date(2026, 1, 8, 9, 5, 0) // Feb 8, 2026 09:05 (local)
    expect(formatDateTime(date)).toBe('08.02.2026 09:05')
  })

  it('returns empty string for invalid date', () => {
    expect(formatDateTime('invalid')).toBe('')
  })
})

describe('formatTimestamp', () => {
  it('formats timestamp as dd.mm.yyyy hh:mm:ss', () => {
    const date = new Date(2026, 1, 8, 15, 4, 35) // Feb 8, 2026 15:04:35 (local)
    expect(formatTimestamp(date)).toBe('08.02.2026 15:04:35')
  })

  it('pads single-digit seconds', () => {
    const date = new Date(2026, 1, 8, 15, 4, 5) // Feb 8, 2026 15:04:05 (local)
    expect(formatTimestamp(date)).toBe('08.02.2026 15:04:05')
  })

  it('returns empty string for invalid date', () => {
    expect(formatTimestamp('invalid')).toBe('')
  })
})
