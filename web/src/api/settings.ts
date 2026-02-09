import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiRequest } from './client'
import type { WorkspaceConfig, TaskHistoryItem } from '@/types/api'
import type { SettingsResponseV2 } from '@/types/schema'

/**
 * Fetch workspace settings with schema.
 * Returns both the schema definition and current values.
 * @param projectId - Optional project ID for global mode
 */
export function useSettings(projectId?: string) {
  const url = projectId ? `/settings?project=${projectId}` : '/settings'

  return useQuery({
    queryKey: ['settings', projectId],
    queryFn: () => apiRequest<SettingsResponseV2>(url),
    staleTime: 30_000,
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
 * Agent capabilities returned by the API
 */
export interface AgentCapabilities {
  streaming: boolean
  tool_use: boolean
  file_operations: boolean
  code_execution: boolean
  multi_turn: boolean
  system_prompt: boolean
  allowed_tools?: string[]
}

/**
 * Agent model information
 */
export interface AgentModel {
  id: string
  name: string
  default?: boolean
  max_tokens?: number
  input_cost_usd?: number
  output_cost_usd?: number
}

/**
 * Agent information returned by the API
 */
export interface AgentInfo {
  name: string
  type: 'built-in' | 'alias'
  extends?: string           // For alias agents - base agent name
  description?: string
  version?: string
  available: boolean
  capabilities?: AgentCapabilities
  models?: AgentModel[]
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

/**
 * Response structure for /docs-url endpoint
 */
interface DocsURLResponse {
  url: string
  version: string
}

/**
 * Fetch documentation URL for current build version.
 * Returns /docs/latest for stable releases (v*), /docs/nightly otherwise.
 */
export function useDocsURL() {
  return useQuery({
    queryKey: ['docs-url'],
    queryFn: () => apiRequest<DocsURLResponse>('/docs-url'),
    staleTime: Infinity, // Version never changes during session
  })
}

/**
 * Response structure for /config/reinit endpoint
 */
interface ConfigReinitResponse {
  status: string
  message: string
  old_version?: number
  new_version?: number
}

/**
 * Re-initialize config while preserving key settings.
 * Used when config version is outdated.
 * @param projectId - Optional project ID for global mode
 */
export function useReinitConfig(projectId?: string) {
  const queryClient = useQueryClient()
  const url = projectId ? `/config/reinit?project=${projectId}` : '/config/reinit'

  return useMutation({
    mutationFn: () =>
      apiRequest<ConfigReinitResponse>(url, {
        method: 'POST',
      }),
    onSuccess: () => {
      // Invalidate status to refresh config version info
      queryClient.invalidateQueries({ queryKey: ['status'] })
      // Also invalidate settings since config changed
      queryClient.invalidateQueries({ queryKey: ['settings', projectId] })
    },
  })
}
