import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Sun, Moon } from 'lucide-react'

type Theme = 'light' | 'dark'

const STORAGE_KEY = 'mehrhof-theme'
const LIGHT_THEME = 'winter'
const DARK_THEME = 'business'

export function ThemeToggle() {
  const { t } = useTranslation()
  const [theme, setTheme] = useState<Theme>(() => {
    // Check localStorage first
    if (typeof window !== 'undefined') {
      const stored = localStorage.getItem(STORAGE_KEY) as Theme | null
      if (stored === 'light' || stored === 'dark') {
        return stored
      }
      // Fall back to system preference
      if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
        return 'dark'
      }
    }
    return 'light'
  })

  // Apply theme to document
  useEffect(() => {
    const root = document.documentElement
    root.setAttribute('data-theme', theme === 'light' ? LIGHT_THEME : DARK_THEME)
    localStorage.setItem(STORAGE_KEY, theme)
  }, [theme])

  const toggleTheme = () => {
    setTheme((prev) => (prev === 'light' ? 'dark' : 'light'))
  }

  const label = theme === 'light' ? t('theme.switchToDark') : t('theme.switchToLight')

  return (
    <button
      onClick={toggleTheme}
      className="btn btn-ghost btn-sm btn-circle"
      title={label}
      aria-label={label}
    >
      {theme === 'light' ? (
        <Moon size={18} className="text-base-content/70" />
      ) : (
        <Sun size={18} className="text-base-content/70" />
      )}
    </button>
  )
}
