import { useMutation } from '@tanstack/react-query'

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
    mutationFn: async (data: StandaloneReviewRequest) => {
      const response = await fetch('/api/v1/workflow/review/standalone', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Review failed')
      }
      return response.json() as Promise<StandaloneReviewResponse>
    },
  })
}

/**
 * Hook for standalone code simplify
 */
export function useStandaloneSimplify() {
  return useMutation({
    mutationFn: async (data: StandaloneSimplifyRequest) => {
      const response = await fetch('/api/v1/workflow/simplify/standalone', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Simplify failed')
      }
      return response.json() as Promise<StandaloneSimplifyResponse>
    },
  })
}
