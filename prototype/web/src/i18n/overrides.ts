import type { i18n as I18nInstance } from 'i18next'
import { apiRequest } from '@/api/client'
import type { I18nOverrides } from './types'
import { EMPTY_OVERRIDES } from './types'

/**
 * Fetches user overrides from the backend and applies them to i18next.
 *
 * This function:
 * 1. Fetches merged overrides (global + project) from the API
 * 2. Applies key overrides using i18next's addResource
 * 3. Sets up a post-processor for terminology replacements
 *
 * Overrides are optional - if the API fails, the app continues with defaults.
 */
export async function applyUserOverrides(i18nInstance: I18nInstance): Promise<void> {
  let overrides: I18nOverrides

  try {
    overrides = await apiRequest<I18nOverrides>('/api/v1/i18n/overrides')
  } catch {
    // API not available or returned error - use empty overrides
    overrides = EMPTY_OVERRIDES
  }

  // Apply key overrides per language
  applyKeyOverrides(i18nInstance, overrides.keys)

  // Set up terminology replacements as a post-processor
  applyTerminologyReplacements(i18nInstance, overrides.terminology)
}

/**
 * Applies direct key overrides to i18next resources.
 *
 * Format: { "en": { "nav.dashboard": "Home" } }
 */
function applyKeyOverrides(
  i18nInstance: I18nInstance,
  keys: Record<string, Record<string, string>>
): void {
  for (const [lang, langKeys] of Object.entries(keys)) {
    for (const [keyPath, value] of Object.entries(langKeys)) {
      // Parse namespace and key from path (e.g., "nav.dashboard" or "settings:sections.git.title")
      const colonIndex = keyPath.indexOf(':')
      let namespace: string
      let key: string

      if (colonIndex !== -1) {
        // Explicit namespace: "settings:sections.git.title"
        namespace = keyPath.slice(0, colonIndex)
        key = keyPath.slice(colonIndex + 1)
      } else {
        // Default namespace, full path is the key
        namespace = 'common'
        key = keyPath
      }

      // Add the override resource
      i18nInstance.addResource(lang, namespace, key, value)
    }
  }
}

/**
 * Sets up terminology replacements as an i18next post-processor.
 *
 * This runs after translation lookup and replaces terms case-insensitively.
 * Format: { "Task": "Ticket", "Workflow": "Pipeline" }
 */
function applyTerminologyReplacements(
  i18nInstance: I18nInstance,
  terminology: Record<string, string>
): void {
  const terms = Object.entries(terminology)

  if (terms.length === 0) {
    return
  }

  // Create regex patterns for each term (case-insensitive, word boundaries)
  const patterns = terms.map(([from, to]) => ({
    regex: new RegExp(`\\b${escapeRegex(from)}\\b`, 'gi'),
    replacement: to,
  }))

  // Register post-processor
  i18nInstance.use({
    type: 'postProcessor',
    name: 'terminology',
    process(value: string): string {
      let result = value
      for (const { regex, replacement } of patterns) {
        result = result.replace(regex, replacement)
      }
      return result
    },
  })

  // Enable the post-processor
  i18nInstance.options.postProcess = ['terminology']
}

/**
 * Escapes special regex characters in a string.
 */
function escapeRegex(str: string): string {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
