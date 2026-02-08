import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import { applyUserOverrides } from './overrides'

// Import type declarations for declaration merging (side-effect import)
import './types'

// Import all translation files
import enCommon from './locales/en/common.json'
import enSettings from './locales/en/settings.json'
import enWorkflow from './locales/en/workflow.json'

// Storage key for language preference (mirrors ThemeToggle pattern)
export const STORAGE_KEY = 'mehrhof-language'

// Supported languages - add 'lv' when Latvian translations are ready
export const SUPPORTED_LANGUAGES = ['en'] as const
export type SupportedLanguage = (typeof SUPPORTED_LANGUAGES)[number]

// Language display names (ready for future languages)
export const LANGUAGE_LABELS: Record<string, string> = {
  en: 'English',
  lv: 'Latviešu',
}

// Namespaces
export const NAMESPACES = ['common', 'settings', 'workflow'] as const
export type Namespace = (typeof NAMESPACES)[number]

// Initialize i18next
i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      en: {
        common: enCommon,
        settings: enSettings,
        workflow: enWorkflow,
      },
      // Add more languages here:
      // lv: { common: lvCommon, settings: lvSettings, workflow: lvWorkflow },
    },
    supportedLngs: [...SUPPORTED_LANGUAGES],
    fallbackLng: 'en',
    defaultNS: 'common',
    ns: [...NAMESPACES],
    interpolation: {
      escapeValue: false, // React already escapes
    },
    detection: {
      order: ['localStorage', 'navigator'],
      lookupLocalStorage: STORAGE_KEY,
      caches: ['localStorage'],
    },
    react: {
      useSuspense: false, // Disable suspense for simpler initial implementation
    },
    // Development mode: log missing translation keys to help identify untranslated strings
    debug: import.meta.env.DEV,
    saveMissing: import.meta.env.DEV,
    missingKeyHandler: (_lngs, ns, key) => {
      if (import.meta.env.DEV) {
        console.warn(`[i18n] Missing translation key: ${ns}:${key}`)
      }
    },
  })
  .then(() => {
    // Apply user overrides after initialization
    applyUserOverrides(i18n).catch(() => {
      // Overrides are optional - silently continue with defaults
    })
  })

export default i18n
