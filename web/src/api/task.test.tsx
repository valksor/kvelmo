import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  useAddNote,
  useAgentLogsHistory,
  useImplementReview,
  useSpecificationFileDiff,
  useTaskCosts,
  useTaskNotes,
  useTaskSpecs,
  useWorkflowDiagram,
} from './task'

const apiRequestMock = vi.fn()

vi.mock('./client', () => ({
  apiRequest: (...args: unknown[]) => apiRequestMock(...args),
}))

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

describe('task api hooks', () => {
  beforeEach(() => {
    apiRequestMock.mockReset()
    apiRequestMock.mockResolvedValue({})
    vi.restoreAllMocks()
  })

  it('task detail hooks stay disabled without task id', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })

    renderHook(() => useTaskSpecs(undefined), { wrapper: createWrapper(queryClient) })
    renderHook(() => useTaskNotes(undefined), { wrapper: createWrapper(queryClient) })
    renderHook(() => useTaskCosts(undefined), { wrapper: createWrapper(queryClient) })
    renderHook(() => useAgentLogsHistory(undefined), { wrapper: createWrapper(queryClient) })

    await new Promise((resolve) => setTimeout(resolve, 10))
    expect(apiRequestMock).not.toHaveBeenCalled()
  })

  it('useAgentLogsHistory encodes task id in query string', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    renderHook(() => useAgentLogsHistory('task/123'), { wrapper: createWrapper(queryClient) })

    await waitFor(() => {
      expect(apiRequestMock).toHaveBeenCalledWith('/agent/logs/history?task_id=task%2F123')
    })
  })

  it('useWorkflowDiagram fetches plain text SVG', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      text: async () => '<svg></svg>',
    })
    vi.stubGlobal('fetch', fetchMock)

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useWorkflowDiagram(), { wrapper: createWrapper(queryClient) })

    await waitFor(() => {
      expect(result.current.data).toBe('<svg></svg>')
    })
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/workflow/diagram')
  })

  it('useAddNote posts note content and invalidates notes query', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')

    const { result } = renderHook(() => useAddNote('task-9'), { wrapper: createWrapper(queryClient) })

    await act(async () => {
      await result.current.mutateAsync('A note')
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/tasks/task-9/notes', {
      method: 'POST',
      body: JSON.stringify({ content: 'A note' }),
    })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['task', 'task-9', 'notes'] })
  })

  it('useImplementReview posts review number and refetches state', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const refetchSpy = vi.spyOn(queryClient, 'refetchQueries')

    const { result } = renderHook(() => useImplementReview(), { wrapper: createWrapper(queryClient) })

    await act(async () => {
      await result.current.mutateAsync(4)
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/workflow/implement/review/4', { method: 'POST' })
    expect(refetchSpy).toHaveBeenCalledWith({ queryKey: ['task', 'active'] })
    expect(refetchSpy).toHaveBeenCalledWith({ queryKey: ['status'] })
  })

  it('useSpecificationFileDiff encodes file path and context', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useSpecificationFileDiff('task-11'), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      await result.current.mutateAsync({
        specNumber: 2,
        filePath: 'src/a b.ts',
        context: 7,
      })
    })

    expect(apiRequestMock).toHaveBeenCalledWith('/tasks/task-11/specs/2/diff?file=src%2Fa+b.ts&context=7')
  })
})
