import { useQuery } from '@tanstack/react-query'
import { apiRequest } from './client'

export interface FSEntry {
  name: string
  type: 'dir'
}

interface FSBrowseResponse {
  path: string
  parent: string
  entries: FSEntry[]
}

/**
 * Hook to browse filesystem directories
 * Used by the folder picker modal in global mode
 * @param path - Directory path to browse (null/undefined = home directory)
 */
export function useFSBrowse(path: string | null) {
  return useQuery<FSBrowseResponse>({
    queryKey: ['fs-browse', path],
    queryFn: () => {
      const url = path ? `/fs/browse?path=${encodeURIComponent(path)}` : '/fs/browse'
      return apiRequest(url)
    },
    // Don't retry on permission errors
    retry: (failureCount, error) => {
      if (error instanceof Error && error.message.includes('permission')) {
        return false
      }
      return failureCount < 2
    },
  })
}
