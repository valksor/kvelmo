import { describe, expect, it } from 'vitest'
import { render, screen } from '@/test/test-utils'
import { TaskSummaryCard } from './TaskSummaryCard'

describe('TaskSummaryCard', () => {
  it('returns nothing when task is missing', () => {
    const { container } = render(<TaskSummaryCard />)
    expect(container).toBeEmptyDOMElement()
  })

  it('renders title, branch, and details link', () => {
    render(
      <TaskSummaryCard
        task={{
          id: 'task-1',
          state: 'implementing',
          ref: 'github:1',
          branch: 'feature/test',
          started: '2026-01-01T00:00:00Z',
        }}
        work={{ title: 'Implement feature' }}
      />
    )

    expect(screen.getByText('Implement feature')).toBeInTheDocument()
    expect(screen.getByText('feature/test')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /view/i })).toHaveAttribute('href', '/task/task-1')
    expect(screen.getByText('implementing')).toBeInTheDocument()
  })

  it('uses progress phase label for idle state', () => {
    render(
      <TaskSummaryCard
        task={{
          id: 'task-2',
          state: 'idle',
          ref: 'quick:2',
        }}
        progressPhase="planned"
      />
    )

    expect(screen.getByText('planned')).toBeInTheDocument()
  })
})
