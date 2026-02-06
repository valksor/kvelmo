import { useQuery } from '@tanstack/react-query'
import { apiRequest } from './client'

// ============================================================================
// Types
// ============================================================================

export interface LibraryCollection {
  id: string
  name: string
  description?: string
  page_count: number
  item_count: number
}

export interface CollectionsResponse {
  collections: LibraryCollection[]
  count: number
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
  count: number
}

interface ServerLibraryCollection {
  id: string
  name: string
  source?: string
  source_type?: string
  page_count: number
}

interface ServerCollectionsResponse {
  collections: ServerLibraryCollection[]
  count: number
  enabled: boolean
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
    queryFn: async () => {
      const data = await apiRequest<ServerCollectionsResponse>('/library')
      return {
        enabled: data.enabled,
        count: data.count,
        collections: data.collections.map((collection) => ({
          id: collection.id,
          name: collection.name,
          description: collection.source_type
            ? `${collection.source_type}: ${collection.source || ''}`
            : undefined,
          page_count: collection.page_count,
          item_count: collection.page_count,
        })),
      } satisfies CollectionsResponse
    },
  })
}

/**
 * Hook to fetch items in a collection
 */
export function useLibraryItems(collectionId?: string) {
  return useQuery({
    queryKey: ['library', 'collections', collectionId, 'items'],
    queryFn: () =>
      apiRequest<LibraryItemsResponse>(`/library/${collectionId}/items`),
    enabled: !!collectionId,
  })
}
