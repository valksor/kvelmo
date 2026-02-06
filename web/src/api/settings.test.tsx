import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  useAgents,
  useSaveSettings,
  useSettings,
  useTaskHistory,
} from './settings'

const apiRequestMock = vi.fn()

vi.mock('./client', () => ({
  apiRequest: (...args: unknown[]) => apiRequestMock(...args),
}))

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

describe('settings api hooks', () => {
  beforeEach(() => {
    apiRequestMock.mockReset()
    apiRequestMock.mockResolvedValue({})
  })

  it('useSettings queries default settings endpoint', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useSettings(), { wrapper: createWrapper(queryClient) })

    await waitFor(() => {
      expect(apiRequestMock).toHaveBeenCalledWith('/settings')
    })
  })

  it('useSettings includes project query parameter', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useSettings('proj-1'), { wrapper: createWrapper(queryClient) })

    await waitFor(() => {
      expect(apiRequestMock).toHaveBeenCalledWith('/settings?project=proj-1')
    })
  })

  it('useSaveSettings posts payload and invalidates matching query', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')

    const { result } = renderHook(() => useSaveSettings('proj-2'), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      await result.current.mutateAsync({ workflow: { auto_init: true } })
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/settings?project=proj-2', {
      method: 'POST',
      body: JSON.stringify({ workflow: { auto_init: true } }),
    })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['settings', 'proj-2'] })
  })

  it('useTaskHistory returns tasks array from response', async () => {
    apiRequestMock.mockResolvedValueOnce({
      tasks: [
        { id: 't1', title: 'Task 1', state: 'done', created_at: '2026-01-01T00:00:00Z' },
      ],
      count: 1,
    })

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useTaskHistory(), {
      wrapper: createWrapper(queryClient),
    })

    await waitFor(() => {
      expect(result.current.data).toEqual([
        { id: 't1', title: 'Task 1', state: 'done', created_at: '2026-01-01T00:00:00Z' },
      ])
    })
  })

  it('useTaskHistory uses empty list fallback when tasks is missing', async () => {
    apiRequestMock.mockResolvedValueOnce({ count: 0 })

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useTaskHistory(), {
      wrapper: createWrapper(queryClient),
    })

    await waitFor(() => {
      expect(result.current.data).toEqual([])
    })
  })

  it('useAgents queries /agents', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useAgents(), { wrapper: createWrapper(queryClient) })

    await waitFor(() => {
      expect(apiRequestMock).toHaveBeenCalledWith('/agents')
    })
  })
})
