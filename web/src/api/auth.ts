/**
 * Authentication API
 */

import { clearCsrfToken } from './client'

interface LoginResponse {
  success: boolean
  username: string
  role: string
}

interface LoginError {
  error: string
}

/**
 * Login with username and password
 */
export async function login(
  username: string,
  password: string
): Promise<{ success: true; data: LoginResponse } | { success: false; error: string }> {
  try {
    // Get CSRF token first (only needed if auth is enabled)
    // In localhost mode, CSRF endpoint may not be available
    let csrfToken = ''
    try {
      const csrfRes = await fetch('/api/v1/auth/csrf', { credentials: 'include' })
      if (csrfRes.ok) {
        const data = (await csrfRes.json()) as { csrf_token: string }
        csrfToken = data.csrf_token
      }
    } catch {
      // CSRF not required in localhost mode
    }

    // Attempt login
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    }
    if (csrfToken) {
      headers['X-Csrf-Token'] = csrfToken
    }

    const res = await fetch('/api/v1/auth/login', {
      method: 'POST',
      credentials: 'include',
      headers,
      body: JSON.stringify({ username, password }),
    })

    if (!res.ok) {
      const data = (await res.json()) as LoginError
      return { success: false, error: data.error || 'Invalid credentials' }
    }

    const data = (await res.json()) as LoginResponse
    return { success: true, data }
  } catch {
    return { success: false, error: 'Network error. Please try again.' }
  }
}

/**
 * Logout current user
 */
export async function logout(): Promise<void> {
  try {
    await fetch('/api/v1/auth/logout', {
      method: 'POST',
      credentials: 'include',
    })
  } catch {
    // Ignore errors - we want to clear local state regardless
  }
  clearCsrfToken()
  window.location.href = '/login'
}
