import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiRequest } from './client'

// ============================================================================
// Types
// ============================================================================

export interface QueueSummary {
  id: string
  title: string
  source: string
  task_count: number
  status: string
  created_at: string
}

export interface QueuesResponse {
  queues: QueueSummary[]
}

export interface PlanTask {
  id: string
  title: string
  description: string
  priority: number
  status: 'pending' | 'ready' | 'blocked' | 'submitted'
  depends_on: string[]
  parent_id?: string
  labels: string[]
}

export interface QueueTasksResponse {
  tasks: PlanTask[]
  queue_id: string
  queue_title: string
}

export interface UpdateTaskRequest {
  title?: string
  description?: string
  priority?: number
  status?: string
  parent_id?: string
  depends_on?: string[]
  labels?: string[]
}

export interface SubmitTasksRequest {
  queue_id: string
  provider: string
  mention?: string
  dry_run?: boolean
}

export interface SubmitTasksResponse {
  submitted: number
  provider: string
  dry_run: boolean
  results?: Array<{
    task_id: string
    external_id?: string
    error?: string
  }>
}

export interface ReorderTasksRequest {
  queue_id: string
}

export interface ReorderTasksResponse {
  reordered: boolean
  message: string
}

export interface StartImplementationRequest {
  queue_id: string
  task_id?: string // Optional: start specific task, otherwise starts next ready task
}

// ============================================================================
// Hooks
// ============================================================================

/**
 * Hook to fetch all queues
 */
export function useQueues(options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: ['project', 'queues'],
    queryFn: () => apiRequest<QueuesResponse>('/project/queues'),
    enabled: options.enabled ?? true,
  })
}

/**
 * Hook to fetch tasks in a specific queue
 */
export function useQueueTasks(queueId?: string) {
  return useQuery({
    queryKey: ['project', 'queues', queueId, 'tasks'],
    queryFn: () => apiRequest<QueueTasksResponse>(`/project/queues/${queueId}/tasks`),
    enabled: !!queueId,
  })
}

/**
 * Hook to update a task
 */
export function useUpdateTask() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ taskId, data }: { taskId: string; data: UpdateTaskRequest }) => {
      return apiRequest<{ status: string }>(`/project/tasks/${taskId}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project', 'queues'] })
    },
  })
}

/**
 * Hook to delete a queue
 */
export function useDeleteQueue() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (queueId: string) => {
      return apiRequest<{ status: string }>(`/project/queues/${queueId}`, {
        method: 'DELETE',
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project', 'queues'] })
    },
  })
}

/**
 * Hook to submit tasks to a provider
 */
export function useSubmitTasks() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: SubmitTasksRequest) => {
      return apiRequest<SubmitTasksResponse>('/project/submit', {
        method: 'POST',
        body: JSON.stringify(data),
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project', 'queues'] })
    },
  })
}

/**
 * Hook to AI reorder tasks
 */
export function useReorderTasks() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: ReorderTasksRequest) => {
      return apiRequest<ReorderTasksResponse>('/project/reorder', {
        method: 'POST',
        body: JSON.stringify(data),
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project', 'queues'] })
    },
  })
}

/**
 * Hook to start implementation
 */
export function useStartImplementation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: StartImplementationRequest) => {
      return apiRequest<{ status: string; task_id?: string }>('/project/start', {
        method: 'POST',
        body: JSON.stringify(data),
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['task'] })
      queryClient.invalidateQueries({ queryKey: ['status'] })
      queryClient.invalidateQueries({ queryKey: ['project', 'queues'] })
    },
  })
}
