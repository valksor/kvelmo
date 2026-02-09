import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@/test/test-utils'
import { TasksPanel } from './TasksPanel'

// Mock hooks
const useQueueTasksMock = vi.fn()
const submitTasksMutateAsyncMock = vi.fn()
const reorderTasksMutateAsyncMock = vi.fn()
const startImplMutateAsyncMock = vi.fn()

vi.mock('@/api/project-planning', () => ({
  useQueueTasks: (queueId?: string) => useQueueTasksMock(queueId),
  useSubmitTasks: () => ({
    mutateAsync: submitTasksMutateAsyncMock,
    isPending: false,
  }),
  useReorderTasks: () => ({
    mutateAsync: reorderTasksMutateAsyncMock,
    isPending: false,
  }),
  useStartImplementation: () => ({
    mutateAsync: startImplMutateAsyncMock,
    isPending: false,
  }),
}))

describe('TasksPanel', () => {
  const mockOnEditTask = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    useQueueTasksMock.mockReturnValue({
      data: { tasks: [], queue_title: 'Test Queue' },
      isLoading: false,
      error: null,
    })
  })

  it('shows message when no queue selected', () => {
    render(<TasksPanel onEditTask={mockOnEditTask} />)

    expect(screen.getByText('Select a queue to view its tasks')).toBeInTheDocument()
  })

  it('shows loading spinner while loading', () => {
    useQueueTasksMock.mockReturnValue({
      data: null,
      isLoading: true,
      error: null,
    })

    render(<TasksPanel queueId="queue-1" onEditTask={mockOnEditTask} />)

    expect(screen.queryByText('Test Queue')).not.toBeInTheDocument()
  })

  it('shows error message when loading fails', () => {
    useQueueTasksMock.mockReturnValue({
      data: null,
      isLoading: false,
      error: { message: 'Failed to fetch' },
    })

    render(<TasksPanel queueId="queue-1" onEditTask={mockOnEditTask} />)

    expect(screen.getByText(/Failed to load tasks: Failed to fetch/)).toBeInTheDocument()
  })

  it('shows empty state when no tasks', () => {
    render(<TasksPanel queueId="queue-1" onEditTask={mockOnEditTask} />)

    expect(screen.getByText('No tasks in this queue')).toBeInTheDocument()
  })

  it('shows queue title and task count', () => {
    useQueueTasksMock.mockReturnValue({
      data: {
        tasks: [
          { id: 'task-1', title: 'Task 1', status: 'ready', priority: 3, depends_on: [] },
          { id: 'task-2', title: 'Task 2', status: 'pending', priority: 2, depends_on: [] },
        ],
        queue_title: 'Sprint 1',
      },
      isLoading: false,
      error: null,
    })

    render(<TasksPanel queueId="queue-1" onEditTask={mockOnEditTask} />)

    expect(screen.getByText('Sprint 1')).toBeInTheDocument()
    expect(screen.getByText('2 tasks')).toBeInTheDocument()
  })

  it('renders tasks in table', () => {
    useQueueTasksMock.mockReturnValue({
      data: {
        tasks: [
          { id: 'task-1', title: 'Fix bug', status: 'ready', priority: 2, depends_on: [] },
          { id: 'task-2', title: 'Add feature', status: 'blocked', priority: 3, depends_on: ['task-1'] },
        ],
        queue_title: 'Tasks',
      },
      isLoading: false,
      error: null,
    })

    render(<TasksPanel queueId="queue-1" onEditTask={mockOnEditTask} />)

    expect(screen.getByText('Fix bug')).toBeInTheDocument()
    expect(screen.getByText('Add feature')).toBeInTheDocument()
  })

  it('shows status badges', () => {
    useQueueTasksMock.mockReturnValue({
      data: {
        tasks: [
          { id: 'task-1', title: 'Ready task', status: 'ready', priority: 3, depends_on: [] },
          { id: 'task-2', title: 'Pending task', status: 'pending', priority: 3, depends_on: [] },
          { id: 'task-3', title: 'Blocked task', status: 'blocked', priority: 3, depends_on: [] },
        ],
        queue_title: 'Tasks',
      },
      isLoading: false,
      error: null,
    })

    render(<TasksPanel queueId="queue-1" onEditTask={mockOnEditTask} />)

    expect(screen.getByText('ready')).toBeInTheDocument()
    expect(screen.getByText('pending')).toBeInTheDocument()
    expect(screen.getByText('blocked')).toBeInTheDocument()
  })

  it('shows priority labels', () => {
    useQueueTasksMock.mockReturnValue({
      data: {
        tasks: [
          { id: 'task-1', title: 'Task', status: 'ready', priority: 1, depends_on: [] },
          { id: 'task-2', title: 'Task 2', status: 'ready', priority: 2, depends_on: [] },
          { id: 'task-3', title: 'Task 3', status: 'ready', priority: 3, depends_on: [] },
          { id: 'task-4', title: 'Task 4', status: 'ready', priority: 4, depends_on: [] },
          { id: 'task-5', title: 'Task 5', status: 'ready', priority: 5, depends_on: [] },
        ],
        queue_title: 'Tasks',
      },
      isLoading: false,
      error: null,
    })

    render(<TasksPanel queueId="queue-1" onEditTask={mockOnEditTask} />)

    expect(screen.getByText('Highest')).toBeInTheDocument()
    expect(screen.getByText('High')).toBeInTheDocument()
    expect(screen.getByText('Medium')).toBeInTheDocument()
    expect(screen.getByText('Low')).toBeInTheDocument()
    expect(screen.getByText('Lowest')).toBeInTheDocument()
  })

  it('shows dependency count', () => {
    useQueueTasksMock.mockReturnValue({
      data: {
        tasks: [
          { id: 'task-1', title: 'Task', status: 'ready', priority: 3, depends_on: ['a', 'b'] },
        ],
        queue_title: 'Tasks',
      },
      isLoading: false,
      error: null,
    })

    render(<TasksPanel queueId="queue-1" onEditTask={mockOnEditTask} />)

    expect(screen.getByText('2 deps')).toBeInTheDocument()
  })

  it('calls onEditTask when edit button clicked', () => {
    const task = { id: 'task-1', title: 'Task', status: 'ready', priority: 3, depends_on: [] }
    useQueueTasksMock.mockReturnValue({
      data: { tasks: [task], queue_title: 'Tasks' },
      isLoading: false,
      error: null,
    })

    render(<TasksPanel queueId="queue-1" onEditTask={mockOnEditTask} />)

    fireEvent.click(screen.getByRole('button', { name: 'Edit task' }))

    expect(mockOnEditTask).toHaveBeenCalledWith(task)
  })

  it('calls AI reorder when button clicked', async () => {
    reorderTasksMutateAsyncMock.mockResolvedValue({})

    useQueueTasksMock.mockReturnValue({
      data: {
        tasks: [{ id: 'task-1', title: 'Task', status: 'ready', priority: 3, depends_on: [] }],
        queue_title: 'Tasks',
      },
      isLoading: false,
      error: null,
    })

    render(<TasksPanel queueId="queue-1" onEditTask={mockOnEditTask} />)

    fireEvent.click(screen.getByRole('button', { name: /ai suggest order/i }))

    await waitFor(() => {
      expect(reorderTasksMutateAsyncMock).toHaveBeenCalledWith({ queue_id: 'queue-1' })
    })
  })

  it('calls submit tasks when Submit button clicked', async () => {
    submitTasksMutateAsyncMock.mockResolvedValue({})

    useQueueTasksMock.mockReturnValue({
      data: {
        tasks: [{ id: 'task-1', title: 'Task', status: 'ready', priority: 3, depends_on: [] }],
        queue_title: 'Tasks',
      },
      isLoading: false,
      error: null,
    })

    render(<TasksPanel queueId="queue-1" onEditTask={mockOnEditTask} />)

    fireEvent.click(screen.getByRole('button', { name: 'Submit' }))

    await waitFor(() => {
      expect(submitTasksMutateAsyncMock).toHaveBeenCalledWith({
        queue_id: 'queue-1',
        provider: 'github',
        mention: undefined,
        dry_run: false,
      })
    })
  })

  it('submits with custom provider and mention', async () => {
    submitTasksMutateAsyncMock.mockResolvedValue({})

    useQueueTasksMock.mockReturnValue({
      data: {
        tasks: [{ id: 'task-1', title: 'Task', status: 'ready', priority: 3, depends_on: [] }],
        queue_title: 'Tasks',
      },
      isLoading: false,
      error: null,
    })

    render(<TasksPanel queueId="queue-1" onEditTask={mockOnEditTask} />)

    // Change provider
    const providerSelect = screen.getByRole('combobox')
    fireEvent.change(providerSelect, { target: { value: 'gitlab' } })

    // Add mention
    const mentionInput = screen.getByPlaceholderText('@username')
    fireEvent.change(mentionInput, { target: { value: '@dev-team' } })

    // Enable dry run
    const dryRunCheckbox = screen.getByRole('checkbox')
    fireEvent.click(dryRunCheckbox)

    // Submit
    fireEvent.click(screen.getByRole('button', { name: 'Submit' }))

    await waitFor(() => {
      expect(submitTasksMutateAsyncMock).toHaveBeenCalledWith({
        queue_id: 'queue-1',
        provider: 'gitlab',
        mention: '@dev-team',
        dry_run: true,
      })
    })
  })

  it('calls start implementation when button clicked', async () => {
    startImplMutateAsyncMock.mockResolvedValue({})

    useQueueTasksMock.mockReturnValue({
      data: {
        tasks: [{ id: 'task-1', title: 'Task', status: 'ready', priority: 3, depends_on: [] }],
        queue_title: 'Tasks',
      },
      isLoading: false,
      error: null,
    })

    render(<TasksPanel queueId="queue-1" onEditTask={mockOnEditTask} />)

    fireEvent.click(screen.getByRole('button', { name: /start implementation/i }))

    await waitFor(() => {
      expect(startImplMutateAsyncMock).toHaveBeenCalledWith({ queue_id: 'queue-1' })
    })
  })

  it('renders hierarchical tasks', () => {
    useQueueTasksMock.mockReturnValue({
      data: {
        tasks: [
          { id: 'parent-1', title: 'Parent Task', status: 'ready', priority: 3, depends_on: [] },
          { id: 'child-1', title: 'Child Task', status: 'pending', priority: 3, depends_on: [], parent_id: 'parent-1' },
        ],
        queue_title: 'Tasks',
      },
      isLoading: false,
      error: null,
    })

    render(<TasksPanel queueId="queue-1" onEditTask={mockOnEditTask} />)

    expect(screen.getByText('Parent Task')).toBeInTheDocument()
    expect(screen.getByText('Child Task')).toBeInTheDocument()
  })

  it('shows 1 task singular', () => {
    useQueueTasksMock.mockReturnValue({
      data: {
        tasks: [{ id: 'task-1', title: 'Only Task', status: 'ready', priority: 3, depends_on: [] }],
        queue_title: 'Tasks',
      },
      isLoading: false,
      error: null,
    })

    render(<TasksPanel queueId="queue-1" onEditTask={mockOnEditTask} />)

    expect(screen.getByText('1 task')).toBeInTheDocument()
  })
})
