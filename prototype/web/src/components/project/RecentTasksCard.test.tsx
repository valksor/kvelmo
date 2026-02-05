import { describe, it, expect } from 'vitest'
import { render, screen } from '@/test/test-utils'
import { RecentTasksCard } from './RecentTasksCard'
import type { TaskHistoryItem } from '@/types/api'

const mockTasks: TaskHistoryItem[] = [
  {
    id: 'task-1',
    title: 'Implement login feature',
    state: 'done',
    created_at: '2026-01-15T10:00:00Z',
  },
  {
    id: 'task-2',
    title: 'Fix navigation bug',
    state: 'implementing',
    created_at: '2026-01-16T14:30:00Z',
    worktree_path: '/tmp/worktree/task-2',
  },
  {
    id: 'task-3',
    title: 'Add unit tests',
    state: 'failed',
    created_at: '2026-01-17T09:15:00Z',
  },
]

describe('RecentTasksCard', () => {
  it('renders heading', () => {
    render(<RecentTasksCard tasks={mockTasks} />)

    expect(screen.getByText('Recent Tasks')).toBeInTheDocument()
  })

  it('renders view all link', () => {
    render(<RecentTasksCard tasks={mockTasks} />)

    const viewAllLink = screen.getByRole('link', { name: /view all/i })
    expect(viewAllLink).toHaveAttribute('href', '/history')
  })

  it('renders task titles', () => {
    render(<RecentTasksCard tasks={mockTasks} />)

    expect(screen.getByText('Implement login feature')).toBeInTheDocument()
    expect(screen.getByText('Fix navigation bug')).toBeInTheDocument()
    expect(screen.getByText('Add unit tests')).toBeInTheDocument()
  })

  it('renders state badges', () => {
    render(<RecentTasksCard tasks={mockTasks} />)

    expect(screen.getByText('done')).toBeInTheDocument()
    expect(screen.getByText('implementing')).toBeInTheDocument()
    expect(screen.getByText('failed')).toBeInTheDocument()
  })

  it('shows worktree indicator when present', () => {
    render(<RecentTasksCard tasks={mockTasks} />)

    expect(screen.getByText('Worktree')).toBeInTheDocument()
  })

  it('shows empty state when no tasks', () => {
    render(<RecentTasksCard tasks={[]} />)

    expect(screen.getByText('No tasks yet')).toBeInTheDocument()
  })

  it('shows loading state', () => {
    render(<RecentTasksCard isLoading />)

    expect(document.querySelector('.animate-spin')).toBeInTheDocument()
  })

  it('limits display to 10 tasks', () => {
    const manyTasks: TaskHistoryItem[] = Array.from({ length: 15 }, (_, i) => ({
      id: `task-${i}`,
      title: `Task ${i}`,
      state: 'done' as const,
      created_at: '2026-01-15T10:00:00Z',
    }))

    render(<RecentTasksCard tasks={manyTasks} />)

    // Should only render 10 tasks
    const taskLinks = screen.getAllByRole('link').filter(
      (link) => link.getAttribute('href')?.startsWith('/task/')
    )
    expect(taskLinks).toHaveLength(10)
  })

  it('renders task links to detail page', () => {
    render(<RecentTasksCard tasks={mockTasks} />)

    const taskLink = screen.getByText('Implement login feature').closest('a')
    expect(taskLink).toHaveAttribute('href', '/task/task-1')
  })

  it('renders when tasks is undefined', () => {
    render(<RecentTasksCard />)

    expect(screen.getByText('No tasks yet')).toBeInTheDocument()
  })
})
