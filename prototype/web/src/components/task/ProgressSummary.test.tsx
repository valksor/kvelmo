import { describe, expect, it } from 'vitest'
import { render, screen } from '@/test/test-utils'
import { ProgressSummary } from './ProgressSummary'
import type { TerminalMessage } from './AgentTerminal'

describe('ProgressSummary', () => {
  const createMessage = (content: string, id: number, type = 'output'): TerminalMessage => ({
    content,
    timestamp: new Date().toISOString(),
    type,
    _id: id,
  })

  it('shows waiting message when no messages', () => {
    render(<ProgressSummary messages={[]} />)

    expect(screen.getByText('Waiting for activity...')).toBeInTheDocument()
  })

  it('shows activity counts for messages', () => {
    const messages: TerminalMessage[] = [
      createMessage('Some info message', 1),
      createMessage('Task completed successfully', 2),
      createMessage('Warning: something happened', 3),
    ]

    render(<ProgressSummary messages={messages} />)

    expect(screen.getByText('Activity:')).toBeInTheDocument()
  })

  it('categorizes success messages correctly', () => {
    const messages: TerminalMessage[] = [
      createMessage('Task completed ✓', 1),
      createMessage('Another success passed', 2),
    ]

    render(<ProgressSummary messages={messages} />)

    // Should show success count
    const successCount = screen.getByTitle('Completed items')
    expect(successCount).toHaveTextContent('2')
  })

  it('categorizes error messages correctly', () => {
    const messages: TerminalMessage[] = [
      createMessage('Error: something failed', 1, 'error'),
      createMessage('Another ❌ failure', 2),
    ]

    render(<ProgressSummary messages={messages} />)

    // Should show error count
    const errorCount = screen.getByTitle('Errors')
    expect(errorCount).toHaveTextContent('2')
  })

  it('categorizes warning messages correctly', () => {
    const messages: TerminalMessage[] = [
      createMessage('Warning: check this', 1),
      createMessage('Caution advised ⚠', 2),
    ]

    render(<ProgressSummary messages={messages} />)

    // Should show warning count
    const warningCount = screen.getByTitle('Warnings')
    expect(warningCount).toHaveTextContent('2')
  })

  it('shows recent messages preview', () => {
    const messages: TerminalMessage[] = [
      createMessage('First message', 1),
      createMessage('Second message', 2),
      createMessage('Third message', 3),
      createMessage('Fourth message', 4),
    ]

    render(<ProgressSummary messages={messages} />)

    expect(screen.getByText('Recent')).toBeInTheDocument()
    // Shows last 3 messages in reverse order
    expect(screen.getByText('Fourth message')).toBeInTheDocument()
    expect(screen.getByText('Third message')).toBeInTheDocument()
    expect(screen.getByText('Second message')).toBeInTheDocument()
  })

  it('truncates long messages', () => {
    const longContent = 'A'.repeat(150)
    const messages: TerminalMessage[] = [createMessage(longContent, 1)]

    render(<ProgressSummary messages={messages} />)

    // Should show truncated message with ellipsis
    expect(screen.getByText(/^A+\.\.\.$/)).toBeInTheDocument()
  })

  it('shows progress bar when specs are detected', () => {
    const messages: TerminalMessage[] = [
      createMessage('Working on spec 1', 1),
      createMessage('spec 1 completed', 2),
      createMessage('Working on spec 2', 3),
    ]

    render(<ProgressSummary messages={messages} />)

    // Should show progress info
    expect(screen.getByText(/steps complete/)).toBeInTheDocument()
    expect(screen.getByRole('progressbar', { name: 'Task progress' })).toBeInTheDocument()
  })

  it('shows current action when in planning state', () => {
    const messages: TerminalMessage[] = [
      createMessage('Planning the implementation', 1),
    ]

    render(<ProgressSummary messages={messages} workflowState="planning" />)

    expect(screen.getByText('Analyzing task and creating plan...')).toBeInTheDocument()
  })

  it('shows current action when implementing', () => {
    const messages: TerminalMessage[] = [
      createMessage('Implementing the changes', 1),
    ]

    render(<ProgressSummary messages={messages} workflowState="implementing" />)

    expect(screen.getByText('Writing code changes...')).toBeInTheDocument()
  })

  it('shows current action when reviewing', () => {
    const messages: TerminalMessage[] = [
      createMessage('Reviewing the code', 1),
    ]

    render(<ProgressSummary messages={messages} workflowState="reviewing" />)

    expect(screen.getByText('Checking code quality...')).toBeInTheDocument()
  })

  it('clears current action when workflow is idle', () => {
    const messages: TerminalMessage[] = [
      createMessage('Planning the implementation', 1),
    ]

    render(<ProgressSummary messages={messages} workflowState="idle" />)

    // Should not show the current action since workflow is idle
    expect(screen.queryByText('Analyzing task and creating plan...')).not.toBeInTheDocument()
  })

  it('shows percentage when progress is available', () => {
    const messages: TerminalMessage[] = [
      createMessage('spec 1 completed', 1),
      createMessage('Working on spec 2', 2),
    ]

    render(<ProgressSummary messages={messages} />)

    // Should show percentage
    expect(screen.getByText(/\d+%/)).toBeInTheDocument()
  })
})
