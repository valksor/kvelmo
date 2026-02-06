import { act, renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useWorkflowSSE } from './useWorkflowSSE'

class MockEventSource {
  static instances: MockEventSource[] = []

  onopen: ((this: EventSource, ev: Event) => unknown) | null = null
  onerror: ((this: EventSource, ev: Event) => unknown) | null = null

  private listeners = new Map<string, Array<(event: MessageEvent) => void>>()

  constructor(url: string) {
    void url
    MockEventSource.instances.push(this)
  }

  addEventListener(type: string, listener: (event: MessageEvent) => void) {
    const handlers = this.listeners.get(type) ?? []
    handlers.push(listener)
    this.listeners.set(type, handlers)
  }

  close() {}

  emit(type: string, data: unknown) {
    const handlers = this.listeners.get(type) ?? []
    const event = { data: JSON.stringify(data) } as MessageEvent
    handlers.forEach((handler) => handler(event))
  }
}

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

describe('useWorkflowSSE', () => {
  beforeEach(() => {
    MockEventSource.instances = []
    vi.stubGlobal('EventSource', MockEventSource)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('refetches workflow diagram when state changes', async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    })

    const refetchSpy = vi.spyOn(queryClient, 'refetchQueries')
    const wrapper = createWrapper(queryClient)

    renderHook(() => useWorkflowSSE(), { wrapper })

    const source = MockEventSource.instances[0]
    if (!source) {
      throw new Error('EventSource was not created')
    }

    act(() => {
      source.emit('state_changed', { from: 'idle', to: 'planning' })
    })

    await waitFor(() => {
      expect(refetchSpy).toHaveBeenCalledWith({ queryKey: ['workflow', 'diagram'] })
    })
  })

  it('streams progress events as agent messages', async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    })

    const wrapper = createWrapper(queryClient)
    const onAgentMessage = vi.fn()

    renderHook(() => useWorkflowSSE({ onAgentMessage }), { wrapper })

    const source = MockEventSource.instances[0]
    if (!source) {
      throw new Error('EventSource was not created')
    }

    act(() => {
      source.emit('progress', { message: 'Planning phase started', task_id: 'task-1' })
    })

    await waitFor(() => {
      expect(onAgentMessage).toHaveBeenCalledWith(
        expect.objectContaining({
          content: 'Planning phase started',
          type: 'info',
          taskId: 'task-1',
        })
      )
    })
  })

  it('filters agent messages by task ID when scoped', async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    })

    const wrapper = createWrapper(queryClient)
    const onAgentMessage = vi.fn()

    renderHook(() => useWorkflowSSE({ taskId: 'task-abc', onAgentMessage }), { wrapper })

    const source = MockEventSource.instances[0]
    if (!source) {
      throw new Error('EventSource was not created')
    }

    act(() => {
      source.emit('agent_message', { content: 'wrong task', task_id: 'task-other' })
      source.emit('agent_message', { content: 'right task', task_id: 'task-abc' })
    })

    await waitFor(() => {
      expect(onAgentMessage).toHaveBeenCalledTimes(1)
      expect(onAgentMessage).toHaveBeenCalledWith(
        expect.objectContaining({
          content: 'right task',
          taskId: 'task-abc',
        })
      )
    })
  })
})
