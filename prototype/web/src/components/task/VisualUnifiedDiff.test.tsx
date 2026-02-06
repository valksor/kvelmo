import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen } from '@/test/test-utils'
import { VisualUnifiedDiff } from './VisualUnifiedDiff'

const parseUnifiedDiffMock = vi.fn()

vi.mock('@/utils/unifiedDiff', () => ({
  parseUnifiedDiff: (diff: string) => parseUnifiedDiffMock(diff),
}))

describe('VisualUnifiedDiff', () => {
  beforeEach(() => {
    parseUnifiedDiffMock.mockReset()
  })

  it('shows fallback when diff is not parseable', () => {
    parseUnifiedDiffMock.mockReturnValue([])
    render(<VisualUnifiedDiff diff="raw" />)
    expect(screen.getByText(/Visual diff is unavailable/i)).toBeInTheDocument()
  })

  it('renders parsed rows in before/after table', () => {
    parseUnifiedDiffMock.mockReturnValue([
      {
        type: 'changed',
        leftLine: 10,
        rightLine: 10,
        leftText: 'old line',
        rightText: 'new line',
      },
    ])

    render(<VisualUnifiedDiff diff="@@" />)

    expect(screen.getByText('Before')).toBeInTheDocument()
    expect(screen.getByText('After')).toBeInTheDocument()
    expect(screen.getByText('old line')).toBeInTheDocument()
    expect(screen.getByText('new line')).toBeInTheDocument()
  })
})
