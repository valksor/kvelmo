import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  useAddQuickTaskNote,
  useCreateQuickTask,
  useDeleteQuickTask,
  useExportQuickTask,
  useOptimizeQuickTask,
  useQuickTask,
  useQuickTasks,
  useStartQuickTask,
  useSubmitQuickTask,
  useSubmitSource,
} from './quick'

const apiRequestMock = vi.fn()

vi.mock('./client', () => ({
  apiRequest: (...args: unknown[]) => apiRequestMock(...args),
}))

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

describe('quick api hooks', () => {
  beforeEach(() => {
    apiRequestMock.mockReset()
    apiRequestMock.mockResolvedValue({ success: true })
  })

  it('list/detail hooks call endpoints and detail waits for id', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })

    renderHook(() => useQuickTasks(), { wrapper: createWrapper(queryClient) })
    renderHook(() => useQuickTask(''), { wrapper: createWrapper(queryClient) })

    await waitFor(() => {
      expect(apiRequestMock).toHaveBeenCalledWith('/quick')
    })
    expect(apiRequestMock).not.toHaveBeenCalledWith('/quick/')

    renderHook(() => useQuickTask('task-1'), { wrapper: createWrapper(queryClient) })
    await waitFor(() => {
      expect(apiRequestMock).toHaveBeenCalledWith('/quick/task-1')
    })
  })

  it('mutation hooks send payloads and invalidate related keys', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')

    const createHook = renderHook(() => useCreateQuickTask(), { wrapper: createWrapper(queryClient) })
    const addNoteHook = renderHook(() => useAddQuickTaskNote(), { wrapper: createWrapper(queryClient) })
    const optimizeHook = renderHook(() => useOptimizeQuickTask(), { wrapper: createWrapper(queryClient) })
    const submitHook = renderHook(() => useSubmitQuickTask(), { wrapper: createWrapper(queryClient) })
    const startHook = renderHook(() => useStartQuickTask(), { wrapper: createWrapper(queryClient) })
    const deleteHook = renderHook(() => useDeleteQuickTask(), { wrapper: createWrapper(queryClient) })
    const sourceHook = renderHook(() => useSubmitSource(), { wrapper: createWrapper(queryClient) })

    await act(async () => {
      await createHook.result.current.mutateAsync({ description: 'desc', labels: ['a'] })
      await addNoteHook.result.current.mutateAsync({ taskId: 'q-1', note: 'note text' })
      await optimizeHook.result.current.mutateAsync({ taskId: 'q-2', agent: 'claude' })
      await submitHook.result.current.mutateAsync({ taskId: 'q-3', provider: 'github' })
      await startHook.result.current.mutateAsync('q-4')
      await deleteHook.result.current.mutateAsync('q-5')
      await sourceHook.result.current.mutateAsync({ source: 'text', provider: 'linear' })
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/quick', {
      method: 'POST',
      body: JSON.stringify({ description: 'desc', labels: ['a'] }),
    })
    expect(apiRequestMock).toHaveBeenCalledWith('/quick/q-1/note', {
      method: 'POST',
      body: JSON.stringify({ note: 'note text' }),
    })
    expect(apiRequestMock).toHaveBeenCalledWith('/quick/q-2/optimize', {
      method: 'POST',
      body: JSON.stringify({ agent: 'claude' }),
    })
    expect(apiRequestMock).toHaveBeenCalledWith('/quick/q-3/submit', {
      method: 'POST',
      body: JSON.stringify({ provider: 'github' }),
    })
    expect(apiRequestMock).toHaveBeenCalledWith('/quick/q-4/start', { method: 'POST' })
    expect(apiRequestMock).toHaveBeenCalledWith('/quick/q-5', { method: 'DELETE' })
    expect(apiRequestMock).toHaveBeenCalledWith('/quick/submit-source', {
      method: 'POST',
      body: JSON.stringify({ source: 'text', provider: 'linear' }),
    })

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['quick-tasks'] })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['quick-tasks', 'q-1'] })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['active-task'] })
  })

  it('useExportQuickTask returns blob branch when output is omitted', async () => {
    const blob = new Blob(['abc'], { type: 'text/markdown' })
    apiRequestMock.mockResolvedValueOnce(blob)

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useExportQuickTask(), { wrapper: createWrapper(queryClient) })

    await act(async () => {
      const response = await result.current.mutateAsync({ taskId: 'q-8' })
      expect(response).toEqual({ success: true, blob })
    })

    expect(apiRequestMock).toHaveBeenCalledWith(
      '/quick/q-8/export',
      { method: 'POST', body: JSON.stringify({}) },
      'blob'
    )
  })

  it('useExportQuickTask sends output payload when provided', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useExportQuickTask(), { wrapper: createWrapper(queryClient) })

    await act(async () => {
      await result.current.mutateAsync({ taskId: 'q-9', output: 'tmp/task.md' })
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/quick/q-9/export', {
      method: 'POST',
      body: JSON.stringify({ output: 'tmp/task.md' }),
    })
  })
})
