/**
 * Authentication API
 */

import { clearCsrfToken, getCsrfToken } from './client'
import { extractErrorMessage } from './errors'

interface LoginResponse {
  success: boolean
  username: string
  role: string
}

/**
 * Login with username and password
 */
export async function login(
  username: string,
  password: string
): Promise<{ success: true; data: LoginResponse } | { success: false; error: string }> {
  try {
    // Get CSRF token first (always required, even in localhost mode)
    // Server uses double-submit cookie pattern in localhost mode
    let csrfToken = ''
    try {
      const csrfRes = await fetch('/api/v1/auth/csrf', { credentials: 'include' })
      if (csrfRes.ok) {
        const data = (await csrfRes.json()) as { csrf_token: string }
        csrfToken = data.csrf_token
      }
    } catch {
      // Network error fetching CSRF - proceed anyway, server will reject if needed
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
      const data = await res.json()
      return { success: false, error: extractErrorMessage(data, 'Invalid credentials') }
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
    const csrfToken = await getCsrfToken()
    const headers: Record<string, string> = {}
    if (csrfToken) {
      headers['X-Csrf-Token'] = csrfToken
    }
    await fetch('/api/v1/auth/logout', {
      method: 'POST',
      credentials: 'include',
      headers,
    })
  } catch {
    // Ignore errors - we want to clear local state regardless
  }
  clearCsrfToken()
  window.location.href = '/login'
}
