import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiRequest } from './client'

// ============================================================================
// Types
// ============================================================================

export interface FileChange {
  path: string
  status: 'added' | 'modified' | 'deleted' | 'renamed'
  additions: number
  deletions: number
  diff?: string
}

export interface ChangesResponse {
  files: FileChange[]
  total_additions: number
  total_deletions: number
  has_staged: boolean
  has_unstaged: boolean
}

export interface AnalyzeRequest {
  include_unstaged?: boolean
}

export interface AnalyzeResponse {
  message: string
  title: string
  body?: string
}

export interface ApplyRequest {
  message: string
}

export interface ApplyResponse {
  commit_hash: string
  message: string
}

// ============================================================================
// Hooks
// ============================================================================

/**
 * Hook to fetch uncommitted changes
 */
export function useChanges(includeUnstaged = false) {
  return useQuery({
    queryKey: ['commit', 'changes', includeUnstaged],
    queryFn: () =>
      apiRequest<ChangesResponse>(`/commit/changes?include_unstaged=${includeUnstaged}`),
  })
}

/**
 * Hook to analyze changes and generate commit message
 */
export function useAnalyzeChanges() {
  return useMutation({
    mutationFn: async (data: AnalyzeRequest) => {
      return apiRequest<AnalyzeResponse>('/commit/analyze', {
        method: 'POST',
        body: JSON.stringify(data),
      })
    },
  })
}

/**
 * Hook to apply a commit
 */
export function useApplyCommit() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: ApplyRequest) => {
      return apiRequest<ApplyResponse>('/commit/apply', {
        method: 'POST',
        body: JSON.stringify(data),
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['commit', 'changes'] })
    },
  })
}
