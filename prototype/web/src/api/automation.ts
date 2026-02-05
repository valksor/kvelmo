import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiRequest } from './client'

// ============================================================================
// Types
// ============================================================================

export type JobStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'

export interface AutomationStatus {
  enabled: boolean
  running: boolean
  queue?: {
    pending: number
    running: number
    completed: number
    failed: number
  }
}

export interface JobEvent {
  id: string
  provider: string
  type: string
  action: string
  repository: string
  sender: string
  issue_number?: number
  pr_number?: number
}

export interface JobResult {
  success: boolean
  pr_number?: number
  pr_url?: string
  comments_posted?: number
  error_message?: string
  duration?: string
}

export interface AutomationJob {
  id: string
  status: JobStatus
  workflow_type: string
  priority: number
  attempts: number
  max_attempts: number
  created_at: string
  started_at?: string
  completed_at?: string
  command?: string
  error?: string
  event?: JobEvent
  result?: JobResult
}

export interface AutomationJobsResponse {
  jobs: AutomationJob[]
  count: number
}

export interface ProviderConfig {
  enabled: boolean
  command_prefix?: string
  use_worktrees?: boolean
  dry_run?: boolean
  trigger_on?: string[]
}

export interface AccessControlConfig {
  mode: string
  allowed_users?: string[]
  blocked_users?: string[]
  require_org_membership?: boolean
  allow_bots?: boolean
}

export interface QueueConfig {
  max_concurrent?: number
  job_timeout?: string
  retry_attempts?: number
  retry_delay?: string
  priority_labels?: string[]
}

export interface LabelsConfig {
  generated?: string
  in_progress?: string
  failed?: string
  skip_review?: string
}

export interface AutomationConfig {
  enabled: boolean
  providers?: Record<string, ProviderConfig>
  access_control?: AccessControlConfig
  queue?: QueueConfig
  labels?: LabelsConfig
}

// ============================================================================
// Hooks
// ============================================================================

/**
 * Hook for getting automation status
 */
export function useAutomationStatus() {
  return useQuery({
    queryKey: ['automation', 'status'],
    queryFn: () => apiRequest<AutomationStatus>('/automation/status'),
    refetchInterval: 10000,
  })
}

/**
 * Hook for listing automation jobs
 */
export function useAutomationJobs(statusFilter?: JobStatus) {
  const params = new URLSearchParams()
  if (statusFilter) {
    params.set('status', statusFilter)
  }
  const queryString = params.toString()
  const endpoint = queryString ? `/automation/jobs?${queryString}` : '/automation/jobs'

  return useQuery({
    queryKey: ['automation', 'jobs', statusFilter],
    queryFn: () => apiRequest<AutomationJobsResponse>(endpoint),
    refetchInterval: 5000,
  })
}

/**
 * Hook for getting a single job
 */
export function useAutomationJob(jobId: string) {
  return useQuery({
    queryKey: ['automation', 'jobs', jobId],
    queryFn: () => apiRequest<AutomationJob>(`/automation/jobs/${jobId}`),
    enabled: !!jobId,
  })
}

/**
 * Hook for getting automation configuration
 */
export function useAutomationConfig() {
  return useQuery({
    queryKey: ['automation', 'config'],
    queryFn: () => apiRequest<AutomationConfig>('/automation/config'),
  })
}

/**
 * Hook for canceling a job
 */
export function useCancelJob() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (jobId: string) => {
      const response = await fetch(`/api/v1/automation/jobs/${jobId}/cancel`, {
        method: 'POST',
        credentials: 'include',
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Failed to cancel job')
      }
      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['automation', 'jobs'] })
      queryClient.invalidateQueries({ queryKey: ['automation', 'status'] })
    },
  })
}

/**
 * Hook for retrying a failed job
 */
export function useRetryJob() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (jobId: string) => {
      const response = await fetch(`/api/v1/automation/jobs/${jobId}/retry`, {
        method: 'POST',
        credentials: 'include',
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Failed to retry job')
      }
      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['automation', 'jobs'] })
      queryClient.invalidateQueries({ queryKey: ['automation', 'status'] })
    },
  })
}
