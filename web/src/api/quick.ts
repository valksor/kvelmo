import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiRequest } from './client'

// ============================================================================
// Types
// ============================================================================

export interface QuickTask {
  id: string
  title: string
  priority: number
  labels: string[]
  status: string
  note_count: number
}

export interface QuickTaskDetail {
  id: string
  title: string
  description: string
  priority: number
  labels: string[]
  status: string
  notes: Array<{
    timestamp: string
    content: string
  }>
}

export interface QuickTaskListResponse {
  tasks: QuickTask[]
  count: number
}

export interface QuickTaskCreateRequest {
  title?: string
  description: string
  priority?: number
  labels?: string[]
}

export interface QuickTaskCreateResponse {
  queue_id: string
  task_id: string
  title: string
  created_at: string
}

export interface QuickTaskOptimizeResponse {
  success: boolean
  title: string
  original_title: string
  added_labels: string[]
  improvements: string[]
}

export interface QuickTaskExportResponse {
  success: boolean
  message: string
  path?: string
}

export interface QuickTaskSubmitRequest {
  provider: string
  labels?: string[]
  dry_run?: boolean
}

export interface QuickTaskSubmitResponse {
  success: boolean
  provider: string
  external_id?: string
  external_url?: string
  dry_run?: boolean
  title?: string
}

export interface SubmitSourceRequest {
  source: string
  provider: string
  notes?: string[]
  title?: string
  instructions?: string
  labels?: string[]
  queue_id?: string
  optimize?: boolean
  dry_run?: boolean
}

export interface SubmitSourceResponse {
  success: boolean
  queue_id: string
  task_id: string
  provider: string
  external_id?: string
  external_url?: string
  dry_run?: boolean
}

// ============================================================================
// Hooks
// ============================================================================

/**
 * Hook for listing quick tasks
 */
export function useQuickTasks() {
  return useQuery({
    queryKey: ['quick-tasks'],
    queryFn: () => apiRequest<QuickTaskListResponse>('/quick'),
    refetchInterval: 5000,
  })
}

/**
 * Hook for getting a single quick task with notes
 */
export function useQuickTask(taskId: string) {
  return useQuery({
    queryKey: ['quick-tasks', taskId],
    queryFn: () => apiRequest<QuickTaskDetail>(`/quick/${taskId}`),
    enabled: !!taskId,
  })
}

/**
 * Hook for creating a quick task
 */
export function useCreateQuickTask() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: QuickTaskCreateRequest) =>
      apiRequest<QuickTaskCreateResponse>('/quick', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['quick-tasks'] })
    },
  })
}

/**
 * Hook for adding a note to a quick task
 */
export function useAddQuickTaskNote() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ taskId, note }: { taskId: string; note: string }) =>
      apiRequest(`/quick/${taskId}/note`, {
        method: 'POST',
        body: JSON.stringify({ note }),
      }),
    onSuccess: (_, { taskId }) => {
      queryClient.invalidateQueries({ queryKey: ['quick-tasks'] })
      queryClient.invalidateQueries({ queryKey: ['quick-tasks', taskId] })
    },
  })
}

/**
 * Hook for optimizing a quick task with AI
 */
export function useOptimizeQuickTask() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ taskId, agent }: { taskId: string; agent?: string }) =>
      apiRequest<QuickTaskOptimizeResponse>(`/quick/${taskId}/optimize`, {
        method: 'POST',
        body: JSON.stringify({ agent }),
      }),
    onSuccess: (_, { taskId }) => {
      queryClient.invalidateQueries({ queryKey: ['quick-tasks'] })
      queryClient.invalidateQueries({ queryKey: ['quick-tasks', taskId] })
    },
  })
}

/**
 * Hook for exporting a quick task to markdown
 */
export function useExportQuickTask() {
  return useMutation({
    mutationFn: async ({ taskId, output }: { taskId: string; output?: string }) => {
      // If no output specified, response is the markdown file (blob)
      if (!output) {
        const blob = await apiRequest<Blob>(
          `/quick/${taskId}/export`,
          { method: 'POST', body: JSON.stringify({}) },
          'blob'
        )
        return { success: true, blob }
      }
      return apiRequest<QuickTaskExportResponse>(`/quick/${taskId}/export`, {
        method: 'POST',
        body: JSON.stringify({ output }),
      })
    },
  })
}

/**
 * Hook for submitting a quick task to a provider
 */
export function useSubmitQuickTask() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ taskId, ...data }: { taskId: string } & QuickTaskSubmitRequest) =>
      apiRequest<QuickTaskSubmitResponse>(`/quick/${taskId}/submit`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['quick-tasks'] })
    },
  })
}

/**
 * Hook for starting a quick task (begins workflow)
 */
export function useStartQuickTask() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (taskId: string) =>
      apiRequest(`/quick/${taskId}/start`, { method: 'POST' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['quick-tasks'] })
      queryClient.invalidateQueries({ queryKey: ['active-task'] })
    },
  })
}

/**
 * Hook for deleting a quick task
 */
export function useDeleteQuickTask() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (taskId: string) =>
      apiRequest(`/quick/${taskId}`, { method: 'DELETE' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['quick-tasks'] })
    },
  })
}

/**
 * Hook for creating and submitting a task from an external source
 */
export function useSubmitSource() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: SubmitSourceRequest) =>
      apiRequest<SubmitSourceResponse>('/quick/submit-source', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['quick-tasks'] })
    },
  })
}
