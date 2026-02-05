import { useQuery } from '@tanstack/react-query'
import { apiRequest } from './client'

// ============================================================================
// Types
// ============================================================================

export interface Link {
  ref: string
  title?: string
  type: string
  file: string
  line?: number
}

export interface LinksResponse {
  links: Link[]
  total: number
}

export interface BacklinksResponse {
  backlinks: Link[]
  ref: string
  total: number
}

export interface LinksStatusResponse {
  enabled: boolean
}

// ============================================================================
// Hooks
// ============================================================================

/**
 * Hook to check if links feature is enabled
 */
export function useLinksStatus() {
  return useQuery({
    queryKey: ['links', 'status'],
    queryFn: () => apiRequest<LinksStatusResponse>('/links/status'),
  })
}

/**
 * Hook to search links
 */
export function useSearchLinks(query: string) {
  return useQuery({
    queryKey: ['links', 'search', query],
    queryFn: () =>
      apiRequest<LinksResponse>(`/links?q=${encodeURIComponent(query)}`),
    enabled: query.length >= 2,
  })
}

/**
 * Hook to get backlinks for a reference
 */
export function useBacklinks(ref: string) {
  return useQuery({
    queryKey: ['links', 'backlinks', ref],
    queryFn: () =>
      apiRequest<BacklinksResponse>(`/links/backlinks/${encodeURIComponent(ref)}`),
    enabled: !!ref,
  })
}
