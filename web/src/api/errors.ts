/**
 * API error handling utilities
 *
 * Extracts user-friendly messages from API error responses and sanitizes
 * technical error details that should not be shown to end users.
 */

interface SimpleErrorResponse {
  error: string
}

interface StructuredErrorResponse {
  success: false
  error: {
    code: string
    message: string
    details?: string
  }
}

type ErrorResponse = SimpleErrorResponse | StructuredErrorResponse

/**
 * Technical error patterns that should be sanitized for end users.
 * Maps regex patterns to user-friendly replacements.
 */
const ERROR_SANITIZERS: Array<[RegExp, string]> = [
  // JSON parsing errors
  [/^invalid request body: json:.*$/i, 'Invalid request format'],
  [/^invalid JSON:.*$/i, 'Invalid request format'],
  [/^json:.*unmarshal.*$/i, 'Invalid request format'],

  // Network/connection errors
  [/^context deadline exceeded$/i, 'Request timed out'],
  [/^context canceled$/i, 'Request was cancelled'],
  [/^connection refused$/i, 'Unable to connect to server'],

  // File/path errors
  [/^open .*: no such file or directory$/i, 'File not found'],
  [/^stat .*: no such file or directory$/i, 'File not found'],

  // Generic Go errors that leak implementation details
  [/^.*: EOF$/i, 'Unexpected end of data'],
  [/^unexpected EOF$/i, 'Unexpected end of data'],
]

/**
 * Sanitizes technical error messages into user-friendly text.
 * Keeps messages that are already user-friendly.
 */
export function sanitizeErrorMessage(message: string): string {
  for (const [pattern, replacement] of ERROR_SANITIZERS) {
    if (pattern.test(message)) {
      return replacement
    }
  }
  return message
}

/**
 * Extracts user-friendly error message from an API response.
 *
 * Handles both error formats:
 * - Simple: {"error": "message"}
 * - Structured: {"success": false, "error": {"code": "...", "message": "..."}}
 */
export function extractErrorMessage(data: unknown, fallback: string): string {
  if (!data || typeof data !== 'object') {
    return fallback
  }

  const response = data as ErrorResponse

  // Structured format: {"success": false, "error": {"message": "..."}}
  if ('error' in response && typeof response.error === 'object' && response.error !== null) {
    const structured = response as StructuredErrorResponse
    if (structured.error.message) {
      return sanitizeErrorMessage(structured.error.message)
    }
  }

  // Simple format: {"error": "message"}
  if ('error' in response && typeof response.error === 'string') {
    return sanitizeErrorMessage(response.error)
  }

  return fallback
}

/**
 * Parses API error response body and extracts user-friendly message.
 * Returns fallback message if parsing fails.
 */
export async function parseApiError(response: Response, fallback: string): Promise<string> {
  try {
    const data = await response.json()
    return extractErrorMessage(data, fallback)
  } catch {
    // Response wasn't JSON
    try {
      const text = await response.text()
      if (text && text.length < 200 && !text.startsWith('{') && !text.startsWith('<')) {
        // Plain text error, short enough to display
        return sanitizeErrorMessage(text)
      }
    } catch {
      // Ignore read errors
    }
    return fallback
  }
}
