import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render } from '@/test/test-utils'
import History from './History'

const useTaskHistoryMock = vi.fn()

vi.mock('@/api/settings', () => ({
  useTaskHistory: () => useTaskHistoryMock(),
}))

describe('History page', () => {
  beforeEach(() => {
    useTaskHistoryMock.mockReset()
  })

  it('shows loading and error states', () => {
    useTaskHistoryMock.mockReturnValueOnce({ data: undefined, isLoading: true, error: null, refetch: vi.fn() })
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { rerender } = render(
      <QueryClientProvider client={queryClient}>
        <History />
      </QueryClientProvider>
    )
    expect(document.querySelector('.animate-spin')).toBeInTheDocument()

    useTaskHistoryMock.mockReturnValueOnce({
      data: undefined,
      isLoading: false,
      error: new Error('boom'),
      refetch: vi.fn(),
    })
    rerender(<History />)
    expect(screen.getByText(/Failed to load task history/i)).toBeInTheDocument()
  })

  it('renders tasks and supports search filter', async () => {
    const user = userEvent.setup()
    useTaskHistoryMock.mockReturnValue({
      data: [
        { id: 'task-1', title: 'Alpha task', state: 'done', created_at: '2026-01-01T00:00:00Z' },
        { id: 'task-2', title: 'Beta task', state: 'failed', created_at: '2026-01-02T00:00:00Z' },
      ],
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    })

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(
      <QueryClientProvider client={queryClient}>
        <History />
      </QueryClientProvider>
    )

    expect(screen.getByText('Alpha task')).toBeInTheDocument()
    expect(screen.getByText('Beta task')).toBeInTheDocument()

    await user.type(screen.getByPlaceholderText(/Search by title or ID/i), 'Alpha')
    await waitFor(() => {
      expect(screen.getByText('Alpha task')).toBeInTheDocument()
      expect(screen.queryByText('Beta task')).not.toBeInTheDocument()
    })
  })
})
