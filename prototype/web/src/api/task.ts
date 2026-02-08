import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiRequest } from './client'
import type {
  SpecificationsResponse,
  NotesResponse,
  CostsResponse,
  SpecificationDiffResponse,
  AgentLogsHistoryResponse,
  WorkResponse,
  LabelsResponse,
} from '@/types/api'

/**
 * Hook for fetching task specifications
 */
export function useTaskSpecs(taskId?: string) {
  return useQuery({
    queryKey: ['task', taskId, 'specs'],
    queryFn: () => apiRequest<SpecificationsResponse>(`/tasks/${taskId}/specs`),
    enabled: !!taskId,
  })
}

/**
 * Hook for fetching task notes
 */
export function useTaskNotes(taskId?: string) {
  return useQuery({
    queryKey: ['task', taskId, 'notes'],
    queryFn: () => apiRequest<NotesResponse>(`/tasks/${taskId}/notes`),
    enabled: !!taskId,
  })
}

/**
 * Hook for fetching task costs
 */
export function useTaskCosts(taskId?: string) {
  return useQuery({
    queryKey: ['task', taskId, 'costs'],
    queryFn: () => apiRequest<CostsResponse>(`/tasks/${taskId}/costs`),
    enabled: !!taskId,
  })
}

/**
 * Hook for fetching persisted agent output history for a task
 */
export function useAgentLogsHistory(taskId?: string) {
  return useQuery({
    queryKey: ['task', taskId, 'agent-logs'],
    queryFn: () =>
      apiRequest<AgentLogsHistoryResponse>(`/agent/logs/history?task_id=${encodeURIComponent(taskId || '')}`),
    enabled: !!taskId,
  })
}

/**
 * Hook to add a note to a task
 */
export function useAddNote(taskId?: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (content: string) => {
      return apiRequest<{ status: string }>(`/tasks/${taskId}/notes`, {
        method: 'POST',
        body: JSON.stringify({ content }),
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['task', taskId, 'notes'] })
    },
  })
}

/**
 * Hook to implement fixes for a specific review
 */
export function useImplementReview() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (reviewNumber: number) => {
      return apiRequest<{ success: boolean; message: string; review_number: number }>(
        `/workflow/implement/review/${reviewNumber}`,
        { method: 'POST' }
      )
    },
    onSuccess: () => {
      queryClient.refetchQueries({ queryKey: ['task', 'active'] })
      queryClient.refetchQueries({ queryKey: ['status'] })
    },
  })
}

interface SpecificationDiffRequest {
  specNumber: number
  filePath: string
  context?: number
}

/**
 * Hook to fetch unified diff for a specification implemented file
 */
export function useSpecificationFileDiff(taskId?: string) {
  return useMutation({
    mutationFn: async ({
      specNumber,
      filePath,
      context = 3,
    }: SpecificationDiffRequest): Promise<SpecificationDiffResponse> => {
      const query = new URLSearchParams({
        file: filePath,
        context: String(context),
      })

      return apiRequest<SpecificationDiffResponse>(`/tasks/${taskId}/specs/${specNumber}/diff?${query}`)
    },
  })
}

/**
 * Hook for fetching work data by task ID.
 * Works for both active and completed tasks.
 */
export function useTaskWork(taskId?: string) {
  return useQuery({
    queryKey: ['work', taskId],
    queryFn: () => apiRequest<WorkResponse>(`/work/${taskId}`),
    enabled: !!taskId,
  })
}

/**
 * Hook for fetching task labels
 */
export function useTaskLabels() {
  return useQuery({
    queryKey: ['task', 'labels'],
    queryFn: () => apiRequest<LabelsResponse>('/labels'),
  })
}

/**
 * Hook to add a label to the current task
 */
export function useAddLabel() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (label: string) => {
      return apiRequest<{ success: boolean; message?: string; error?: string }>('/task/labels', {
        method: 'POST',
        body: JSON.stringify({ action: 'add', label }),
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['task', 'labels'] })
      queryClient.invalidateQueries({ queryKey: ['work'] })
    },
  })
}

/**
 * Hook to remove a label from the current task
 */
export function useRemoveLabel() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (label: string) => {
      return apiRequest<{ success: boolean; message?: string; error?: string }>('/task/labels', {
        method: 'POST',
        body: JSON.stringify({ action: 'remove', label }),
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['task', 'labels'] })
      queryClient.invalidateQueries({ queryKey: ['work'] })
    },
  })
}
