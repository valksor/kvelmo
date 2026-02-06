import { describe, expect, it } from 'vitest'
import userEvent from '@testing-library/user-event'
import { render, screen } from '@/test/test-utils'
import { ActiveWorkCard } from './ActiveWorkCard'

describe('ActiveWorkCard', () => {
  it('renders nothing when task is missing', () => {
    const { container } = render(<ActiveWorkCard />)
    expect(container).toBeEmptyDOMElement()
  })

  it('renders task details and opens modal', async () => {
    const user = userEvent.setup()
    render(
      <ActiveWorkCard
        task={{
          id: 'task-1',
          state: 'implementing',
          ref: 'github:123',
          branch: 'feature/x',
          worktree_path: '/tmp/worktree',
          started: '2026-01-01T00:00:00Z',
        }}
        work={{
          title: 'Build feature',
          external_key: 'GH-123',
          description: 'Long description of the task',
        }}
      />
    )

    expect(screen.getByText('Build feature')).toBeInTheDocument()
    expect(screen.getByText('GH-123')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /technical details/i }))
    expect(screen.getByText('feature/x')).toBeInTheDocument()
    expect(screen.getByText('/tmp/worktree')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /view/i }))
    expect(screen.getAllByText('Build feature').length).toBeGreaterThan(0)
    expect(screen.getByText('Content')).toBeInTheDocument()
    expect(screen.getAllByText('Long description of the task').length).toBeGreaterThan(1)
  })
})
