import { describe, expect, it } from 'vitest'
import { extractErrorMessage, parseApiError, sanitizeErrorMessage } from './errors'

describe('sanitizeErrorMessage', () => {
  it('sanitizes known technical messages', () => {
    expect(sanitizeErrorMessage('context deadline exceeded')).toBe('Request timed out')
    expect(sanitizeErrorMessage('invalid JSON: unexpected token')).toBe('Invalid request format')
    expect(sanitizeErrorMessage('unexpected EOF')).toBe('Unexpected end of data')
  })

  it('keeps user-friendly messages unchanged', () => {
    expect(sanitizeErrorMessage('Task title is required')).toBe('Task title is required')
  })
})

describe('extractErrorMessage', () => {
  it('uses structured error message when available', () => {
    const result = extractErrorMessage(
      {
        success: false,
        error: {
          code: 'VALIDATION_ERROR',
          message: 'open /tmp/x: no such file or directory',
        },
      },
      'fallback'
    )

    expect(result).toBe('File not found')
  })

  it('uses simple error message format', () => {
    const result = extractErrorMessage({ error: 'invalid request body: json: bad value' }, 'fallback')
    expect(result).toBe('Invalid request format')
  })

  it('returns fallback for invalid payload', () => {
    expect(extractErrorMessage(null, 'fallback')).toBe('fallback')
    expect(extractErrorMessage('nope', 'fallback')).toBe('fallback')
    expect(extractErrorMessage({ message: 'missing error key' }, 'fallback')).toBe('fallback')
  })
})

describe('parseApiError', () => {
  it('parses JSON body when available', async () => {
    const response = new Response(JSON.stringify({ error: 'context canceled' }), {
      status: 400,
      headers: { 'Content-Type': 'application/json' },
    })

    await expect(parseApiError(response, 'fallback')).resolves.toBe('Request was cancelled')
  })

  it('uses plain text body when short and displayable', async () => {
    const response = {
      json: async () => {
        throw new Error('not json')
      },
      text: async () => 'connection refused',
    } as Response

    await expect(parseApiError(response, 'fallback')).resolves.toBe('Unable to connect to server')
  })

  it('returns fallback for non-displayable text', async () => {
    const response = {
      json: async () => {
        throw new Error('not json')
      },
      text: async () => '<html>error</html>',
    } as Response

    await expect(parseApiError(response, 'fallback')).resolves.toBe('fallback')
  })
})
