import { useQuery } from '@tanstack/react-query'

interface AuthStatus {
  state: string
  running: boolean
  task_id?: string
}

export function useAuth() {
  const { data, isLoading, error } = useQuery<AuthStatus>({
    queryKey: ['auth', 'status'],
    queryFn: async () => {
      const res = await fetch('/api/v1/status', { credentials: 'include' })
      if (res.status === 401) {
        // Redirect to login (Go template page handles auth)
        window.location.href = `/login?next=${encodeURIComponent(window.location.pathname)}`
        throw new Error('Unauthorized')
      }
      if (!res.ok) {
        throw new Error('Failed to check auth status')
      }
      return res.json()
    },
    retry: false,
    staleTime: 30000, // 30 seconds
  })

  return {
    status: data,
    isLoading,
    isAuthenticated: !!data && !error,
    error,
  }
}
