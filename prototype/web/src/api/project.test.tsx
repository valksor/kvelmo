import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  useCreatePlan,
  useCreateQuickTask,
  useCreateSource,
  useQueues,
  useQuickTasks,
  useStartTask,
  useSubmitSource,
  useUploadFile,
} from './project'

const apiRequestMock = vi.fn()
const getCsrfTokenMock = vi.fn()

vi.mock('./client', () => ({
  apiRequest: (...args: unknown[]) => apiRequestMock(...args),
  getCsrfToken: () => getCsrfTokenMock(),
}))

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

describe('project api hooks', () => {
  beforeEach(() => {
    apiRequestMock.mockReset()
    getCsrfTokenMock.mockReset()
    apiRequestMock.mockResolvedValue({ status: 'ok' })
    getCsrfTokenMock.mockResolvedValue('csrf-project')
  })

  it('query hooks call expected endpoints', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })

    renderHook(() => useQuickTasks(), { wrapper: createWrapper(queryClient) })
    renderHook(() => useQueues(), { wrapper: createWrapper(queryClient) })

    await waitFor(() => {
      expect(apiRequestMock).toHaveBeenCalledWith('/quick')
      expect(apiRequestMock).toHaveBeenCalledWith('/project/queues')
    })
  })

  it('create and submit mutations post payload and invalidate expected query keys', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')

    const createQuickTask = renderHook(() => useCreateQuickTask(), {
      wrapper: createWrapper(queryClient),
    })
    const submitSource = renderHook(() => useSubmitSource(), {
      wrapper: createWrapper(queryClient),
    })
    const createPlan = renderHook(() => useCreatePlan(), {
      wrapper: createWrapper(queryClient),
    })
    const createSource = renderHook(() => useCreateSource(), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      await createQuickTask.result.current.mutateAsync({ description: 'new task', priority: 2 })
      await submitSource.result.current.mutateAsync({ source: 'src', provider: 'github' })
      await createPlan.result.current.mutateAsync({ source: 'file:task.md', use_schema: true })
      await createSource.result.current.mutateAsync({ type: 'text', value: 'task text' })
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/quick', {
      method: 'POST',
      body: JSON.stringify({ description: 'new task', priority: 2 }),
    })
    expect(apiRequestMock).toHaveBeenCalledWith('/quick/submit-source', {
      method: 'POST',
      body: JSON.stringify({ source: 'src', provider: 'github' }),
    })
    expect(apiRequestMock).toHaveBeenCalledWith('/project/plan', {
      method: 'POST',
      body: JSON.stringify({ source: 'file:task.md', use_schema: true }),
    })
    expect(apiRequestMock).toHaveBeenCalledWith('/project/source', {
      method: 'POST',
      body: JSON.stringify({ type: 'text', value: 'task text' }),
    })

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['quick', 'tasks'] })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['project', 'queues'] })
  })

  it('useUploadFile sends multipart request with csrf token', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ source: 'upload:1', filename: 'a.txt' }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useUploadFile(), { wrapper: createWrapper(queryClient) })

    const file = new File(['hello'], 'a.txt', { type: 'text/plain' })

    await act(async () => {
      await result.current.mutateAsync(file)
    })

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/project/upload',
      expect.objectContaining({
        method: 'POST',
        credentials: 'include',
        headers: { 'X-Csrf-Token': 'csrf-project' },
      })
    )

    const callOptions = fetchMock.mock.calls[0]?.[1]
    expect(callOptions?.body).toBeInstanceOf(FormData)
  })

  it('useStartTask posts ref and invalidates task/status', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')

    const { result } = renderHook(() => useStartTask(), { wrapper: createWrapper(queryClient) })

    await act(async () => {
      await result.current.mutateAsync({ ref: 'quick:123' })
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/start', {
      method: 'POST',
      body: JSON.stringify({ ref: 'quick:123' }),
    })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['task'] })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['status'] })
  })
})
