/**
 * Calculate exponential backoff delay with jitter for reconnection attempts.
 *
 * @param attempt - The current attempt number (1-based)
 * @param maxDelay - Maximum delay in ms (default 30000)
 * @returns delay in ms with ±20% jitter
 */
export function reconnectDelay(attempt: number, maxDelay = 30000): number {
  const base = Math.min(1000 * Math.pow(2, attempt - 1), maxDelay)
  return Math.round(base * (0.8 + Math.random() * 0.4))
}
