import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  useAutomationConfig,
  useAutomationJob,
  useAutomationJobs,
  useAutomationStatus,
  useCancelJob,
  useRetryJob,
} from './automation'

const apiRequestMock = vi.fn()

vi.mock('./client', () => ({
  apiRequest: (...args: unknown[]) => apiRequestMock(...args),
}))

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

describe('automation api hooks', () => {
  beforeEach(() => {
    apiRequestMock.mockReset()
    apiRequestMock.mockResolvedValue({})
  })

  it('useAutomationStatus queries status endpoint', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useAutomationStatus(), { wrapper: createWrapper(queryClient) })

    await waitFor(() => {
      expect(apiRequestMock).toHaveBeenCalledWith('/automation/status')
    })
  })

  it('useAutomationJobs uses base endpoint without filter', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useAutomationJobs(), { wrapper: createWrapper(queryClient) })

    await waitFor(() => {
      expect(apiRequestMock).toHaveBeenCalledWith('/automation/jobs')
    })
  })

  it('useAutomationJobs appends status filter when provided', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useAutomationJobs('failed'), { wrapper: createWrapper(queryClient) })

    await waitFor(() => {
      expect(apiRequestMock).toHaveBeenCalledWith('/automation/jobs?status=failed')
    })
  })

  it('useAutomationJob is disabled for empty id', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useAutomationJob(''), { wrapper: createWrapper(queryClient) })

    await new Promise((resolve) => setTimeout(resolve, 10))
    expect(apiRequestMock).not.toHaveBeenCalled()
  })

  it('useAutomationConfig queries config endpoint', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useAutomationConfig(), { wrapper: createWrapper(queryClient) })

    await waitFor(() => {
      expect(apiRequestMock).toHaveBeenCalledWith('/automation/config')
    })
  })

  it('cancel and retry mutations invalidate automation lists and status', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')

    const cancelHook = renderHook(() => useCancelJob(), { wrapper: createWrapper(queryClient) })
    const retryHook = renderHook(() => useRetryJob(), { wrapper: createWrapper(queryClient) })

    await act(async () => {
      await cancelHook.result.current.mutateAsync('job-1')
      await retryHook.result.current.mutateAsync('job-2')
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/automation/jobs/job-1/cancel', { method: 'POST' })
    expect(apiRequestMock).toHaveBeenCalledWith('/automation/jobs/job-2/retry', { method: 'POST' })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['automation', 'jobs'] })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['automation', 'status'] })
  })
})
