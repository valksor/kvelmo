import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useProjects, useSwitchProject, useSwitchToGlobal } from './projects'

const apiRequestMock = vi.fn()

vi.mock('./client', () => ({
  apiRequest: (...args: unknown[]) => apiRequestMock(...args),
}))

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

describe('projects api hooks', () => {
  beforeEach(() => {
    apiRequestMock.mockReset()
    apiRequestMock.mockResolvedValue({})
  })

  it('useProjects does not query when disabled', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useProjects({ enabled: false }), { wrapper: createWrapper(queryClient) })

    await new Promise((resolve) => setTimeout(resolve, 10))
    expect(apiRequestMock).not.toHaveBeenCalled()
  })

  it('useProjects queries /projects when enabled', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useProjects({ enabled: true }), { wrapper: createWrapper(queryClient) })

    await waitFor(() => {
      expect(apiRequestMock).toHaveBeenCalledWith('/projects')
    })
  })

  it('useSwitchProject posts selected path', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useSwitchProject(), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      await result.current.mutateAsync('/tmp/workspace')
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/projects/select', {
      method: 'POST',
      body: JSON.stringify({ path: '/tmp/workspace' }),
    })
  })

  it('useSwitchToGlobal posts switch endpoint', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useSwitchToGlobal(), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      await result.current.mutateAsync()
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/projects/switch', {
      method: 'POST',
    })
  })
})
