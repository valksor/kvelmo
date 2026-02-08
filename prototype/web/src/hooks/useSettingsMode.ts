import { useState, useEffect, useCallback } from 'react'

type SettingsMode = 'simple' | 'advanced'
const STORAGE_KEY = 'mehrhof-settings-mode'

/**
 * Hook for managing Settings page mode (simple vs advanced).
 * Persists preference to localStorage.
 *
 * Simple mode: Shows only essential settings (agent, budget, basic git)
 * Advanced mode: Shows all settings (current behavior)
 */
export function useSettingsMode() {
  const [mode, setModeState] = useState<SettingsMode>(() => {
    if (typeof window !== 'undefined') {
      const stored = localStorage.getItem(STORAGE_KEY)
      if (stored === 'simple' || stored === 'advanced') {
        return stored
      }
    }
    return 'simple' // Default to simple mode for non-technical users
  })

  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, mode)
  }, [mode])

  const toggleMode = useCallback(() => {
    setModeState((prev) => (prev === 'simple' ? 'advanced' : 'simple'))
  }, [])

  const setMode = useCallback((newMode: SettingsMode) => {
    setModeState(newMode)
  }, [])

  return {
    mode,
    toggleMode,
    setMode,
    isSimple: mode === 'simple',
    isAdvanced: mode === 'advanced',
  }
}
