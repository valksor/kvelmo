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
    mutationFn: async (data: QuickTaskCreateRequest) => {
      const response = await fetch('/api/v1/quick', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Failed to create task')
      }
      return response.json() as Promise<QuickTaskCreateResponse>
    },
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
    mutationFn: async ({ taskId, note }: { taskId: string; note: string }) => {
      const response = await fetch(`/api/v1/quick/${taskId}/note`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ note }),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Failed to add note')
      }
      return response.json()
    },
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
    mutationFn: async ({ taskId, agent }: { taskId: string; agent?: string }) => {
      const response = await fetch(`/api/v1/quick/${taskId}/optimize`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ agent }),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Optimization failed')
      }
      return response.json() as Promise<QuickTaskOptimizeResponse>
    },
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
      const response = await fetch(`/api/v1/quick/${taskId}/export`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ output }),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Export failed')
      }
      // If no output specified, response is the markdown file
      if (!output) {
        const blob = await response.blob()
        return { success: true, blob }
      }
      return response.json() as Promise<QuickTaskExportResponse>
    },
  })
}

/**
 * Hook for submitting a quick task to a provider
 */
export function useSubmitQuickTask() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ taskId, ...data }: { taskId: string } & QuickTaskSubmitRequest) => {
      const response = await fetch(`/api/v1/quick/${taskId}/submit`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Submission failed')
      }
      return response.json() as Promise<QuickTaskSubmitResponse>
    },
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
    mutationFn: async (taskId: string) => {
      const response = await fetch(`/api/v1/quick/${taskId}/start`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Failed to start task')
      }
      return response.json()
    },
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
    mutationFn: async (taskId: string) => {
      const response = await fetch(`/api/v1/quick/${taskId}`, {
        method: 'DELETE',
        credentials: 'include',
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Failed to delete task')
      }
      return response.json()
    },
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
    mutationFn: async (data: SubmitSourceRequest) => {
      const response = await fetch('/api/v1/quick/submit-source', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Submission failed')
      }
      return response.json() as Promise<SubmitSourceResponse>
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['quick-tasks'] })
    },
  })
}
