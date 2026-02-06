import { describe, expect, it, vi, beforeEach } from 'vitest'
import userEvent from '@testing-library/user-event'
import { render, screen } from '@/test/test-utils'
import { CostsCard } from './CostsCard'

const useTaskCostsMock = vi.fn()

vi.mock('@/api/task', () => ({
  useTaskCosts: (taskId?: string) => useTaskCostsMock(taskId),
}))

describe('CostsCard', () => {
  beforeEach(() => {
    useTaskCostsMock.mockReset()
  })

  it('renders loading state', () => {
    useTaskCostsMock.mockReturnValue({ data: undefined, isLoading: true })
    render(<CostsCard taskId="task-1" />)
    expect(screen.getByText('Loading costs...')).toBeInTheDocument()
  })

  it('renders nothing when no costs are available', () => {
    useTaskCostsMock.mockReturnValue({ data: undefined, isLoading: false })
    const { container } = render(<CostsCard taskId="task-1" />)
    expect(container).toBeEmptyDOMElement()
  })

  it('renders summary and toggles step breakdown', async () => {
    const user = userEvent.setup()
    useTaskCostsMock.mockReturnValue({
      isLoading: false,
      data: {
        total_cost_usd: 0.1234,
        total_tokens: 12345,
        input_tokens: 5000,
        output_tokens: 6000,
        cached_tokens: 1345,
        cached_percent: 10.9,
        budget: {
          type: 'monthly',
          used: '$12',
          max: '$100',
          pct: 12,
          warned: false,
          limit_hit: false,
        },
        steps: [
          { name: 'planning', total_tokens: 1000, cost: '$0.01' },
          { name: 'implementing', total_tokens: 2000, cost: '$0.02' },
        ],
      },
    })

    render(<CostsCard taskId="task-1" />)

    expect(screen.getByText('Cost Summary')).toBeInTheDocument()
    expect(screen.getByText('$0.1234')).toBeInTheDocument()
    expect(screen.getByText('12,345')).toBeInTheDocument()
    expect(screen.queryByText('Per-Step Costs')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /show breakdown/i }))
    expect(screen.getByText('Per-Step Costs')).toBeInTheDocument()
    expect(screen.getByText('planning')).toBeInTheDocument()
  })
})
