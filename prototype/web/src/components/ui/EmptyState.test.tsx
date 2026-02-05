import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { EmptyState } from './EmptyState'

describe('EmptyState', () => {
  it('renders default title and description', () => {
    render(<EmptyState />)

    expect(screen.getByText('No active task')).toBeInTheDocument()
    expect(screen.getByText('Start a new task to begin working.')).toBeInTheDocument()
  })

  it('renders custom title and description', () => {
    render(
      <EmptyState
        title="Custom Title"
        description="Custom description text"
      />
    )

    expect(screen.getByText('Custom Title')).toBeInTheDocument()
    expect(screen.getByText('Custom description text')).toBeInTheDocument()
  })

  it('renders action slot when provided', () => {
    render(
      <EmptyState
        action={<button>Click Me</button>}
      />
    )

    expect(screen.getByRole('button', { name: /click me/i })).toBeInTheDocument()
  })

  it('does not render action slot when not provided', () => {
    render(<EmptyState />)

    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('renders inbox icon', () => {
    render(<EmptyState />)

    // The Inbox icon from lucide-react should be present
    expect(document.querySelector('svg')).toBeInTheDocument()
  })
})
