import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { storeName } from '../meta'

type Theme = 'light' | 'dark'

const THEME_MAP = {
  light: 'corporate',
  dark: 'business',
} as const

interface ThemeState {
  theme: Theme
  setTheme: (theme: Theme) => void
  toggle: () => void
}

const applyTheme = (theme: Theme) => {
  document.documentElement.setAttribute('data-theme', THEME_MAP[theme])
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set, get) => ({
      theme: 'light',
      setTheme: (theme: Theme) => {
        set({ theme })
        applyTheme(theme)
      },
      toggle: () => {
        const newTheme = get().theme === 'light' ? 'dark' : 'light'
        set({ theme: newTheme })
        applyTheme(newTheme)
      },
    }),
    {
      name: storeName('theme'),
      onRehydrateStorage: () => (state) => {
        if (state) {
          applyTheme(state.theme)
        }
      },
    }
  )
)
