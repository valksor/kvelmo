import { useEffect, useRef, useState, type MutableRefObject } from 'react'
import { useQueryClient, type QueryClient } from '@tanstack/react-query'
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

interface ParsedEventData {
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

interface WorkflowSubscriber {
  setConnected: (connected: boolean) => void
  enabledRef: MutableRefObject<boolean>
  taskIDRef: MutableRefObject<string | undefined>
  onStateChangeRef: MutableRefObject<UseWorkflowSSEOptions['onStateChange']>
  onAgentMessageRef: MutableRefObject<UseWorkflowSSEOptions['onAgentMessage']>
  onQuestionRef: MutableRefObject<UseWorkflowSSEOptions['onQuestion']>
  onErrorRef: MutableRefObject<UseWorkflowSSEOptions['onError']>
}

let nextSubscriberID = 0
const subscribers = new Map<number, WorkflowSubscriber>()

let sharedQueryClient: QueryClient | null = null
let sharedEventSource: EventSource | null = null
let reconnectTimeoutID: number | undefined
let isSharedConnected = false

function hasEnabledSubscribers(): boolean {
  for (const subscriber of subscribers.values()) {
    if (subscriber.enabledRef.current) {
      return true
    }
  }
  return false
}

function broadcastConnected(connected: boolean) {
  isSharedConnected = connected
  subscribers.forEach((subscriber) => {
    subscriber.setConnected(connected)
  })
}

function refreshQueriesForEvent(eventType: SSEEventType) {
  if (!sharedQueryClient) {
    return
  }

  switch (eventType) {
    case 'state_changed':
      sharedQueryClient.refetchQueries({ queryKey: ['status'] })
      sharedQueryClient.refetchQueries({ queryKey: ['task'] })
      sharedQueryClient.refetchQueries({ queryKey: ['workflow', 'diagram'] })
      break
    case 'progress':
    case 'agent_message':
      sharedQueryClient.invalidateQueries({ queryKey: ['task', 'active'] })
      break
    case 'costs_updated':
      sharedQueryClient.invalidateQueries({ queryKey: ['task', 'active'] })
      break
    case 'spec_updated':
      sharedQueryClient.refetchQueries({ queryKey: ['task'] })
      break
    case 'question_asked':
      sharedQueryClient.refetchQueries({ queryKey: ['task', 'active'] })
      break
  }
}

function notifySubscribers(eventType: string, data: ParsedEventData) {
  const eventTaskID = data.task_id

  subscribers.forEach((subscriber) => {
    if (!subscriber.enabledRef.current) {
      return
    }

    const scopedTaskID = subscriber.taskIDRef.current
    const matchesTask = !scopedTaskID || !eventTaskID || scopedTaskID === eventTaskID

    if (eventType === 'state_changed' && matchesTask && subscriber.onStateChangeRef.current) {
      const newState = data.to || data.state
      if (newState) {
        subscriber.onStateChangeRef.current(newState)
      }
    }

    if ((eventType === 'agent_message' || eventType === 'progress') && matchesTask && subscriber.onAgentMessageRef.current) {
      const content =
        eventType === 'progress'
          ? (data.message || '').trim()
          : (data.content || data.message || data.text || '').trim()

      if (content) {
        subscriber.onAgentMessageRef.current({
          content,
          timestamp: data.timestamp || new Date().toISOString(),
          taskId: eventTaskID,
          type: eventType === 'progress' ? 'info' : data.error ? 'error' : 'output',
        })
      }
    }

    if (eventType === 'question_asked' && subscriber.onQuestionRef.current) {
      subscriber.onQuestionRef.current({
        question: data.question,
        options: data.options,
      })
    }

    if (subscriber.onErrorRef.current && data.error) {
      subscriber.onErrorRef.current(data.error)
    }
  })
}

function handleIncomingEvent(eventType: string, event: MessageEvent) {
  let data: ParsedEventData
  try {
    data = JSON.parse(event.data) as ParsedEventData
  } catch {
    return
  }

  refreshQueriesForEvent(eventType as SSEEventType)
  notifySubscribers(eventType, data)
}

function closeSharedConnection() {
  if (reconnectTimeoutID) {
    clearTimeout(reconnectTimeoutID)
    reconnectTimeoutID = undefined
  }

  if (sharedEventSource) {
    sharedEventSource.close()
    sharedEventSource = null
  }

  broadcastConnected(false)
}

function connectShared() {
  if (sharedEventSource || !sharedQueryClient || !hasEnabledSubscribers()) {
    return
  }

  const source = new EventSource('/api/v1/events')
  sharedEventSource = source

  source.onopen = () => {
    broadcastConnected(true)
  }

  const eventTypes: string[] = [
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
    source.addEventListener(eventType, (event) => {
      handleIncomingEvent(eventType, event as MessageEvent)
    })
  })

  source.onerror = () => {
    source.close()
    if (sharedEventSource === source) {
      sharedEventSource = null
    }

    broadcastConnected(false)

    if (reconnectTimeoutID) {
      clearTimeout(reconnectTimeoutID)
    }

    reconnectTimeoutID = window.setTimeout(() => {
      reconnectTimeoutID = undefined
      if (hasEnabledSubscribers()) {
        connectShared()
      }
    }, 2000)
  }
}

function syncConnection() {
  if (hasEnabledSubscribers()) {
    connectShared()
    return
  }

  closeSharedConnection()
}

/**
 * Hook for SSE connection to receive real-time workflow updates.
 * Uses a shared singleton EventSource to avoid opening duplicate connections.
 */
export function useWorkflowSSE(options: UseWorkflowSSEOptions = {}) {
  const { enabled = true, taskId, onStateChange, onAgentMessage, onQuestion, onError } = options

  const queryClient = useQueryClient()
  const [connected, setConnected] = useState(isSharedConnected)

  const enabledRef = useRef(enabled)
  const taskIDRef = useRef(taskId)
  const onStateChangeRef = useRef(onStateChange)
  const onAgentMessageRef = useRef(onAgentMessage)
  const onQuestionRef = useRef(onQuestion)
  const onErrorRef = useRef(onError)
  const subscriberIDRef = useRef<number | null>(null)

  useEffect(() => {
    enabledRef.current = enabled
    taskIDRef.current = taskId
    onStateChangeRef.current = onStateChange
    onAgentMessageRef.current = onAgentMessage
    onQuestionRef.current = onQuestion
    onErrorRef.current = onError
  })

  useEffect(() => {
    sharedQueryClient = queryClient

    const subscriberID = ++nextSubscriberID
    subscriberIDRef.current = subscriberID

    subscribers.set(subscriberID, {
      setConnected,
      enabledRef,
      taskIDRef,
      onStateChangeRef,
      onAgentMessageRef,
      onQuestionRef,
      onErrorRef,
    })

    syncConnection()

    return () => {
      subscribers.delete(subscriberID)
      subscriberIDRef.current = null

      if (subscribers.size === 0) {
        sharedQueryClient = null
      }

      syncConnection()
    }
  }, [queryClient])

  useEffect(() => {
    const subscriberID = subscriberIDRef.current
    if (!subscriberID) {
      return
    }

    const subscriber = subscribers.get(subscriberID)
    if (subscriber) {
      subscriber.enabledRef.current = enabled
    }

    syncConnection()
  }, [enabled])

  return { connected }
}
