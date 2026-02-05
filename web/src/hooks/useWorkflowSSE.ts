import { useEffect, useRef, useCallback, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import type { SSEEventType } from '@/types/api'

export interface AgentMessage {
  content: string
  timestamp: string
  type?: 'output' | 'error' | 'info'
}

export interface QuestionData {
  question?: string
  options?: string[]
}

interface UseWorkflowSSEOptions {
  enabled?: boolean
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
  const { enabled = true, onStateChange, onAgentMessage, onQuestion, onError } = options
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

  // Update refs when callbacks change (doesn't trigger reconnection)
  useEffect(() => {
    onStateChangeRef.current = onStateChange
    onAgentMessageRef.current = onAgentMessage
    onQuestionRef.current = onQuestion
    onErrorRef.current = onError
    enabledRef.current = enabled
  })

  const handleEvent = useCallback(
    (eventType: SSEEventType) => {
      switch (eventType) {
        case 'state_changed':
          queryClient.invalidateQueries({ queryKey: ['status'] })
          queryClient.invalidateQueries({ queryKey: ['task', 'active'] })
          break
        case 'progress':
        case 'agent_message':
          queryClient.invalidateQueries({ queryKey: ['task', 'active'] })
          break
        case 'costs_updated':
          queryClient.invalidateQueries({ queryKey: ['task', 'active'] })
          break
        case 'spec_updated':
          queryClient.invalidateQueries({ queryKey: ['specifications'] })
          break
        case 'question_asked':
          queryClient.invalidateQueries({ queryKey: ['task', 'active'] })
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

    es.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as {
          type: SSEEventType
          state?: string
          content?: string
          message?: string
          error?: string
        }
        handleEvent(data.type)

        // Callback for state changes (using ref for stable reference)
        if (data.type === 'state_changed' && data.state && onStateChangeRef.current) {
          onStateChangeRef.current(data.state)
        }

        // Callback for agent messages (using ref for stable reference)
        if (data.type === 'agent_message' && onAgentMessageRef.current) {
          const content = data.content || data.message || ''
          if (content) {
            onAgentMessageRef.current({
              content,
              timestamp: new Date().toISOString(),
              type: data.error ? 'error' : 'output',
            })
          }
        }

        // Callback for questions (using ref for stable reference)
        if (data.type === 'question_asked' && onQuestionRef.current) {
          onQuestionRef.current({
            question: (data as { question?: string }).question,
            options: (data as { options?: string[] }).options,
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
