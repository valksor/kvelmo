import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiRequest } from './client'

/**
 * User-customizable i18n overrides.
 * Supports terminology replacements and direct key overrides.
 */
export interface I18nOverrides {
  /** Find/replace terms applied across all translations */
  terminology: Record<string, string>
  /** Direct key overrides per language: { "en": { "nav.dashboard": "Home" } } */
  keys: Record<string, Record<string, string>>
}

/**
 * Separated overrides by scope for the editor UI.
 */
export interface I18nOverridesByScope {
  global: I18nOverrides
  project: I18nOverrides
}

/**
 * Response from /i18n/keys endpoint.
 */
interface I18nKeysResponse {
  keys: string[]
}

/**
 * Fetch merged i18n overrides (global + project combined).
 * Project overrides take precedence over global.
 * Used by the i18n runtime to apply all active overrides.
 */
export function useI18nOverrides() {
  return useQuery({
    queryKey: ['i18n-overrides'],
    queryFn: () => apiRequest<I18nOverrides>('/i18n/overrides'),
    staleTime: 30_000,
  })
}

/**
 * Fetch overrides separated by scope for the editor UI.
 * Returns both global and project overrides independently.
 */
export function useI18nOverridesByScope() {
  return useQuery({
    queryKey: ['i18n-overrides-by-scope'],
    queryFn: async (): Promise<I18nOverridesByScope> => {
      const [global, project] = await Promise.all([
        apiRequest<I18nOverrides>('/i18n/overrides/global'),
        apiRequest<I18nOverrides>('/i18n/overrides/project'),
      ])
      return { global, project }
    },
    staleTime: 30_000,
  })
}

/**
 * Fetch available translation keys for the key override editor.
 * Returns a list of commonly overridden keys.
 */
export function useI18nKeys() {
  return useQuery({
    queryKey: ['i18n-keys'],
    queryFn: async () => {
      const response = await apiRequest<I18nKeysResponse>('/i18n/keys')
      return response.keys
    },
    staleTime: Infinity, // Keys list rarely changes
  })
}

/**
 * Save i18n overrides to a specific scope.
 * After successful save, reloads the page to apply changes.
 */
export function useSaveI18nOverrides(scope: 'global' | 'project') {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (overrides: I18nOverrides) =>
      apiRequest<{ status: string; message: string }>(`/i18n/overrides/${scope}`, {
        method: 'POST',
        body: JSON.stringify(overrides),
      }),
    onSuccess: () => {
      // Invalidate both queries so UI stays fresh
      queryClient.invalidateQueries({ queryKey: ['i18n-overrides'] })
      queryClient.invalidateQueries({ queryKey: ['i18n-overrides-by-scope'] })
      // Reload page to apply new translations
      window.location.reload()
    },
  })
}

/**
 * Create empty overrides object.
 */
export function createEmptyOverrides(): I18nOverrides {
  return {
    terminology: {},
    keys: {},
  }
}
