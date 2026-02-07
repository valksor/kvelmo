import { useState, useCallback, type ReactNode } from 'react'
import { AnnouncerContext } from './useAnnouncer'

type Priority = 'polite' | 'assertive'

export function ScreenReaderAnnouncer({ children }: { children: ReactNode }) {
  const [politeMessage, setPoliteMessage] = useState('')
  const [assertiveMessage, setAssertiveMessage] = useState('')

  const announce = useCallback((message: string, priority: Priority = 'polite') => {
    const setter = priority === 'assertive' ? setAssertiveMessage : setPoliteMessage

    // Clear first to ensure re-announcement even with identical messages
    setter('')
    requestAnimationFrame(() => {
      setter(message)
    })

    // Auto-clear after screen reader has time to announce
    setTimeout(() => setter(''), 1000)
  }, [])

  return (
    <AnnouncerContext.Provider value={{ announce }}>
      {children}
      <div role="status" aria-live="polite" aria-atomic="true" className="sr-only">
        {politeMessage}
      </div>
      <div role="alert" aria-live="assertive" aria-atomic="true" className="sr-only">
        {assertiveMessage}
      </div>
    </AnnouncerContext.Provider>
  )
}
