import userEvent from '@testing-library/user-event'
import { act } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@/test/test-utils'
import { NotificationCenter } from './NotificationCenter'

type SSEHandlers = {
  onStateChange?: (state: string) => void
  onQuestion?: (question: { question?: string }) => void
  onError?: (error: string) => void
}

let handlers: SSEHandlers = {}

vi.mock('@/hooks/useWorkflowSSE', () => ({
  useWorkflowSSE: (h: SSEHandlers) => {
    handlers = h
  },
}))

describe('NotificationCenter', () => {
  it('shows new notification badge and marks all as read', async () => {
    const user = userEvent.setup()
    render(<NotificationCenter />)

    act(() => {
      handlers.onStateChange?.('done')
    })

    await user.click(screen.getAllByRole('button')[0]!)
    expect(screen.getByText('Workflow Complete')).toBeInTheDocument()
    expect(screen.getByText('1')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /mark all read/i }))
    expect(screen.queryByText('1')).not.toBeInTheDocument()
  })

  it('adds question and error notifications and can clear all', async () => {
    const user = userEvent.setup()
    const { container } = render(<NotificationCenter />)

    act(() => {
      handlers.onQuestion?.({
        question: 'Need input before continuing with implementation details?',
      })
      handlers.onError?.('Connection lost')
    })

    await user.click(screen.getAllByRole('button')[0]!)
    expect(screen.getByText('Question Pending')).toBeInTheDocument()
    expect(screen.getByText('Error')).toBeInTheDocument()

    const clearAllButton = container.querySelector('.btn.btn-ghost.btn-xs.text-error') as HTMLElement
    await user.click(clearAllButton)
    await user.click(screen.getAllByRole('button')[0]!)
    expect(screen.getByText('No notifications')).toBeInTheDocument()
  })
})
