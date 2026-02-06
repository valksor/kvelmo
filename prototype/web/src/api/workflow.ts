import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiRequest } from './client'
import type { StatusResponse, TaskResponse, WorkflowAction, ImplementOptions } from '@/types/api'

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
    mutationFn: async ({
      action,
      options,
      implementOptions,
    }: {
      action: WorkflowAction
      options?: Record<string, unknown>
      implementOptions?: ImplementOptions
    }) => {
      let endpoint = `/workflow/${action}`

      // Add query params for implement action
      if (action === 'implement' && implementOptions) {
        const params = new URLSearchParams()
        if (implementOptions.component) {
          params.set('component', implementOptions.component)
        }
        if (implementOptions.parallel && implementOptions.parallel > 0) {
          params.set('parallel', String(implementOptions.parallel))
        }
        const queryString = params.toString()
        if (queryString) {
          endpoint += `?${queryString}`
        }
      }

      return apiRequest(endpoint, {
        method: 'POST',
        body: options ? JSON.stringify(options) : undefined,
      })
    },
    onSuccess: () => {
      // Force immediate refetch for responsive UI after user action
      queryClient.refetchQueries({ queryKey: ['task', 'active'] })
      queryClient.refetchQueries({ queryKey: ['status'] })
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
      // Force immediate refetch after answering question
      queryClient.refetchQueries({ queryKey: ['task', 'active'] })
    },
  })
}
