import { useQuery, useMutation } from '@tanstack/react-query'
import { apiRequest } from './client'

export interface Project {
  id: string
  name: string
  path: string
  remote_url?: string
  last_access: string
}

interface ProjectsResponse {
  count: number
  projects: Project[]
}

/**
 * Hook to list all discovered projects (global mode only)
 * @param enabled - Whether to fetch projects (pass isGlobalMode to avoid 404 in project mode)
 */
export function useProjects(enabled: boolean = true) {
  return useQuery<ProjectsResponse>({
    queryKey: ['projects'],
    queryFn: () => apiRequest('/projects'),
    enabled,
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
