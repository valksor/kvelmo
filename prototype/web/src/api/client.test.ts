import { beforeEach, describe, expect, it, vi } from 'vitest'

describe('apiRequest', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
  })

  it('fetches CSRF token once and includes it in request headers', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ csrf_token: 'csrf-1' }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ success: true }),
      })

    vi.stubGlobal('fetch', fetchMock)

    const { apiRequest } = await import('./client')
    const result = await apiRequest<{ success: boolean }>('/status')

    expect(result.success).toBe(true)
    expect(fetchMock).toHaveBeenNthCalledWith(1, '/api/v1/auth/csrf', { credentials: 'include' })
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      '/api/v1/status',
      expect.objectContaining({
        credentials: 'include',
        headers: expect.objectContaining({
          'Content-Type': 'application/json',
          'X-Csrf-Token': 'csrf-1',
        }),
      })
    )
  })

  it('retries once on 403 after refreshing CSRF token', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ csrf_token: 'old-token' }),
      })
      .mockResolvedValueOnce({
        ok: false,
        status: 403,
        json: async () => ({ error: 'forbidden' }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ csrf_token: 'new-token' }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ success: true }),
      })

    vi.stubGlobal('fetch', fetchMock)

    const { apiRequest } = await import('./client')
    const result = await apiRequest<{ success: boolean }>('/task')

    expect(result.success).toBe(true)
    expect(fetchMock).toHaveBeenCalledTimes(4)

    const finalRequestHeaders = fetchMock.mock.calls[3]?.[1]
    expect(finalRequestHeaders).toEqual(
      expect.objectContaining({
        headers: expect.objectContaining({
          'X-Csrf-Token': 'new-token',
        }),
      })
    )
  })

  it('extracts structured JSON error messages', async () => {
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
          status: 400,
          json: async () => ({
            success: false,
            error: { code: 'X', message: 'context deadline exceeded' },
          }),
        })
    )

    const { apiRequest } = await import('./client')

    await expect(apiRequest('/settings')).rejects.toThrow('Request timed out')
  })

  it('supports text response type', async () => {
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
          ok: true,
          status: 200,
          text: async () => 'plain-text-response',
        })
    )

    const { apiRequest } = await import('./client')
    await expect(apiRequest<string>('/workflow/diagram', {}, 'text')).resolves.toBe('plain-text-response')
  })

  it('getCsrfToken returns null when csrf endpoint fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
      })
    )

    const { getCsrfToken } = await import('./client')
    await expect(getCsrfToken()).resolves.toBeNull()
  })
})
