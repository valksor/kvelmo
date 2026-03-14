import { useEffect, useState, useCallback } from 'react'
import { useLayoutStore } from '../stores/layoutStore'

export interface Shortcut {
  key: string
  label: string
  description: string
}

export const SHORTCUTS: Shortcut[] = [
  { key: 'Ctrl+Shift+1', label: 'Ctrl+Shift+1', description: 'Switch to first tab' },
  { key: 'Ctrl+Shift+2', label: 'Ctrl+Shift+2', description: 'Switch to second tab' },
  { key: 'Ctrl+Shift+3', label: 'Ctrl+Shift+3', description: 'Switch to third tab' },
  { key: 'Ctrl+Shift+4', label: 'Ctrl+Shift+4', description: 'Switch to fourth tab' },
  { key: 'Ctrl+Shift+5', label: 'Ctrl+Shift+5', description: 'Switch to fifth tab' },
  { key: 'Ctrl+/', label: 'Ctrl+/', description: 'Show keyboard shortcuts' },
]

/**
 * Global keyboard shortcut handler.
 * Returns a boolean state for whether the shortcuts help dialog is open,
 * and a toggle function.
 */
export function useKeyboardShortcuts() {
  const [showHelp, setShowHelp] = useState(false)

  const toggleHelp = useCallback(() => {
    setShowHelp(prev => !prev)
  }, [])

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      // Ignore shortcuts when typing in inputs
      const target = e.target as HTMLElement
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
        // Only allow Ctrl+/ to pass through from inputs
        if (!(e.ctrlKey && e.key === '/')) return
      }

      // Ctrl+Shift+1..5 — switch to tab by index
      if (e.ctrlKey && e.shiftKey && e.key >= '1' && e.key <= '5') {
        e.preventDefault()
        const index = parseInt(e.key) - 1
        const { tabs, setActiveTab } = useLayoutStore.getState()
        if (index < tabs.length) {
          setActiveTab(tabs[index].id)
        }
        return
      }

      // Ctrl+/ — toggle shortcuts help
      if (e.ctrlKey && e.key === '/') {
        e.preventDefault()
        setShowHelp(prev => !prev)
        return
      }
    }

    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [])

  return { showHelp, setShowHelp, toggleHelp }
}
