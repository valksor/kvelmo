import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useAnalyzeChanges, useApplyCommit, useChanges } from './commit'

const apiRequestMock = vi.fn()

vi.mock('./client', () => ({
  apiRequest: (...args: unknown[]) => apiRequestMock(...args),
}))

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

describe('commit api hooks', () => {
  beforeEach(() => {
    apiRequestMock.mockReset()
    apiRequestMock.mockResolvedValue({})
  })

  it('useChanges requests staged-only by default', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useChanges(), { wrapper: createWrapper(queryClient) })

    await waitFor(() => {
      expect(apiRequestMock).toHaveBeenCalledWith('/commit/changes?include_unstaged=false')
    })
  })

  it('useChanges can include unstaged files', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useChanges(true), { wrapper: createWrapper(queryClient) })

    await waitFor(() => {
      expect(apiRequestMock).toHaveBeenCalledWith('/commit/changes?include_unstaged=true')
    })
  })

  it('useAnalyzeChanges posts analyze payload', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useAnalyzeChanges(), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      await result.current.mutateAsync({ include_unstaged: true })
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/commit/analyze', {
      method: 'POST',
      body: JSON.stringify({ include_unstaged: true }),
    })
  })

  it('useApplyCommit invalidates changes after success', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')

    const { result } = renderHook(() => useApplyCommit(), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      await result.current.mutateAsync({ message: 'feat: add test' })
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/commit/apply', {
      method: 'POST',
      body: JSON.stringify({ message: 'feat: add test' }),
    })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['commit', 'changes'] })
  })
})
