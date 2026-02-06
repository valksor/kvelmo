import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  useDeleteQueue,
  useQueues,
  useQueueTasks,
  useReorderTasks,
  useStartImplementation,
  useSubmitTasks,
  useUpdateTask,
} from './project-planning'

const apiRequestMock = vi.fn()

vi.mock('./client', () => ({
  apiRequest: (...args: unknown[]) => apiRequestMock(...args),
}))

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

describe('project-planning api hooks', () => {
  beforeEach(() => {
    apiRequestMock.mockReset()
    apiRequestMock.mockResolvedValue({ status: 'ok' })
  })

  it('useQueueTasks does not run without queue id', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useQueueTasks(undefined), { wrapper: createWrapper(queryClient) })

    await new Promise((resolve) => setTimeout(resolve, 10))
    expect(apiRequestMock).not.toHaveBeenCalled()
  })

  it('useQueues does not run when disabled', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useQueues({ enabled: false }), { wrapper: createWrapper(queryClient) })

    await new Promise((resolve) => setTimeout(resolve, 10))
    expect(apiRequestMock).not.toHaveBeenCalled()
  })

  it('useQueueTasks calls queue tasks endpoint when queue id exists', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useQueueTasks('queue-1'), { wrapper: createWrapper(queryClient) })

    await waitFor(() => {
      expect(apiRequestMock).toHaveBeenCalledWith('/project/queues/queue-1/tasks')
    })
  })

  it('useUpdateTask sends PUT and invalidates queues', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')

    const { result } = renderHook(() => useUpdateTask(), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      await result.current.mutateAsync({
        taskId: 'task-1',
        data: { title: 'Updated title', priority: 3 },
      })
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/project/tasks/task-1', {
      method: 'PUT',
      body: JSON.stringify({ title: 'Updated title', priority: 3 }),
    })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['project', 'queues'] })
  })

  it('useDeleteQueue sends DELETE and invalidates queues', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')

    const { result } = renderHook(() => useDeleteQueue(), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      await result.current.mutateAsync('queue-2')
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/project/queues/queue-2', { method: 'DELETE' })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['project', 'queues'] })
  })

  it('useSubmitTasks sends POST and invalidates queues', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')

    const { result } = renderHook(() => useSubmitTasks(), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      await result.current.mutateAsync({ queue_id: 'q-1', provider: 'github', dry_run: true })
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/project/submit', {
      method: 'POST',
      body: JSON.stringify({ queue_id: 'q-1', provider: 'github', dry_run: true }),
    })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['project', 'queues'] })
  })

  it('useReorderTasks sends POST and invalidates queues', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')

    const { result } = renderHook(() => useReorderTasks(), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      await result.current.mutateAsync({ queue_id: 'q-2' })
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/project/reorder', {
      method: 'POST',
      body: JSON.stringify({ queue_id: 'q-2' }),
    })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['project', 'queues'] })
  })

  it('useStartImplementation invalidates task, status, and queue queries', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')

    const { result } = renderHook(() => useStartImplementation(), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      await result.current.mutateAsync({ queue_id: 'q-3', task_id: 'task-3' })
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/project/start', {
      method: 'POST',
      body: JSON.stringify({ queue_id: 'q-3', task_id: 'task-3' }),
    })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['task'] })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['status'] })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['project', 'queues'] })
  })
})
