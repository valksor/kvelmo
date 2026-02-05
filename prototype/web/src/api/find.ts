import { useQuery } from '@tanstack/react-query'
import { apiRequest } from './client'

// ============================================================================
// Types
// ============================================================================

export interface FindResult {
  file: string
  line: number
  content: string
  context_before?: string[]
  context_after?: string[]
  score?: number
}

export interface FindResponse {
  query: string
  results: FindResult[]
  total: number
}

// ============================================================================
// Hooks
// ============================================================================

/**
 * Hook for searching code with AI
 */
export function useFindCode(query: string) {
  return useQuery({
    queryKey: ['find', query],
    queryFn: () =>
      apiRequest<FindResponse>(`/find?q=${encodeURIComponent(query)}`),
    enabled: query.length >= 3, // Only search with 3+ characters
    staleTime: 60000, // Cache for 1 minute
  })
}
