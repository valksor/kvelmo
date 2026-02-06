import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  useActiveTask,
  useAnswerQuestion,
  useStatus,
  useWorkflowAction,
} from './workflow'

const apiRequestMock = vi.fn()

vi.mock('./client', () => ({
  apiRequest: (...args: unknown[]) => apiRequestMock(...args),
}))

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

describe('workflow api hooks', () => {
  beforeEach(() => {
    apiRequestMock.mockReset()
    apiRequestMock.mockResolvedValue({ ok: true })
  })

  it('useStatus calls /status', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useStatus(), { wrapper: createWrapper(queryClient) })

    await waitFor(() => {
      expect(apiRequestMock).toHaveBeenCalledWith('/status')
    })
  })

  it('useActiveTask respects enabled=false', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useActiveTask({ enabled: false }), { wrapper: createWrapper(queryClient) })

    await new Promise((resolve) => setTimeout(resolve, 10))
    expect(apiRequestMock).not.toHaveBeenCalled()
  })

  it('useWorkflowAction builds implement query params and refetches', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const refetchSpy = vi.spyOn(queryClient, 'refetchQueries')

    const { result } = renderHook(() => useWorkflowAction(), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      await result.current.mutateAsync({
        action: 'implement',
        implementOptions: { component: 'api', parallel: 3 },
      })
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/workflow/implement?component=api&parallel=3', {
      method: 'POST',
      body: undefined,
    })
    expect(refetchSpy).toHaveBeenCalledWith({ queryKey: ['task', 'active'] })
    expect(refetchSpy).toHaveBeenCalledWith({ queryKey: ['status'] })
  })

  it('useAnswerQuestion posts answer and refetches active task', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const refetchSpy = vi.spyOn(queryClient, 'refetchQueries')

    const { result } = renderHook(() => useAnswerQuestion(), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      await result.current.mutateAsync({ answer: 'Yes, continue' })
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/workflow/answer', {
      method: 'POST',
      body: JSON.stringify({ answer: 'Yes, continue' }),
    })
    expect(refetchSpy).toHaveBeenCalledWith({ queryKey: ['task', 'active'] })
  })
})
