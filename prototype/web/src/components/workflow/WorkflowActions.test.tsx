import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@/test/test-utils'
import { WorkflowActions } from './WorkflowActions'

// Mock hooks
const mutateWorkflowMock = vi.fn()
const useWorkflowActionMock = vi.fn()
const navigateMock = vi.fn()

vi.mock('@/api/workflow', () => ({
  useWorkflowAction: () => useWorkflowActionMock(),
}))

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => navigateMock,
  }
})

// Mock window.confirm
const confirmMock = vi.fn()
window.confirm = confirmMock

describe('WorkflowActions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useWorkflowActionMock.mockReturnValue({
      mutate: mutateWorkflowMock,
      isPending: false,
    })
    confirmMock.mockReturnValue(true)
  })

  it('renders actions card', () => {
    render(<WorkflowActions hasTask={true} />)

    expect(screen.getByText('Actions')).toBeInTheDocument()
  })

  it('renders primary action buttons', () => {
    render(<WorkflowActions hasTask={true} state="idle" />)

    expect(screen.getByRole('button', { name: /plan/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /implement/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /review/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /finish/i })).toBeInTheDocument()
  })

  it('disables actions when no task', () => {
    render(<WorkflowActions hasTask={false} state="idle" />)

    expect(screen.getByRole('button', { name: /plan/i })).toBeDisabled()
    expect(screen.getByRole('button', { name: /implement/i })).toBeDisabled()
    expect(screen.getByRole('button', { name: /review/i })).toBeDisabled()
    expect(screen.getByRole('button', { name: /finish/i })).toBeDisabled()
  })

  it('disables actions when state is active (ending in "ing")', () => {
    render(<WorkflowActions hasTask={true} state="planning" />)

    expect(screen.getByRole('button', { name: /plan/i })).toBeDisabled()
    expect(screen.getByRole('button', { name: /implement/i })).toBeDisabled()
  })

  it('calls plan action when Plan button clicked', async () => {
    render(<WorkflowActions hasTask={true} state="idle" />)

    fireEvent.click(screen.getByRole('button', { name: /plan/i }))

    expect(mutateWorkflowMock).toHaveBeenCalledWith(
      { action: 'plan', options: undefined, implementOptions: undefined },
      expect.any(Object)
    )
  })

  it('calls implement action when Implement button clicked', async () => {
    render(<WorkflowActions hasTask={true} state="idle" progressPhase="planned" />)

    fireEvent.click(screen.getByRole('button', { name: /implement/i }))

    expect(mutateWorkflowMock).toHaveBeenCalledWith(
      { action: 'implement', options: undefined, implementOptions: undefined },
      expect.any(Object)
    )
  })

  it('shows advanced actions when toggled', () => {
    render(<WorkflowActions hasTask={true} state="idle" progressPhase="implemented" />)

    // Click to show advanced actions
    fireEvent.click(screen.getByRole('button', { name: /advanced actions/i }))

    expect(screen.getByRole('button', { name: /sync/i })).toBeInTheDocument()
  })

  it('shows undo/redo/abandon/reset in advanced mode', () => {
    render(<WorkflowActions hasTask={true} state="idle" progressPhase="implemented" />)

    fireEvent.click(screen.getByRole('button', { name: /advanced actions/i }))

    expect(screen.getByRole('button', { name: 'Undo' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Redo' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Abandon' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Reset' })).toBeInTheDocument()
  })

  it('shows confirmation for abandon action', () => {
    render(<WorkflowActions hasTask={true} state="idle" />)

    fireEvent.click(screen.getByRole('button', { name: /advanced actions/i }))
    fireEvent.click(screen.getByRole('button', { name: 'Abandon' }))

    expect(confirmMock).toHaveBeenCalled()
  })

  it('does not call abandon when user cancels confirmation', () => {
    confirmMock.mockReturnValue(false)

    render(<WorkflowActions hasTask={true} state="idle" />)

    fireEvent.click(screen.getByRole('button', { name: /advanced actions/i }))
    fireEvent.click(screen.getByRole('button', { name: 'Abandon' }))

    expect(mutateWorkflowMock).not.toHaveBeenCalled()
  })

  it('navigates home after abandon action', async () => {
    render(<WorkflowActions hasTask={true} state="idle" />)

    fireEvent.click(screen.getByRole('button', { name: /advanced actions/i }))
    fireEvent.click(screen.getByRole('button', { name: 'Abandon' }))

    // Get the onSuccess callback
    const onSuccessCallback = mutateWorkflowMock.mock.calls[0][1].onSuccess
    onSuccessCallback({})

    expect(navigateMock).toHaveBeenCalledWith('/')
  })

  it('navigates home after finish action', async () => {
    render(<WorkflowActions hasTask={true} state="idle" progressPhase="implemented" />)

    fireEvent.click(screen.getByRole('button', { name: /finish/i }))

    const onSuccessCallback = mutateWorkflowMock.mock.calls[0][1].onSuccess
    onSuccessCallback({})

    expect(navigateMock).toHaveBeenCalledWith('/')
  })

  it('shows implementation options panel when toggled', () => {
    render(
      <WorkflowActions
        hasTask={true}
        state="idle"
        progressPhase="planned"
        specs={[{ component: 'frontend' }, { component: 'backend' }]}
      />
    )

    // First expand advanced actions
    fireEvent.click(screen.getByRole('button', { name: /advanced actions/i }))

    // Then click the implement options chevron
    fireEvent.click(screen.getByRole('button', { name: /implementation options/i }))

    expect(screen.getByLabelText('Component')).toBeInTheDocument()
    expect(screen.getByLabelText('Parallel workers')).toBeInTheDocument()
  })

  it('calls implement with options', async () => {
    render(
      <WorkflowActions
        hasTask={true}
        state="idle"
        progressPhase="planned"
        specs={[{ component: 'frontend' }, { component: 'backend' }]}
      />
    )

    // Show advanced actions and options
    fireEvent.click(screen.getByRole('button', { name: /advanced actions/i }))
    fireEvent.click(screen.getByRole('button', { name: /implementation options/i }))

    // Select component
    const componentSelect = screen.getByLabelText('Component')
    fireEvent.change(componentSelect, { target: { value: 'frontend' } })

    // Set parallel workers
    const parallelInput = screen.getByLabelText('Parallel workers')
    fireEvent.change(parallelInput, { target: { value: '3' } })

    // Click implement with options
    fireEvent.click(screen.getByRole('button', { name: /implement with options/i }))

    expect(mutateWorkflowMock).toHaveBeenCalledWith({
      action: 'implement',
      implementOptions: { component: 'frontend', parallel: 3 },
    })
  })

  it('disables implement when phase is started', () => {
    render(<WorkflowActions hasTask={true} state="idle" progressPhase="started" />)

    expect(screen.getByRole('button', { name: /implement/i })).toBeDisabled()
  })

  it('disables review when phase is started', () => {
    render(<WorkflowActions hasTask={true} state="idle" progressPhase="started" />)

    expect(screen.getByRole('button', { name: /review/i })).toBeDisabled()
  })

  it('disables finish when phase is planned', () => {
    render(<WorkflowActions hasTask={true} state="idle" progressPhase="planned" />)

    expect(screen.getByRole('button', { name: /finish/i })).toBeDisabled()
  })

  it('shows sync result when sync succeeds', async () => {
    render(<WorkflowActions hasTask={true} state="idle" taskId="task-123" progressPhase="implemented" />)

    fireEvent.click(screen.getByRole('button', { name: /advanced actions/i }))
    fireEvent.click(screen.getByRole('button', { name: /sync/i }))

    // Simulate success callback with sync result
    const onSuccessCallback = mutateWorkflowMock.mock.calls[0][1].onSuccess
    onSuccessCallback({
      message: 'Sync completed',
      has_changes: true,
      spec_generated: 'delta-spec-001',
      changes_summary: 'Updated 3 files',
    })

    await waitFor(() => {
      expect(screen.getByText('Sync completed')).toBeInTheDocument()
      expect(screen.getByText(/delta-spec-001/)).toBeInTheDocument()
    })
  })

  it('disables all buttons when mutation is pending', () => {
    useWorkflowActionMock.mockReturnValue({
      mutate: mutateWorkflowMock,
      isPending: true,
    })

    render(<WorkflowActions hasTask={true} state="idle" />)

    expect(screen.getByRole('button', { name: /plan/i })).toBeDisabled()
    expect(screen.getByRole('button', { name: /implement/i })).toBeDisabled()
    expect(screen.getByRole('button', { name: /review/i })).toBeDisabled()
    expect(screen.getByRole('button', { name: /finish/i })).toBeDisabled()
  })

  it('resets implement options when advanced actions toggled off', () => {
    render(
      <WorkflowActions
        hasTask={true}
        state="idle"
        progressPhase="planned"
        specs={[{ component: 'frontend' }]}
      />
    )

    // Show advanced actions and options
    fireEvent.click(screen.getByRole('button', { name: /advanced actions/i }))
    fireEvent.click(screen.getByRole('button', { name: /implementation options/i }))

    // Verify options panel is visible
    expect(screen.getByLabelText('Component')).toBeInTheDocument()

    // Toggle off advanced actions
    fireEvent.click(screen.getByRole('button', { name: /advanced actions/i }))

    // Verify options panel is hidden
    expect(screen.queryByLabelText('Component')).not.toBeInTheDocument()
  })
})
