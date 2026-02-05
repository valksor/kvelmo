import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiRequest } from './client'
import type { SpecificationsResponse, NotesResponse, CostsResponse } from '@/types/api'

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
 * Hook for fetching workflow diagram SVG
 */
export function useWorkflowDiagram() {
  return useQuery({
    queryKey: ['workflow', 'diagram'],
    queryFn: async () => {
      const response = await fetch('/api/v1/workflow/diagram')
      if (!response.ok) {
        throw new Error('Failed to fetch workflow diagram')
      }
      return response.text()
    },
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
