import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen } from '@/test/test-utils'
import TaskDetail from './TaskDetail'

// Mock all hooks
const useActiveTaskMock = vi.fn()
const useTaskWorkMock = vi.fn()
const useTaskSpecsMock = vi.fn()
const useTaskNotesMock = vi.fn()
const useAgentLogsHistoryMock = vi.fn()
const useWorkflowSSEMock = vi.fn()

vi.mock('@/api/workflow', () => ({
  useActiveTask: () => useActiveTaskMock(),
}))

vi.mock('@/api/task', () => ({
  useTaskWork: () => useTaskWorkMock(),
  useTaskSpecs: () => useTaskSpecsMock(),
  useTaskNotes: () => useTaskNotesMock(),
  useAgentLogsHistory: () => useAgentLogsHistoryMock(),
}))

vi.mock('@/hooks/useWorkflowSSE', () => ({
  useWorkflowSSE: (options: { onAgentMessage?: (msg: unknown) => void }) => {
    useWorkflowSSEMock(options)
    return { connected: true }
  },
}))

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useParams: () => ({ id: 'task-123' }),
  }
})

// Mock child components to simplify testing
vi.mock('@/components/task/ActiveWorkCard', () => ({
  ActiveWorkCard: ({ task }: { task: { id: string } }) => (
    <div data-testid="active-work-card">Active: {task.id}</div>
  ),
}))

vi.mock('@/components/task/CompletedWorkCard', () => ({
  CompletedWorkCard: ({ work }: { work: { metadata: { title: string } } }) => (
    <div data-testid="completed-work-card">Completed: {work.metadata.title}</div>
  ),
}))

vi.mock('@/components/workflow/WorkflowActions', () => ({
  WorkflowActions: () => <div data-testid="workflow-actions">Actions</div>,
}))

vi.mock('@/components/workflow/QuestionPrompt', () => ({
  QuestionPrompt: () => <div data-testid="question-prompt">Question</div>,
}))

vi.mock('@/components/task/SpecificationsList', () => ({
  SpecificationsList: () => <div data-testid="specs-list">Specs</div>,
}))

vi.mock('@/components/task/ReviewsList', () => ({
  ReviewsList: () => <div data-testid="reviews-list">Reviews</div>,
}))

vi.mock('@/components/task/NotesCard', () => ({
  NotesCard: () => <div data-testid="notes-card">Notes</div>,
}))

vi.mock('@/components/task/LabelsCard', () => ({
  LabelsCard: () => <div data-testid="labels-card">Labels</div>,
}))

vi.mock('@/components/task/AgentTerminal', () => ({
  AgentTerminal: () => <div data-testid="agent-terminal">Terminal</div>,
}))

vi.mock('@/components/task/QuickQuestion', () => ({
  QuickQuestion: () => <div data-testid="quick-question">Quick Q</div>,
}))

vi.mock('@/components/task/CostsCard', () => ({
  CostsCard: () => <div data-testid="costs-card">Costs</div>,
}))

describe('TaskDetail page', () => {
  beforeEach(() => {
    vi.clearAllMocks()

    // Default mock implementations
    useActiveTaskMock.mockReturnValue({
      data: null,
      isLoading: false,
    })
    useTaskWorkMock.mockReturnValue({
      data: null,
      isLoading: false,
    })
    useTaskSpecsMock.mockReturnValue({
      data: null,
      isLoading: false,
    })
    useTaskNotesMock.mockReturnValue({
      data: null,
    })
    useAgentLogsHistoryMock.mockReturnValue({
      data: null,
    })
  })

  it('shows task not found when no task data exists', () => {
    render(<TaskDetail />)

    expect(screen.getByText('Task Not Found')).toBeInTheDocument()
    expect(screen.getByText(/This task may have been completed/)).toBeInTheDocument()
  })

  it('shows back to dashboard link when task not found', () => {
    render(<TaskDetail />)

    expect(screen.getByRole('link', { name: /back to dashboard/i })).toBeInTheDocument()
  })

  it('shows loading skeleton while loading', () => {
    useActiveTaskMock.mockReturnValue({
      data: null,
      isLoading: true,
    })
    useTaskWorkMock.mockReturnValue({
      data: null,
      isLoading: true,
    })

    render(<TaskDetail />)

    // Should show loading skeleton, not "not found"
    expect(screen.queryByText('Task Not Found')).not.toBeInTheDocument()
  })

  it('shows active work card for active tasks', () => {
    useActiveTaskMock.mockReturnValue({
      data: {
        active: true,
        task: { id: 'task-123', state: 'implementing' },
        work: { title: 'Fix bug' },
      },
      isLoading: false,
    })

    render(<TaskDetail />)

    expect(screen.getByTestId('active-work-card')).toBeInTheDocument()
    expect(screen.getByText('Active: task-123')).toBeInTheDocument()
  })

  it('shows completed work card for completed tasks', () => {
    useActiveTaskMock.mockReturnValue({
      data: { active: false, task: null },
      isLoading: false,
    })
    useTaskWorkMock.mockReturnValue({
      data: {
        work: {
          metadata: { id: 'task-123', title: 'Completed task', state: 'done' },
        },
      },
      isLoading: false,
    })

    render(<TaskDetail />)

    expect(screen.getByTestId('completed-work-card')).toBeInTheDocument()
    expect(screen.getByText('Completed: Completed task')).toBeInTheDocument()
  })

  it('shows question prompt when pending question exists', () => {
    useActiveTaskMock.mockReturnValue({
      data: {
        active: true,
        task: { id: 'task-123', state: 'waiting' },
        pending_question: { id: 'q-1', text: 'What API?' },
      },
      isLoading: false,
    })

    render(<TaskDetail />)

    expect(screen.getByTestId('question-prompt')).toBeInTheDocument()
  })

  it('renders all main components for active task', () => {
    useActiveTaskMock.mockReturnValue({
      data: {
        active: true,
        task: { id: 'task-123', state: 'implementing' },
        work: { title: 'Fix bug' },
      },
      isLoading: false,
    })

    render(<TaskDetail />)

    expect(screen.getByTestId('active-work-card')).toBeInTheDocument()
    expect(screen.getByTestId('quick-question')).toBeInTheDocument()
    expect(screen.getByTestId('specs-list')).toBeInTheDocument()
    expect(screen.getByTestId('reviews-list')).toBeInTheDocument()
    expect(screen.getByTestId('notes-card')).toBeInTheDocument()
    expect(screen.getByTestId('agent-terminal')).toBeInTheDocument()
    expect(screen.getByTestId('workflow-actions')).toBeInTheDocument()
    expect(screen.getByTestId('labels-card')).toBeInTheDocument()
    expect(screen.getByTestId('costs-card')).toBeInTheDocument()
  })

  it('shows connected status indicator', () => {
    useActiveTaskMock.mockReturnValue({
      data: {
        active: true,
        task: { id: 'task-123', state: 'implementing' },
      },
      isLoading: false,
    })

    render(<TaskDetail />)

    expect(screen.getByText('Connected')).toBeInTheDocument()
  })

  it('shows dashboard link in header', () => {
    useActiveTaskMock.mockReturnValue({
      data: {
        active: true,
        task: { id: 'task-123', state: 'implementing' },
      },
      isLoading: false,
    })

    render(<TaskDetail />)

    expect(screen.getByRole('link', { name: /dashboard/i })).toHaveAttribute('href', '/')
  })
})
