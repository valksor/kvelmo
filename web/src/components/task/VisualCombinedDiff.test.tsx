import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen } from '@/test/test-utils'
import { VisualCombinedDiff } from './VisualCombinedDiff'

const parseUnifiedDiffMock = vi.fn()

vi.mock('@/utils/unifiedDiff', () => ({
  parseUnifiedDiff: (diff: string) => parseUnifiedDiffMock(diff),
}))

describe('VisualCombinedDiff', () => {
  beforeEach(() => {
    parseUnifiedDiffMock.mockReset()
  })

  it('shows fallback when diff is not parseable', () => {
    parseUnifiedDiffMock.mockReturnValue([])
    render(<VisualCombinedDiff diff="raw" />)
    expect(screen.getByText(/Visual diff is unavailable/i)).toBeInTheDocument()
  })

  it('renders context lines with neutral styling', () => {
    parseUnifiedDiffMock.mockReturnValue([
      {
        type: 'context',
        leftLine: 5,
        rightLine: 5,
        leftText: 'unchanged line',
        rightText: 'unchanged line',
      },
    ])

    render(<VisualCombinedDiff diff="@@" />)
    expect(screen.getByText('unchanged line')).toBeInTheDocument()
    expect(screen.getByText('5')).toBeInTheDocument()
  })

  it('renders removed lines with minus prefix', () => {
    parseUnifiedDiffMock.mockReturnValue([
      {
        type: 'removed',
        leftLine: 10,
        leftText: 'deleted line',
      },
    ])

    render(<VisualCombinedDiff diff="@@" />)
    expect(screen.getByText('deleted line')).toBeInTheDocument()
    expect(screen.getByText('-')).toBeInTheDocument()
    expect(screen.getByText('10')).toBeInTheDocument()
  })

  it('renders added lines with plus prefix', () => {
    parseUnifiedDiffMock.mockReturnValue([
      {
        type: 'added',
        rightLine: 15,
        rightText: 'new line',
      },
    ])

    render(<VisualCombinedDiff diff="@@" />)
    expect(screen.getByText('new line')).toBeInTheDocument()
    expect(screen.getByText('+')).toBeInTheDocument()
    expect(screen.getByText('15')).toBeInTheDocument()
  })

  it('renders changed rows as two separate lines', () => {
    parseUnifiedDiffMock.mockReturnValue([
      {
        type: 'changed',
        leftLine: 20,
        rightLine: 20,
        leftText: 'old version',
        rightText: 'new version',
      },
    ])

    render(<VisualCombinedDiff diff="@@" />)
    expect(screen.getByText('old version')).toBeInTheDocument()
    expect(screen.getByText('new version')).toBeInTheDocument()
    expect(screen.getByText('-')).toBeInTheDocument()
    expect(screen.getByText('+')).toBeInTheDocument()
  })
})
