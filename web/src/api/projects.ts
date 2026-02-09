import { useQuery, useMutation } from '@tanstack/react-query'
import { apiRequest } from './client'

export interface Project {
  id: string
  name: string
  path: string
  remote_url?: string
  last_access: string
  is_favorite?: boolean
}

interface ProjectsResponse {
  count: number
  projects: Project[]
  favorites?: string[]
}

/**
 * Hook to list all auto-tracked projects (global mode only)
 * @param options - Query options, especially `enabled` to control fetching
 */
export function useProjects(options?: { enabled?: boolean }) {
  return useQuery<ProjectsResponse>({
    queryKey: ['projects'],
    queryFn: () => apiRequest('/projects'),
    enabled: options?.enabled ?? true,
  })
}

/**
 * Hook to switch to a project (global mode)
 */
export function useSwitchProject() {
  return useMutation({
    mutationFn: async (projectPath: string) => {
      return apiRequest('/projects/select', {
        method: 'POST',
        body: JSON.stringify({ path: projectPath }),
      })
    },
    onSuccess: () => {
      // Mode changed - reload to get fresh state
      window.location.href = '/'
    },
  })
}

/**
 * Hook to switch back to global mode (exit current project)
 */
export function useSwitchToGlobal() {
  return useMutation({
    mutationFn: async () => {
      return apiRequest('/projects/switch', {
        method: 'POST',
      })
    },
    onSuccess: () => {
      // Mode changed - reload to get fresh state
      window.location.href = '/'
    },
  })
}

/**
 * Hook to toggle a project's favorite status
 */
export function useToggleFavorite() {
  return useMutation<{ is_favorite: boolean; path: string }, Error, string>({
    mutationFn: async (projectPath: string) => {
      return apiRequest('/projects/favorite', {
        method: 'POST',
        body: JSON.stringify({ path: projectPath }),
      })
    },
  })
}

/**
 * Hook to remove a project from tracking
 */
export function useRemoveProject() {
  return useMutation<{ path: string }, Error, string>({
    mutationFn: async (projectPath: string) => {
      return apiRequest('/projects', {
        method: 'DELETE',
        body: JSON.stringify({ path: projectPath }),
      })
    },
  })
}

/**
 * Hook to add a new project by path
 */
export function useAddProject() {
  return useMutation<{ path: string; name: string }, Error, string>({
    mutationFn: async (projectPath: string) => {
      return apiRequest('/projects', {
        method: 'POST',
        body: JSON.stringify({ path: projectPath }),
      })
    },
  })
}
