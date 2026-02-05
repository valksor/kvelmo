import { useMutation } from '@tanstack/react-query'
import { apiRequest } from './client'

// ============================================================================
// Types
// ============================================================================

export type StandaloneMode = 'uncommitted' | 'branch' | 'range' | 'files'

export interface StandaloneReviewRequest {
  mode: StandaloneMode
  base_branch?: string
  range?: string
  files?: string[]
  agent?: string
  create_checkpoint?: boolean
}

export interface ReviewIssue {
  file: string
  line?: number
  severity: 'error' | 'warning' | 'info'
  message: string
  rule?: string
}

export interface StandaloneReviewResponse {
  success: boolean
  issues: ReviewIssue[]
  summary?: string
  total_issues: number
}

export interface StandaloneSimplifyRequest {
  mode: StandaloneMode
  base_branch?: string
  range?: string
  files?: string[]
  context?: number
  agent?: string
  create_checkpoint?: boolean
}

export interface FileChange {
  path: string
  operation: 'create' | 'update' | 'delete'
}

export interface UsageInfo {
  input_tokens: number
  output_tokens: number
  cached_tokens: number
  cost_usd: number
}

export interface StandaloneSimplifyResponse {
  success: boolean
  summary?: string
  changes: FileChange[]
  usage?: UsageInfo
  error?: string
}

// ============================================================================
// Hooks
// ============================================================================

/**
 * Hook for standalone code review
 */
export function useStandaloneReview() {
  return useMutation({
    mutationFn: (data: StandaloneReviewRequest) =>
      apiRequest<StandaloneReviewResponse>('/workflow/review/standalone', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  })
}

/**
 * Hook for standalone code simplify
 */
export function useStandaloneSimplify() {
  return useMutation({
    mutationFn: (data: StandaloneSimplifyRequest) =>
      apiRequest<StandaloneSimplifyResponse>('/workflow/simplify/standalone', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  })
}
