import { createContext, useContext } from 'react'

type Priority = 'polite' | 'assertive'

export interface AnnouncerContextValue {
  announce: (message: string, priority?: Priority) => void
}

export const AnnouncerContext = createContext<AnnouncerContextValue | null>(null)

export function useAnnouncer() {
  const context = useContext(AnnouncerContext)
  if (!context) {
    throw new Error('useAnnouncer must be used within ScreenReaderAnnouncer')
  }
  return context
}
