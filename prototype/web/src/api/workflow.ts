import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiRequest } from './client'
import type { StatusResponse, TaskResponse, WorkflowAction } from '@/types/api'

/**
 * Hook to get server status and workflow state
 */
export function useStatus() {
  return useQuery<StatusResponse>({
    queryKey: ['status'],
    queryFn: () => apiRequest('/status'),
    refetchInterval: 30000, // Fallback polling every 30s
  })
}

/**
 * Hook to get active task details
 */
export function useActiveTask(options?: { enabled?: boolean }) {
  return useQuery<TaskResponse>({
    queryKey: ['task', 'active'],
    queryFn: () => apiRequest('/task'),
    enabled: options?.enabled ?? true,
  })
}

/**
 * Hook to execute workflow actions (plan, implement, review, finish, etc.)
 */
export function useWorkflowAction() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ action, options }: { action: WorkflowAction; options?: Record<string, unknown> }) => {
      return apiRequest(`/workflow/${action}`, {
        method: 'POST',
        body: options ? JSON.stringify(options) : undefined,
      })
    },
    onSuccess: () => {
      // Invalidate relevant queries after action
      queryClient.invalidateQueries({ queryKey: ['task'] })
      queryClient.invalidateQueries({ queryKey: ['status'] })
    },
  })
}

/**
 * Hook to answer a pending agent question
 */
export function useAnswerQuestion() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ answer }: { answer: string }) => {
      return apiRequest('/workflow/answer', {
        method: 'POST',
        body: JSON.stringify({ answer }),
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['task'] })
    },
  })
}
