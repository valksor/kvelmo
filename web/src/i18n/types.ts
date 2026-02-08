/**
 * TypeScript declaration merging for type-safe translation keys.
 *
 * This enables:
 * - Autocomplete for translation keys in IDE
 * - Compile-time errors for missing/typo keys
 * - Type inference for interpolation parameters
 */

import type enCommon from './locales/en/common.json'
import type enSettings from './locales/en/settings.json'
import type enWorkflow from './locales/en/workflow.json'

declare module 'i18next' {
  interface CustomTypeOptions {
    defaultNS: 'common'
    resources: {
      common: typeof enCommon
      settings: typeof enSettings
      workflow: typeof enWorkflow
    }
  }
}

/**
 * User override types for terminology replacements and key overrides.
 */
export interface I18nOverrides {
  /** Find/replace terminology across all translations */
  terminology: Record<string, string>
  /** Direct key overrides per language */
  keys: Record<string, Record<string, string>>
}

/**
 * Empty overrides for initialization.
 */
export const EMPTY_OVERRIDES: I18nOverrides = {
  terminology: {},
  keys: {},
}
