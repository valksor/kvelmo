import { beforeEach, describe, expect, it, vi } from 'vitest'
import { login, logout } from './auth'

const getCsrfTokenMock = vi.fn()
const clearCsrfTokenMock = vi.fn()

vi.mock('./client', () => ({
  getCsrfToken: () => getCsrfTokenMock(),
  clearCsrfToken: () => clearCsrfTokenMock(),
}))

describe('auth api', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getCsrfTokenMock.mockResolvedValue('logout-csrf-token')
  })

  it('login fetches csrf and sends login request with csrf header', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ csrf_token: 'login-csrf-token' }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ success: true, username: 'alice', role: 'admin' }),
      })

    vi.stubGlobal('fetch', fetchMock)

    const result = await login('alice', 'secret')

    expect(result).toEqual({
      success: true,
      data: { success: true, username: 'alice', role: 'admin' },
    })

    expect(fetchMock).toHaveBeenNthCalledWith(1, '/api/v1/auth/csrf', { credentials: 'include' })
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      '/api/v1/auth/login',
      expect.objectContaining({
        method: 'POST',
        credentials: 'include',
        headers: expect.objectContaining({
          'Content-Type': 'application/json',
          'X-Csrf-Token': 'login-csrf-token',
        }),
      })
    )
  })

  it('login returns friendly error on failed credentials', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: async () => ({ csrf_token: 'token' }),
        })
        .mockResolvedValueOnce({
          ok: false,
          status: 401,
          json: async () => ({ error: 'context deadline exceeded' }),
        })
    )

    await expect(login('alice', 'bad-password')).resolves.toEqual({
      success: false,
      error: 'Request timed out',
    })
  })

  it('logout posts logout request, clears csrf cache, and redirects', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 200 })
    vi.stubGlobal('fetch', fetchMock)

    await logout()

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/auth/logout', {
      method: 'POST',
      credentials: 'include',
      headers: { 'X-Csrf-Token': 'logout-csrf-token' },
    })
    expect(clearCsrfTokenMock).toHaveBeenCalled()
  })
})
