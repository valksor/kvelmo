/**
 * API client with CSRF token handling and auth redirect
 *
 * In localhost mode (no auth enabled), CSRF tokens are not required.
 * The client gracefully handles both authenticated and unauthenticated modes.
 */

let csrfToken: string | null = null
let csrfAttempted = false // Track if we've tried to get CSRF token

async function refreshCsrfToken(): Promise<string | null> {
  try {
    const res = await fetch('/api/v1/auth/csrf', { credentials: 'include' })
    if (res.status === 401) {
      // No session - redirect to login (only in authenticated mode)
      window.location.href = `/login?next=${encodeURIComponent(window.location.pathname)}`
      throw new Error('Session expired')
    }
    if (!res.ok) {
      // CSRF not available - likely localhost mode
      csrfAttempted = true
      return null
    }
    const data = (await res.json()) as { csrf_token: string }
    csrfToken = data.csrf_token
    csrfAttempted = true
    return data.csrf_token
  } catch (err) {
    csrfAttempted = true
    // Re-throw auth errors, swallow network errors (localhost mode)
    if (err instanceof Error && err.message === 'Session expired') throw err
    return null
  }
}

export async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  // Only try to get CSRF token once
  if (!csrfAttempted) {
    await refreshCsrfToken()
  }

  const makeRequest = async () => {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string>),
    }
    if (csrfToken) {
      headers['X-Csrf-Token'] = csrfToken
    }

    return fetch(`/api/v1${endpoint}`, {
      ...options,
      credentials: 'include',
      headers,
    })
  }

  let res = await makeRequest()

  // Retry once on 403 (CSRF token may have expired)
  if (res.status === 403 && csrfToken) {
    csrfAttempted = false
    await refreshCsrfToken()
    res = await makeRequest()
  }

  // Redirect to login on 401
  if (res.status === 401) {
    window.location.href = `/login?next=${encodeURIComponent(window.location.pathname)}`
    throw new Error('Unauthorized')
  }

  if (!res.ok) {
    const error = await res.text()
    throw new Error(error || `API error: ${res.status}`)
  }

  return res.json()
}

/**
 * Clear cached CSRF token (useful on logout)
 */
export function clearCsrfToken() {
  csrfToken = null
}
