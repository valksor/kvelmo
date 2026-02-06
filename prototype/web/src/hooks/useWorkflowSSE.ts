import { useEffect, useRef, useCallback, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import type { SSEEventType } from '@/types/api'

export interface AgentMessage {
  content: string
  timestamp: string
  type?: 'output' | 'error' | 'info'
  taskId?: string
}

export interface QuestionData {
  question?: string
  options?: string[]
}

interface UseWorkflowSSEOptions {
  enabled?: boolean
  taskId?: string
  onStateChange?: (state: string) => void
  onAgentMessage?: (message: AgentMessage) => void
  onQuestion?: (data: QuestionData) => void
  onError?: (error: string) => void
}

/**
 * Hook for SSE connection to receive real-time workflow updates.
 * Only use this on pages that need real-time updates (Dashboard, Task Detail).
 *
 * Uses refs for callback functions to prevent unnecessary reconnections
 * when callbacks change between renders.
 */
export function useWorkflowSSE(options: UseWorkflowSSEOptions = {}) {
  const { enabled = true, taskId, onStateChange, onAgentMessage, onQuestion, onError } = options
  const queryClient = useQueryClient()
  const eventSourceRef = useRef<EventSource | null>(null)
  const reconnectTimeoutRef = useRef<number | undefined>(undefined)
  const [connected, setConnected] = useState(false)

  // Store callbacks in refs to avoid dependency changes that would cause reconnections
  const onStateChangeRef = useRef(onStateChange)
  const onAgentMessageRef = useRef(onAgentMessage)
  const onQuestionRef = useRef(onQuestion)
  const onErrorRef = useRef(onError)
  const enabledRef = useRef(enabled)
  const taskIDRef = useRef(taskId)

  // Update refs when callbacks change (doesn't trigger reconnection)
  useEffect(() => {
    onStateChangeRef.current = onStateChange
    onAgentMessageRef.current = onAgentMessage
    onQuestionRef.current = onQuestion
    onErrorRef.current = onError
    enabledRef.current = enabled
    taskIDRef.current = taskId
  })

  const handleEvent = useCallback(
    (eventType: SSEEventType) => {
      switch (eventType) {
        case 'state_changed':
          // Force immediate refetch for responsive UI on state changes.
          // Use ['task'] prefix to refetch ALL task queries (active, specs, notes, costs),
          // and refetch workflow diagram SVG so state highlight updates immediately.
          queryClient.refetchQueries({ queryKey: ['status'] })
          queryClient.refetchQueries({ queryKey: ['task'] })
          queryClient.refetchQueries({ queryKey: ['workflow', 'diagram'] })
          break
        case 'progress':
        case 'agent_message':
          queryClient.invalidateQueries({ queryKey: ['task', 'active'] })
          break
        case 'costs_updated':
          queryClient.invalidateQueries({ queryKey: ['task', 'active'] })
          break
        case 'spec_updated':
          // Also refetch specs immediately so Implement button enables
          queryClient.refetchQueries({ queryKey: ['task'] })
          break
        case 'question_asked':
          queryClient.refetchQueries({ queryKey: ['task', 'active'] })
          break
      }
    },
    [queryClient]
  )

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = undefined
    }
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
    }
    setConnected(false)
  }, [])

  // Ref to hold the connect function for recursive reconnection
  const connectRef = useRef<() => void>(() => {})

  const connect = useCallback(() => {
    if (eventSourceRef.current) return

    const es = new EventSource('/api/v1/events')
    eventSourceRef.current = es

    es.onopen = () => {
      setConnected(true)
    }

    // Helper to handle events - parses data and triggers callbacks
    const createEventHandler = (eventType: string) => (event: MessageEvent) => {
      try {
        const data = JSON.parse(event.data) as {
          from?: string
          to?: string
          state?: string
          content?: string
          message?: string
          text?: string
          error?: string
          question?: string
          options?: string[]
          task_id?: string
          timestamp?: string
        }

        // Trigger query invalidation/refetch based on event type
        handleEvent(eventType as SSEEventType)

        const scopedTaskID = taskIDRef.current
        const eventTaskID = data.task_id
        const matchesTask = !scopedTaskID || !eventTaskID || scopedTaskID === eventTaskID

        // Callback for state changes (using ref for stable reference)
        // Backend sends { from, to, event, task_id } for state_changed
        if (eventType === 'state_changed' && matchesTask && onStateChangeRef.current) {
          const newState = data.to || data.state
          if (newState) {
            onStateChangeRef.current(newState)
          }
        }

        // Callback for terminal output (agent events + progress updates)
        if ((eventType === 'agent_message' || eventType === 'progress') && matchesTask && onAgentMessageRef.current) {
          const content =
            eventType === 'progress'
              ? (data.message || '').trim()
              : (data.content || data.message || data.text || '').trim()
          if (content) {
            onAgentMessageRef.current({
              content,
              timestamp: data.timestamp || new Date().toISOString(),
              taskId: eventTaskID,
              type: eventType === 'progress' ? 'info' : data.error ? 'error' : 'output',
            })
          }
        }

        // Callback for questions (using ref for stable reference)
        if (eventType === 'question_asked' && onQuestionRef.current) {
          onQuestionRef.current({
            question: data.question,
            options: data.options,
          })
        }

        // Callback for errors (using ref for stable reference)
        if (onErrorRef.current && data.error) {
          onErrorRef.current(data.error)
        }
      } catch {
        // Ignore parse errors
      }
    }

    // Listen for specific SSE event types (NOT onmessage which only handles "message" events!)
    // The backend sends named events like "event: state_changed\ndata: {...}"
    const eventTypes = [
      'state_changed',
      'progress',
      'agent_message',
      'costs_updated',
      'spec_updated',
      'question_asked',
      'heartbeat',
      'connected',
      'error',
    ]

    eventTypes.forEach((eventType) => {
      es.addEventListener(eventType, createEventHandler(eventType))
    })

    es.onerror = () => {
      es.close()
      eventSourceRef.current = null
      setConnected(false)

      // Reconnect after 2 seconds if still enabled (using ref for current value)
      reconnectTimeoutRef.current = window.setTimeout(() => {
        if (enabledRef.current) connectRef.current()
      }, 2000)
    }
  }, [handleEvent]) // Only depends on handleEvent (which only depends on queryClient)

  // Keep connectRef in sync with connect
  useEffect(() => {
    connectRef.current = connect
  })

  // Main effect: only re-runs when `enabled` changes
  useEffect(() => {
    if (enabled) {
      connect()
    } else {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- valid cleanup pattern
      disconnect()
    }

    return disconnect
  }, [enabled, connect, disconnect])

  return { connected }
}
