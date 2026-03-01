import { useState, useCallback, useRef, useEffect, type ReactNode } from 'react'
import { AnnouncerContext } from './useAnnouncer'

type Priority = 'polite' | 'assertive'

export function ScreenReaderAnnouncer({ children }: { children: ReactNode }) {
  const [politeMessage, setPoliteMessage] = useState('')
  const [assertiveMessage, setAssertiveMessage] = useState('')

  // Track pending timeouts to prevent race conditions when announce is called rapidly
  const politeTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const assertiveTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Cleanup timeouts on unmount
  useEffect(() => {
    return () => {
      if (politeTimeoutRef.current) clearTimeout(politeTimeoutRef.current)
      if (assertiveTimeoutRef.current) clearTimeout(assertiveTimeoutRef.current)
    }
  }, [])

  const announce = useCallback((message: string, priority: Priority = 'polite') => {
    const setter = priority === 'assertive' ? setAssertiveMessage : setPoliteMessage
    const timeoutRef = priority === 'assertive' ? assertiveTimeoutRef : politeTimeoutRef

    // Cancel any pending clear timeout for this priority level
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current)
      timeoutRef.current = null
    }

    // Clear first so re-announcing the same message still triggers screen readers.
    // setTimeout(fn, 0) defers the set to the next task, same effect as rAF but
    // testable with vi.runAllTimers() without needing to advance animation frames.
    setter('')
    setTimeout(() => setter(message), 0)

    // Schedule clear and track the timeout
    timeoutRef.current = setTimeout(() => {
      setter('')
      timeoutRef.current = null
    }, 1000)
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
