/**
 * API client with CSRF token handling and auth redirect
 *
 * CSRF tokens are ALWAYS required (even in localhost mode).
 * In localhost mode, the server uses double-submit cookie pattern.
 * In auth mode, the server validates against session-bound tokens.
 */

import { extractErrorMessage, sanitizeErrorMessage } from './errors'

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
      // CSRF endpoint failed - server may be unavailable
      csrfAttempted = true
      return null
    }
    const data = (await res.json()) as { csrf_token: string }
    csrfToken = data.csrf_token
    csrfAttempted = true
    return data.csrf_token
  } catch (err) {
    csrfAttempted = true
    // Re-throw auth errors, swallow network errors
    if (err instanceof Error && err.message === 'Session expired') throw err
    return null
  }
}

export type ResponseType = 'json' | 'blob' | 'text'

export async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {},
  responseType: ResponseType = 'json'
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
    const fallback = `Request failed (${res.status})`
    let message = fallback
    try {
      const data = await res.json()
      message = extractErrorMessage(data, fallback)
    } catch {
      // Response wasn't JSON, try plain text
      try {
        const text = await res.text()
        if (text && text.length < 200 && !text.startsWith('{') && !text.startsWith('<')) {
          message = sanitizeErrorMessage(text)
        }
      } catch {
        // Ignore read errors
      }
    }
    throw new Error(message)
  }

  // Handle different response types
  switch (responseType) {
    case 'blob':
      return res.blob() as Promise<T>
    case 'text':
      return res.text() as Promise<T>
    default:
      return res.json()
  }
}

/**
 * Clear cached CSRF token (useful on logout)
 */
export function clearCsrfToken() {
  csrfToken = null
}

/**
 * Get the current CSRF token, fetching if needed.
 * Use this for non-JSON requests (file uploads) that can't use apiRequest.
 */
export async function getCsrfToken(): Promise<string | null> {
  if (!csrfAttempted) {
    await refreshCsrfToken()
  }
  return csrfToken
}
