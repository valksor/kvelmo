import { useQuery } from '@tanstack/react-query'
import { apiRequest } from './client'

// ============================================================================
// Types
// ============================================================================

export interface LibraryCollection {
  id: string
  name: string
  description?: string
  item_count: number
}

export interface CollectionsResponse {
  collections: LibraryCollection[]
  enabled: boolean
}

export interface LibraryItem {
  id: string
  title: string
  content: string
  collection: string
  tags?: string[]
  created_at: string
  updated_at?: string
}

export interface LibraryItemsResponse {
  items: LibraryItem[]
  collection: string
}

// ============================================================================
// Hooks
// ============================================================================

/**
 * Hook to fetch library collections
 */
export function useLibraryCollections() {
  return useQuery({
    queryKey: ['library', 'collections'],
    queryFn: () => apiRequest<CollectionsResponse>('/library'),
  })
}

/**
 * Hook to fetch items in a collection
 */
export function useLibraryItems(collectionId?: string) {
  return useQuery({
    queryKey: ['library', 'collections', collectionId, 'items'],
    queryFn: () =>
      apiRequest<LibraryItemsResponse>(`/library/collections/${collectionId}`),
    enabled: !!collectionId,
  })
}

/**
 * Hook to fetch a single library item
 */
export function useLibraryItem(itemId?: string) {
  return useQuery({
    queryKey: ['library', 'items', itemId],
    queryFn: () => apiRequest<LibraryItem>(`/library/${itemId}`),
    enabled: !!itemId,
  })
}
