import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@/test/test-utils'
import Quick from './Quick'

// Mock hooks
const useStatusMock = vi.fn()
const useQuickTasksMock = vi.fn()
const createMutateAsyncMock = vi.fn()
const optimizeMutateAsyncMock = vi.fn()
const startMutateAsyncMock = vi.fn()
const deleteMutateAsyncMock = vi.fn()
const submitMutateAsyncMock = vi.fn()
const addNoteMutateAsyncMock = vi.fn()
const exportMutateAsyncMock = vi.fn()
const submitSourceMutateAsyncMock = vi.fn()

vi.mock('@/api/workflow', () => ({
  useStatus: () => useStatusMock(),
}))

vi.mock('@/api/quick', () => ({
  useQuickTasks: () => useQuickTasksMock(),
  useCreateQuickTask: () => ({
    mutateAsync: createMutateAsyncMock,
    isPending: false,
    isError: false,
    error: null,
  }),
  useOptimizeQuickTask: () => ({
    mutateAsync: optimizeMutateAsyncMock,
    isPending: false,
    isError: false,
    error: null,
  }),
  useStartQuickTask: () => ({
    mutateAsync: startMutateAsyncMock,
    isPending: false,
    isError: false,
    error: null,
  }),
  useDeleteQuickTask: () => ({
    mutateAsync: deleteMutateAsyncMock,
    isPending: false,
    isError: false,
    error: null,
  }),
  useSubmitQuickTask: () => ({
    mutateAsync: submitMutateAsyncMock,
    isPending: false,
    isSuccess: false,
    isError: false,
    data: null,
    error: null,
  }),
  useAddQuickTaskNote: () => ({
    mutateAsync: addNoteMutateAsyncMock,
    isPending: false,
    isError: false,
    error: null,
  }),
  useExportQuickTask: () => ({
    mutateAsync: exportMutateAsyncMock,
    isPending: false,
    isError: false,
    error: null,
  }),
  useSubmitSource: () => ({
    mutateAsync: submitSourceMutateAsyncMock,
    isPending: false,
    isSuccess: false,
    isError: false,
    data: null,
    error: null,
  }),
}))

vi.mock('@/components/project/ProjectSelector', () => ({
  ProjectSelector: () => <div data-testid="project-selector">Project Selector</div>,
}))

// Mock window.confirm
const confirmMock = vi.fn()
window.confirm = confirmMock

