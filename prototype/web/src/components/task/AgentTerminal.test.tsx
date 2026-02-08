import { describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test/test-utils'
import { AgentTerminal, type TerminalMessage } from './AgentTerminal'

function makeMessages(count: number): TerminalMessage[] {
  return Array.from({ length: count }, (_, i) => ({
    content: `Message ${i}`,
    timestamp: `2026-01-01T10:${String(i % 60).padStart(2, '0')}:00Z`,
    type: 'output' as const,
    _id: i + 1,
  }))
}

// Helper to switch to Details view
function switchToDetails() {
  const detailsButton = screen.getByRole('button', { name: /Details/i })
  fireEvent.click(detailsButton)
}

describe('AgentTerminal', () => {
  it('renders empty state in Summary view when no messages', () => {
    render(<AgentTerminal messages={[]} onClear={vi.fn()} />)
    // Summary view shows "Waiting for activity..."
    expect(screen.getByText('Waiting for activity...')).toBeInTheDocument()
  })

  it('renders empty state in Details view when no messages', () => {
    render(<AgentTerminal messages={[]} onClear={vi.fn()} />)
    switchToDetails()
    expect(screen.getByText('No updates yet...')).toBeInTheDocument()
  })

  it('renders messages in reverse chronological order in Details view', () => {
    const messages: TerminalMessage[] = [
      { content: 'First', timestamp: '2026-01-01T10:00:00Z', type: 'output', _id: 1 },
      { content: 'Second', timestamp: '2026-01-01T10:01:00Z', type: 'output', _id: 2 },
    ]
    render(<AgentTerminal messages={messages} onClear={vi.fn()} />)
    switchToDetails()
    const items = screen.getAllByText(/First|Second/)
    expect(items[0]).toHaveTextContent('Second')
    expect(items[1]).toHaveTextContent('First')
  })

  it('caps displayed messages at 500 in Details view', () => {
    const messages = makeMessages(600)
    const { container } = render(<AgentTerminal messages={messages} onClear={vi.fn()} />)
    switchToDetails()
    const lines = container.querySelectorAll('[class*="py-0.5"]')
    expect(lines.length).toBe(500)
  })

  it('formats timestamp as HH:MM:SS from ISO string in Details view', () => {
    const messages: TerminalMessage[] = [
      { content: 'Test', timestamp: '2026-03-15T14:30:05.123Z', type: 'output', _id: 1 },
    ]
    render(<AgentTerminal messages={messages} onClear={vi.fn()} />)
    switchToDetails()
    expect(screen.getByText('14:30:05')).toBeInTheDocument()
  })

  it('applies error styling for error messages in Details view', () => {
    const messages: TerminalMessage[] = [
      { content: 'Oh no', timestamp: '2026-01-01T10:00:00Z', type: 'error', _id: 1 },
    ]
    render(<AgentTerminal messages={messages} onClear={vi.fn()} />)
    switchToDetails()
    const errorEl = screen.getByText('Oh no')
    expect(errorEl.className).toContain('text-error')
  })

  it('shows total message count in header badge', () => {
    const messages = makeMessages(600)
    render(<AgentTerminal messages={messages} onClear={vi.fn()} />)
    // The badge in the header shows the total count - find it by class
    const badges = screen.getAllByText('600')
    const headerBadge = badges.find((el) => el.className.includes('badge'))
    expect(headerBadge).toBeDefined()
    expect(headerBadge?.className).toContain('badge')
  })

  it('defaults to Summary view', () => {
    const messages = makeMessages(5)
    render(<AgentTerminal messages={messages} onClear={vi.fn()} />)
    // Summary button should be active (has btn-primary class)
    const summaryButton = screen.getByRole('button', { name: /Summary/i })
    expect(summaryButton.className).toContain('btn-primary')
  })

  it('shows activity counts in Summary view', () => {
    const messages: TerminalMessage[] = [
      { content: 'Done', timestamp: '2026-01-01T10:00:00Z', type: 'output', _id: 1 },
      { content: 'Error occurred', timestamp: '2026-01-01T10:01:00Z', type: 'error', _id: 2 },
    ]
    render(<AgentTerminal messages={messages} onClear={vi.fn()} />)
    // Summary view shows activity label
    expect(screen.getByText('Activity:')).toBeInTheDocument()
  })
})
