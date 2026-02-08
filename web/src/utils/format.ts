/**
 * Formatting utilities for consistent display across the application.
 * Consolidates duplicated formatting logic from multiple components.
 */

/**
 * Formats a token count with K/M suffixes for readability.
 * @param n - The number of tokens
 * @returns Formatted string like "1.2M", "500K", or "123"
 */
export function formatTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return n.toString()
}

/**
 * Formats a cost amount as currency.
 * @param amount - The cost amount
 * @param currency - Currency code (default: 'USD')
 * @returns Formatted currency string like "$1.23"
 */
export function formatCost(amount: number, currency: string = 'USD'): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: currency,
    minimumFractionDigits: 2,
  }).format(amount)
}

/**
 * Formats a cost amount with a simple dollar sign.
 * Use for inline displays where full Intl formatting is overkill.
 * @param amount - The cost amount
 * @returns Formatted string like "$1.23"
 */
export function formatCostSimple(amount: number): string {
  return `$${amount.toFixed(2)}`
}

/**
 * Formats a date as dd.mm.yyyy
 * @param date - Date object or ISO string
 * @returns Formatted string like "08.02.2026"
 */
export function formatDate(date: Date | string): string {
  const d = typeof date === 'string' ? new Date(date) : date
  if (isNaN(d.getTime())) return ''
  const day = d.getDate().toString().padStart(2, '0')
  const month = (d.getMonth() + 1).toString().padStart(2, '0')
  const year = d.getFullYear()
  return `${day}.${month}.${year}`
}

/**
 * Formats a datetime as dd.mm.yyyy hh:mm
 * @param date - Date object or ISO string
 * @returns Formatted string like "08.02.2026 15:04"
 */
export function formatDateTime(date: Date | string): string {
  const d = typeof date === 'string' ? new Date(date) : date
  if (isNaN(d.getTime())) return ''
  const hours = d.getHours().toString().padStart(2, '0')
  const mins = d.getMinutes().toString().padStart(2, '0')
  return `${formatDate(d)} ${hours}:${mins}`
}

/**
 * Formats a datetime with seconds as dd.mm.yyyy hh:mm:ss
 * @param date - Date object or ISO string
 * @returns Formatted string like "08.02.2026 15:04:35"
 */
export function formatTimestamp(date: Date | string): string {
  const d = typeof date === 'string' ? new Date(date) : date
  if (isNaN(d.getTime())) return ''
  const secs = d.getSeconds().toString().padStart(2, '0')
  return `${formatDateTime(d)}:${secs}`
}