describe('Quick page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useStatusMock.mockReturnValue({
      data: { mode: 'project' },
      isLoading: false,
    })
    useQuickTasksMock.mockReturnValue({
      data: { tasks: [] },
      isLoading: false,
      error: null,
    })
    confirmMock.mockReturnValue(true)
  })

  it('renders page title and description', () => {
    render(<Quick />)

    expect(screen.getByRole('heading', { name: 'Quick Tasks' })).toBeInTheDocument()
    expect(screen.getByText(/Capture ideas quickly/)).toBeInTheDocument()
  })

  it('shows loading spinner while status is loading', () => {
    useStatusMock.mockReturnValue({
      data: null,
      isLoading: true,
    })

    render(<Quick />)

    expect(screen.queryByRole('heading', { name: 'Quick Tasks' })).not.toBeInTheDocument()
  })

  it('shows project selector in global mode', () => {
    useStatusMock.mockReturnValue({
      data: { mode: 'global' },
      isLoading: false,
    })

    render(<Quick />)

    expect(screen.getByTestId('project-selector')).toBeInTheDocument()
  })

  it('shows empty state when no tasks', () => {
    render(<Quick />)

    expect(screen.getByText('No quick tasks yet')).toBeInTheDocument()
  })

  it('shows task list when tasks exist', () => {
    useQuickTasksMock.mockReturnValue({
      data: {
        tasks: [
          { id: 'task-1', title: 'Fix bug', status: 'pending', priority: 1, labels: [], note_count: 0 },
          { id: 'task-2', title: 'Add feature', status: 'ready', priority: 2, labels: ['frontend'], note_count: 2 },
        ],
      },
      isLoading: false,
      error: null,
    })

    render(<Quick />)

    expect(screen.getByText('Fix bug')).toBeInTheDocument()
    expect(screen.getByText('Add feature')).toBeInTheDocument()
    expect(screen.getByText('Quick Tasks (2)')).toBeInTheDocument()
  })

  it('shows priority badges correctly', () => {
    useQuickTasksMock.mockReturnValue({
      data: {
        tasks: [
          { id: 'task-1', title: 'Low priority', status: 'pending', priority: 0, labels: [], note_count: 0 },
          { id: 'task-2', title: 'Normal priority', status: 'pending', priority: 1, labels: [], note_count: 0 },
          { id: 'task-3', title: 'High priority', status: 'pending', priority: 2, labels: [], note_count: 0 },
        ],
      },
      isLoading: false,
      error: null,
    })

    render(<Quick />)

    // Priority badges appear as badge elements (also in dropdown, so use getAllByText)
    expect(screen.getAllByText('Low').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Normal').length).toBeGreaterThan(0)
    expect(screen.getAllByText('High').length).toBeGreaterThan(0)
  })

  it('shows labels for tasks', () => {
    useQuickTasksMock.mockReturnValue({
      data: {
        tasks: [
          { id: 'task-1', title: 'Task', status: 'pending', priority: 1, labels: ['bug', 'urgent'], note_count: 0 },
        ],
      },
      isLoading: false,
      error: null,
    })

    render(<Quick />)

    expect(screen.getByText('bug')).toBeInTheDocument()
    expect(screen.getByText('urgent')).toBeInTheDocument()
  })

  it('shows error when tasks fail to load', () => {
    useQuickTasksMock.mockReturnValue({
      data: null,
      isLoading: false,
      error: new Error('Network error'),
    })

    render(<Quick />)

    expect(screen.getByText('Network error')).toBeInTheDocument()
  })

  it('creates a new task on form submit', async () => {
    createMutateAsyncMock.mockResolvedValue({ id: 'new-task' })

    render(<Quick />)

    const descInput = screen.getByPlaceholderText('What needs to be done?')
    fireEvent.change(descInput, { target: { value: 'New task description' } })

    const createButton = screen.getByRole('button', { name: /create task/i })
    fireEvent.click(createButton)

    await waitFor(() => {
      expect(createMutateAsyncMock).toHaveBeenCalledWith({
        title: undefined,
        description: 'New task description',
        priority: 1,
        labels: [],
      })
    })
  })

  it('creates task with title and labels', async () => {
    createMutateAsyncMock.mockResolvedValue({ id: 'new-task' })

    render(<Quick />)

    const titleInput = screen.getByPlaceholderText('Brief summary')
    fireEvent.change(titleInput, { target: { value: 'My Task' } })

    const descInput = screen.getByPlaceholderText('What needs to be done?')
    fireEvent.change(descInput, { target: { value: 'Task description' } })

    const labelsInput = screen.getByPlaceholderText('bug, frontend, urgent')
    fireEvent.change(labelsInput, { target: { value: 'bug, frontend' } })

    const createButton = screen.getByRole('button', { name: /create task/i })
    fireEvent.click(createButton)

    await waitFor(() => {
      expect(createMutateAsyncMock).toHaveBeenCalledWith({
        title: 'My Task',
        description: 'Task description',
        priority: 1,
        labels: ['bug', 'frontend'],
      })
    })
  })

  it('expands source import form when clicked', () => {
    render(<Quick />)

    const importButton = screen.getByText('Import from External Source')
    fireEvent.click(importButton)

    expect(screen.getByPlaceholderText('https://github.com/org/repo/issues/123')).toBeInTheDocument()
  })

  it('submits source import form', async () => {
    submitSourceMutateAsyncMock.mockResolvedValue({ external_url: 'https://github.com/test' })

    render(<Quick />)

    // Expand form
    fireEvent.click(screen.getByText('Import from External Source'))

    // Fill form
    const refInput = screen.getByPlaceholderText('https://github.com/org/repo/issues/123')
    fireEvent.change(refInput, { target: { value: 'https://github.com/org/repo/issues/42' } })

    // Submit
    const submitButton = screen.getByRole('button', { name: /import & submit/i })
    fireEvent.click(submitButton)

    await waitFor(() => {
      expect(submitSourceMutateAsyncMock).toHaveBeenCalledWith({
        source: 'https://github.com/org/repo/issues/42',
        provider: 'github',
        notes: undefined,
        optimize: true,
      })
    })
  })

  it('toggles notes section for a task', () => {
    useQuickTasksMock.mockReturnValue({
      data: {
        tasks: [
          { id: 'task-1', title: 'Task', status: 'pending', priority: 1, labels: [], note_count: 3 },
        ],
      },
      isLoading: false,
      error: null,
    })

    render(<Quick />)

    const notesButton = screen.getByText('Notes (3)')
    fireEvent.click(notesButton)

    expect(screen.getByPlaceholderText('Add a note...')).toBeInTheDocument()
  })

  it('adds a note to a task', async () => {
    addNoteMutateAsyncMock.mockResolvedValue({})

    useQuickTasksMock.mockReturnValue({
      data: {
        tasks: [
          { id: 'task-1', title: 'Task', status: 'pending', priority: 1, labels: [], note_count: 0 },
        ],
      },
      isLoading: false,
      error: null,
    })

    render(<Quick />)

    // Expand notes
    fireEvent.click(screen.getByText('Notes (0)'))

    // Add note
    const noteInput = screen.getByPlaceholderText('Add a note...')
    fireEvent.change(noteInput, { target: { value: 'My note' } })

    const addButton = screen.getByRole('button', { name: 'Add' })
    fireEvent.click(addButton)

    await waitFor(() => {
      expect(addNoteMutateAsyncMock).toHaveBeenCalledWith({ taskId: 'task-1', note: 'My note' })
    })
  })

  it('opens submit modal when submit button clicked', () => {
    useQuickTasksMock.mockReturnValue({
      data: {
        tasks: [
          { id: 'task-1', title: 'Task', status: 'pending', priority: 1, labels: [], note_count: 0 },
        ],
      },
      isLoading: false,
      error: null,
    })

    render(<Quick />)

    // Find submit button by its title
    const submitButton = screen.getByTitle('Submit to provider')
    fireEvent.click(submitButton)

    expect(screen.getByText('Submit this task to an external provider.')).toBeInTheDocument()
  })

  it('opens delete confirmation when delete button clicked', () => {
    useQuickTasksMock.mockReturnValue({
      data: {
        tasks: [
          { id: 'task-1', title: 'Task', status: 'pending', priority: 1, labels: [], note_count: 0 },
        ],
      },
      isLoading: false,
      error: null,
    })

    render(<Quick />)

    const deleteButton = screen.getByRole('button', { name: 'Delete task' })
    fireEvent.click(deleteButton)

    expect(screen.getByText('This removes the quick task permanently.')).toBeInTheDocument()
  })

  it('calls delete mutation when confirmed', async () => {
    deleteMutateAsyncMock.mockResolvedValue({})

    useQuickTasksMock.mockReturnValue({
      data: {
        tasks: [
          { id: 'task-1', title: 'Task', status: 'pending', priority: 1, labels: [], note_count: 0 },
        ],
      },
      isLoading: false,
      error: null,
    })

    render(<Quick />)

    // Open delete modal
    fireEvent.click(screen.getByRole('button', { name: 'Delete task' }))

    // Confirm delete
    const confirmButton = screen.getByRole('button', { name: 'Delete' })
    fireEvent.click(confirmButton)

    await waitFor(() => {
      expect(deleteMutateAsyncMock).toHaveBeenCalledWith('task-1')
    })
  })

  it('calls optimize mutation when optimize button clicked', async () => {
    optimizeMutateAsyncMock.mockResolvedValue({})

    useQuickTasksMock.mockReturnValue({
      data: {
        tasks: [
          { id: 'task-1', title: 'Task', status: 'pending', priority: 1, labels: [], note_count: 0 },
        ],
      },
      isLoading: false,
      error: null,
    })

    render(<Quick />)

    const optimizeButton = screen.getByTitle('Optimize with AI')
    fireEvent.click(optimizeButton)

    await waitFor(() => {
      expect(optimizeMutateAsyncMock).toHaveBeenCalledWith({ taskId: 'task-1' })
    })
  })

  it('calls start mutation when start button clicked', async () => {
    startMutateAsyncMock.mockResolvedValue({})

    useQuickTasksMock.mockReturnValue({
      data: {
        tasks: [
          { id: 'task-1', title: 'Task', status: 'pending', priority: 1, labels: [], note_count: 0 },
        ],
      },
      isLoading: false,
      error: null,
    })

    render(<Quick />)

    const startButton = screen.getByTitle('Start working')
    fireEvent.click(startButton)

    await waitFor(() => {
      expect(startMutateAsyncMock).toHaveBeenCalledWith('task-1')
    })
  })

  it('changes priority in create form', async () => {
    createMutateAsyncMock.mockResolvedValue({ id: 'new-task' })

    render(<Quick />)

    // Change priority to High
    const prioritySelect = screen.getByRole('combobox')
    fireEvent.change(prioritySelect, { target: { value: '2' } })

    // Fill description
    const descInput = screen.getByPlaceholderText('What needs to be done?')
    fireEvent.change(descInput, { target: { value: 'High priority task' } })

    // Submit
    fireEvent.click(screen.getByRole('button', { name: /create task/i }))

    await waitFor(() => {
      expect(createMutateAsyncMock).toHaveBeenCalledWith(
        expect.objectContaining({ priority: 2 })
      )
    })
  })
})
