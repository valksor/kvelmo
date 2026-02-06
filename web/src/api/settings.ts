import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiRequest } from './client'
import type { WorkspaceConfig, TaskHistoryItem } from '@/types/api'

/**
 * Fetch workspace settings (full WorkspaceConfig)
 * @param projectId - Optional project ID for global mode
 */
export function useSettings(projectId?: string) {
  const url = projectId ? `/settings?project=${projectId}` : '/settings'
  return useQuery({
    queryKey: ['settings', projectId],
    queryFn: () => apiRequest<WorkspaceConfig>(url),
    staleTime: 30_000, // Settings don't change often
  })
}

/**
 * Save workspace settings
 * @param projectId - Optional project ID for global mode
 */
export function useSaveSettings(projectId?: string) {
  const queryClient = useQueryClient()
  const url = projectId ? `/settings?project=${projectId}` : '/settings'

  return useMutation({
    mutationFn: (settings: Partial<WorkspaceConfig>) =>
      apiRequest<{ status: string; message: string }>(url, {
        method: 'POST',
        body: JSON.stringify(settings),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings', projectId] })
    },
  })
}

/**
 * Response structure for /tasks endpoint
 */
interface TaskHistoryResponse {
  tasks: TaskHistoryItem[]
  count: number
}

/**
 * Fetch task history (completed and active tasks)
 * @param options.enabled - Whether to fetch (pass false in global mode)
 */
export function useTaskHistory(options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: ['tasks', 'history'],
    queryFn: async () => {
      const response = await apiRequest<TaskHistoryResponse>('/tasks')
      return response.tasks ?? []
    },
    enabled: options.enabled ?? true,
  })
}

/**
 * Agent information returned by the API
 */
interface AgentInfo {
  name: string
  type: string
  available: boolean
  models?: { id: string; name: string }[]
}

/**
 * Response structure for /agents endpoint
 */
interface AgentsResponse {
  agents: AgentInfo[]
  count: number
}

/**
 * Fetch available agents
 */
export function useAgents() {
  return useQuery({
    queryKey: ['agents'],
    queryFn: () => apiRequest<AgentsResponse>('/agents'),
    staleTime: 60_000, // Agent list rarely changes
  })
}
