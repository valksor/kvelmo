import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiRequest } from './client'

// ============================================================================
// Types
// ============================================================================

export interface QuickTask {
  id: string
  title: string
  description: string
  priority: number
  labels: string[]
  status: string
  created_at: string
}

export interface QuickTasksResponse {
  tasks: QuickTask[]
  count: number
}

export interface CreateQuickTaskRequest {
  description: string
  title?: string
  priority?: number
  labels?: string[]
}

export interface Queue {
  id: string
  title: string
  source: string
  task_count: number
  status: string
  created_at: string
}

export interface QueuesResponse {
  queues: Queue[]
}

export interface PlanTask {
  id: string
  title: string
  description: string
  priority: number
  status: string
  depends_on: string[]
  parent_id?: string
  labels: string[]
}

export interface CreatePlanRequest {
  source: string
  title?: string
  instructions?: string
  use_schema?: boolean
}

export interface CreatePlanResponse {
  queue_id: string
  tasks: PlanTask[]
  source: string
}

export interface SourceRequest {
  type: 'text' | 'url'
  value: string
  filename?: string
}

export interface SourceResponse {
  source: string
}

export interface UploadResponse {
  source: string
  filename: string
}

export interface SubmitSourceRequest {
  source: string
  provider: string
  title?: string
  instructions?: string
  notes?: string[]
  labels?: string[]
  optimize?: boolean
  dry_run?: boolean
}

// ============================================================================
// Quick Tasks Hooks
// ============================================================================

/**
 * Hook for fetching quick tasks list
 */
export function useQuickTasks() {
  return useQuery({
    queryKey: ['quick', 'tasks'],
    queryFn: () => apiRequest<QuickTasksResponse>('/quick'),
  })
}

/**
 * Hook to create a quick task
 */
export function useCreateQuickTask() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: CreateQuickTaskRequest) => {
      return apiRequest<QuickTask>('/quick', {
        method: 'POST',
        body: JSON.stringify(data),
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['quick', 'tasks'] })
    },
  })
}

/**
 * Hook to submit from source (quick task mode)
 */
export function useSubmitSource() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: SubmitSourceRequest) => {
      return apiRequest<{ status: string }>('/quick/submit-source', {
        method: 'POST',
        body: JSON.stringify(data),
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['quick', 'tasks'] })
    },
  })
}

// ============================================================================
// Project Planning Hooks
// ============================================================================

/**
 * Hook for fetching project queues
 */
export function useQueues() {
  return useQuery({
    queryKey: ['project', 'queues'],
    queryFn: () => apiRequest<QueuesResponse>('/project/queues'),
  })
}

/**
 * Hook to create a project plan
 */
export function useCreatePlan() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: CreatePlanRequest) => {
      return apiRequest<CreatePlanResponse>('/project/plan', {
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
 * Hook to upload a file for project planning
 */
export function useUploadFile() {
  return useMutation({
    mutationFn: async (file: File) => {
      const formData = new FormData()
      formData.append('file', file)

      const response = await fetch('/api/v1/project/upload', {
        method: 'POST',
        credentials: 'include',
        body: formData,
      })

      if (!response.ok) {
        throw new Error('Failed to upload file')
      }

      return response.json() as Promise<UploadResponse>
    },
  })
}

/**
 * Hook to create a source from text or URL
 */
export function useCreateSource() {
  return useMutation({
    mutationFn: async (data: SourceRequest) => {
      return apiRequest<SourceResponse>('/project/source', {
        method: 'POST',
        body: JSON.stringify(data),
      })
    },
  })
}

// ============================================================================
// Start Task Hooks
// ============================================================================

export interface StartTaskRequest {
  ref: string
}

export interface StartTaskResponse {
  task_id: string
  branch: string
}

/**
 * Hook to start a workflow task
 */
export function useStartTask() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: StartTaskRequest) => {
      return apiRequest<StartTaskResponse>('/start', {
        method: 'POST',
        body: JSON.stringify(data),
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['task'] })
      queryClient.invalidateQueries({ queryKey: ['status'] })
    },
  })
}
