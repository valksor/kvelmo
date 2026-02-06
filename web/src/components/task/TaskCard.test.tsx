import { describe, expect, it } from 'vitest'
import { render, screen } from '@/test/test-utils'
import { TaskCard } from './TaskCard'
import type { TaskResponse } from '@/types/api'

describe('TaskCard', () => {
  it('renders nothing when task is inactive', () => {
    const { container } = render(<TaskCard task={{ active: false }} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('renders active task details and cost summary', () => {
    const task: TaskResponse = {
      active: true,
      task: {
        id: 'task-1',
        state: 'implementing',
        progress_phase: 'started',
        ref: 'github:1',
        branch: 'feature/x',
        worktree_path: '/tmp/w',
        started: '2026-01-01T00:00:00Z',
      },
      work: {
        title: 'Implement API',
        external_key: 'GH-1',
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
        costs: {
          total_input_tokens: 1200,
          total_output_tokens: 800,
          total_cost_usd: 0.25,
        },
      },
    }

    render(<TaskCard task={task} />)

    expect(screen.getByRole('link', { name: 'Implement API' })).toHaveAttribute('href', '/task/task-1')
    expect(screen.getByText('GH-1')).toBeInTheDocument()
    expect(screen.getByText('feature/x')).toBeInTheDocument()
    expect(screen.getByText('$0.25')).toBeInTheDocument()
    expect(screen.getByText('2.0K')).toBeInTheDocument()
  })
})
