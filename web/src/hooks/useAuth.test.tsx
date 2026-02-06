import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useAuth } from './useAuth'

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

describe('useAuth', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns authenticated status on successful response', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ state: 'idle', running: true, task_id: 'task-1' }),
      })
    )

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useAuth(), { wrapper: createWrapper(queryClient) })

    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(true)
      expect(result.current.status).toEqual({ state: 'idle', running: true, task_id: 'task-1' })
    })
  })

  it('returns error when unauthorized', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
      })
    )

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useAuth(), { wrapper: createWrapper(queryClient) })

    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(false)
      expect(result.current.error).toBeDefined()
    })
  })

  it('returns error when status endpoint fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
      })
    )

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useAuth(), { wrapper: createWrapper(queryClient) })

    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(false)
      expect(result.current.error).toBeDefined()
    })
  })
})
