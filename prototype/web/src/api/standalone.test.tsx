import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook } from '@testing-library/react'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useStandaloneReview, useStandaloneSimplify } from './standalone'

const apiRequestMock = vi.fn()

vi.mock('./client', () => ({
  apiRequest: (...args: unknown[]) => apiRequestMock(...args),
}))

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

describe('standalone api hooks', () => {
  beforeEach(() => {
    apiRequestMock.mockReset()
    apiRequestMock.mockResolvedValue({ success: true })
  })

  it('useStandaloneReview posts review payload', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useStandaloneReview(), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      await result.current.mutateAsync({ mode: 'files', files: ['a.ts', 'b.ts'] })
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/workflow/review/standalone', {
      method: 'POST',
      body: JSON.stringify({ mode: 'files', files: ['a.ts', 'b.ts'] }),
    })
  })

  it('useStandaloneSimplify posts simplify payload', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useStandaloneSimplify(), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      await result.current.mutateAsync({ mode: 'branch', base_branch: 'main', context: 5 })
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/workflow/simplify/standalone', {
      method: 'POST',
      body: JSON.stringify({ mode: 'branch', base_branch: 'main', context: 5 }),
    })
  })
})
