import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { QuickQuestion } from './QuickQuestion'

const apiRequestMock = vi.fn()

vi.mock('@/api/client', () => ({
  apiRequest: (...args: unknown[]) => apiRequestMock(...args),
}))

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

describe('QuickQuestion', () => {
  beforeEach(() => {
    apiRequestMock.mockReset()
    apiRequestMock.mockResolvedValue({ success: true })
  })

  it('does not render for non-active workflow states', () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { container } = render(<QuickQuestion state="idle" taskId="task-1" />, {
      wrapper: createWrapper(queryClient),
    })
    expect(container).toBeEmptyDOMElement()
  })

  it('renders and submits question for active states', async () => {
    const user = userEvent.setup()
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })

    render(<QuickQuestion state="implementing" taskId="task-1" />, {
      wrapper: createWrapper(queryClient),
    })

    await user.type(screen.getByPlaceholderText(/ask a question/i), 'Need more context')
    await user.click(screen.getByRole('button'))

    expect(apiRequestMock).toHaveBeenCalledWith('/workflow/question', {
      method: 'POST',
      body: JSON.stringify({ question: 'Need more context' }),
    })
  })
})
