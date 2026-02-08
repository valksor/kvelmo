import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@/test/test-utils'
import { LabelsCard } from './LabelsCard'

// Mock the API hooks
vi.mock('@/api/task', () => ({
  useTaskLabels: () => ({
    data: { labels: ['bug', 'feature'], count: 2 },
    isLoading: false,
  }),
  useAddLabel: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  useRemoveLabel: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
}))

describe('LabelsCard', () => {
  it('renders nothing when no active task', () => {
    render(<LabelsCard hasActiveTask={false} />)
    expect(screen.queryByText('Labels')).not.toBeInTheDocument()
  })

  it('renders labels when has active task', () => {
    render(<LabelsCard hasActiveTask={true} />)
    expect(screen.getByText('Labels (2)')).toBeInTheDocument()
    expect(screen.getByText('bug')).toBeInTheDocument()
    expect(screen.getByText('feature')).toBeInTheDocument()
  })

  it('shows add button', () => {
    render(<LabelsCard hasActiveTask={true} />)
    expect(screen.getByRole('button', { name: /add new label/i })).toBeInTheDocument()
  })

  it('shows remove buttons for each label', () => {
    render(<LabelsCard hasActiveTask={true} />)
    expect(screen.getByRole('button', { name: /remove label bug/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /remove label feature/i })).toBeInTheDocument()
  })
})
